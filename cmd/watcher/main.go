/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Container Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"

	watcher "github.com/IBM/cos-pv-watcher/watcher"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"
	"strings"
)

var master = flag.String(
	"master",
	"",
	"Master URL to build a client config from. Either this or kubeconfig needs to be set if the watcher is being run out of cluster.",
)
var kubeconfig = flag.String(
	"kubeconfig",
	"",
	"Absolute path to the kubeconfig file. Either this or master needs to be set if the watcher is being run out of cluster.",
)

// ZapLogger is the global logger
var ZapLogger *zap.Logger

// GetZapLogger returns an instance of the logger, initializing a new logger
func GetZapLogger() (*zap.Logger, error) {
	if ZapLogger == nil {
		return NewZapLogger()
	}
	return ZapLogger, nil
}

// NewZapLogger creates and returns a new global logger. It overwrites the
// existing global logger if that has been previously defined.
func NewZapLogger() (*zap.Logger, error) {
	productionConfig := zap.NewProductionConfig()
	productionConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	ZapLogger, _ := productionConfig.Build()
	return ZapLogger, nil
}

func getEnv(key string) string {
	return os.Getenv(strings.ToUpper(key))
}

// GetConfigBool ...
func GetConfigBool(envKey string, defaultConf bool, logger zap.Logger) bool {
	if val := getEnv(envKey); val != "" {
		if envBool, err := strconv.ParseBool(val); err == nil {
			return envBool
		}
		logger.Error("error parsing env val to bool", zap.String("env", envKey))
	}
	return defaultConf
}

func main() {
	var err error
	logger, _ := GetZapLogger()
	loggerLevel := zap.NewAtomicLevel()
	err = flag.Set("logtostderr", "true")
	if err != nil {
		logger.Info("Failed to set flag:", zap.Error(err))
	}
	flag.Parse()

	// Enable debug trace
	debugTrace := GetConfigBool("DEBUG_TRACE", false, *logger)
	if debugTrace {
		loggerLevel.SetLevel(zap.DebugLevel)
	}

	var config *rest.Config
	config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	if err != nil {
		logger.Fatal("Failed to create config:", zap.Error(err))
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("Failed to create client:", zap.Error(err))
	}

	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		logger.Fatal("Error getting server version:", zap.Error(err))
	}

	// Start watcher for persistent volumes
	watcher.WatchPersistentVolumes(clientset, *logger)
}
