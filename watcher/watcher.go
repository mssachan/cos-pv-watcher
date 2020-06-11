/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Container Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package watcher

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var lgr zap.Logger
var clientset kubernetes.Interface

type objectStorageCredentials struct {
	// AccessKey is the account identifier in AWS authentication
	AccessKey string
	// SecretKey is the "password" in AWS authentication
	SecretKey string
	// APIKey is the "password" in IBM IAM authentication
	APIKey string
	// ServiceInstanceID is the account identifier in IBM IAM authentication
	ServiceInstanceID string
	// ResConfAPIKey is API key with "Manager" role, used for setting up firewall rules for COS bucket
	ResConfAPIKey string
}

type pvMetadata struct {
	Name        string        `json:"name"`
	Annotations pvAnnotations `json:"annotations,omitempty"`
}

type pvAnnotations struct {
	Bucket               string `json:"ibm.io/bucket"`
	SecretName           string `json:"ibm.io/secret-name"`
	SecretNamespace      string `json:"ibm.io/secret-namespace,omitempty"`
	ProvisionedBy        string `json:"pv.kubernetes.io/provisioned-by"`
	FirewallRulesApplied bool   `json:"ibm.io/firewalls-rules-applied,string"`
}

func parseSecret(secret *v1.Secret, keyName string) (string, error) {
	bytesVal, ok := secret.Data[keyName]
	if !ok {
		return "", fmt.Errorf("%s secret missing", keyName)
	}

	return string(bytesVal), nil
}

func getCredentials(secretName, secretNamespace string) (*objectStorageCredentials, string, error) {
	secrets, err := clientset.Core().Secrets(secretNamespace).Get(secretName, apiv1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("cannot retrieve secret %s: %v", secretName, err)
	}

	var accessKey, secretKey, apiKey, serviceInstanceID string

	apiKey, err = parseSecret(secrets, "api-key")
	if err != nil {
		accessKey, err = parseSecret(secrets, "access-key")
		if err != nil {
			return nil, "", err
		}

		secretKey, err = parseSecret(secrets, "secret-key")
		if err != nil {
			return nil, "", err
		}
	} else {
		serviceInstanceID, err = parseSecret(secrets, "service-instance-id")
	}

	resConfApiKey, _ := secrets.Data["res-conf-apikey"]
	allowedIPs, _ := secrets.Data["allowed_ips"]

	return &objectStorageCredentials{
		AccessKey:         accessKey,
		SecretKey:         secretKey,
		APIKey:            apiKey,
		ServiceInstanceID: serviceInstanceID,
		ResConfAPIKey:     string(resConfApiKey),
	}, string(allowedIPs), nil

}

// WatchPersistentVolumes ...
func WatchPersistentVolumes(client kubernetes.Interface, log zap.Logger) {
	lgr = log
	clientset = client
	watchlist := cache.NewListWatchFromClient(client.Core().RESTClient(), "persistentvolumes", apiv1.NamespaceAll, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &v1.PersistentVolume{}, time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ValidatePersistentVolume,
			DeleteFunc: nil,
			UpdateFunc: nil,
		},
	)
	stopch := wait.NeverStop
	go controller.Run(stopch)
	lgr.Info("WatchPersistentVolume")
	<-stopch
}

func ValidatePersistentVolume(pvObj interface{}) {
	pv, _ := pvObj.(*v1.PersistentVolume)
	var pvmetadata pvMetadata
	jsonBytes, err := json.Marshal(pv.ObjectMeta)
	if err != nil {
		lgr.Error("cannot marshal data", zap.Reflect("error", err))
	}
	err = json.Unmarshal(jsonBytes, &pvmetadata)
	if err != nil {
		lgr.Error("cannot unmarshal data", zap.Reflect("error", err))
	}
	annots := pvmetadata.Annotations
	// lgr.Info("PV metadata Info", zap.String("PV Name", pvmetadata.Name), zap.Reflect("Annotations", annots))

	if strings.Contains(annots.ProvisionedBy, "ibmc-s3fs") && annots.FirewallRulesApplied != true {
		creds, allowedIPs, err := getCredentials(annots.SecretName, annots.SecretNamespace)
		if err != nil {
			lgr.Error(pvmetadata.Name+":cannot get credentials", zap.Reflect("Error", err))
		}
		if allowedIPs != "" {
			if creds.ResConfAPIKey == "" {
				if creds.AccessKey != "" {
					lgr.Error(pvmetadata.Name + ":Firewall rules cannot be set without api key")
				} else if creds.APIKey != "" {
					creds.ResConfAPIKey = creds.APIKey
				}
			}
			err = UpdateFirewallRules(allowedIPs, creds.ResConfAPIKey, annots.Bucket, lgr)
			if err != nil {
				lgr.Error(pvmetadata.Name+":"+"Setting firewall rules failed", zap.String("Bucket", annots.Bucket), zap.Reflect("Error", err))
			} else {
				lgr.Info("Firewall rules for persistent volume updated successfully")
				annots.FirewallRulesApplied = true
				jsonAnnots, _ := json.Marshal(annots)
				patchData := "{\"metadata\": {\"annotations\":" + string(jsonAnnots) + "}}"
				_, errPatch := clientset.Core().PersistentVolumes().Patch(pvmetadata.Name, types.MergePatchType, []byte(patchData))
				if errPatch != nil {
					lgr.Error("Failed to patch annotations", zap.String("for PV", pvmetadata.Name), zap.Error(errPatch))
				} else {
					lgr.Info("Annotations updated successfully", zap.String("for PV", pvmetadata.Name))
				}
			}
		}
	}
}
