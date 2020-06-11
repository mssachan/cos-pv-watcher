## Steps to deploy Persistent Volume watcher for updating firewall rules

**1. Build Watcher image**<br>
**Note:** Set GOPATH properly.
```
mkdir -p $GOPATH/src/github.com/IBM/
git clone https://github.com/mssachan/cos-pv-watcher.git
cd cos-pv-watcher
go mod vendor
docker build -t watcher-builder --pull -f images/watcher/Dockerfile.builder .
docker run watcher-builder /bin/true
docker cp `docker ps -q -n=1`:/root/ca-certs.tar.gz ./
docker cp `docker ps -q -n=1`:/root/watcher.tar.gz ./
docker build \
     --build-arg git_commit_id=${GIT_COMMIT_SHA} \
     --build-arg git_remote_url=${GIT_REMOTE_URL} \
     --build-arg build_date=${BUILD_DATE} \
     -t <image>:<tag> -f ./images/watcher/Dockerfile .
rm -f watcher.tar.gz
rm -f ca-certs.tar.gz
```

**2. Push watcher image to registry**
```
docker push <image>:<tag>
```

**3. Deploy watcher pod**
```
kubectl apply -f deploy/watcher-sa.yaml
kubectl apply -f deploy/watcher.yaml
```

**4. Verification**
```
kubectl get pods -n kube-system | grep ibmcloud-object-storage-plugin-pv-watcher
kubectl logs -n kube-system <watcher_pod_name>
```

**Sample watcher logs:**
```
$ kubectl logs -n kube-system ibmcloud-object-storage-plugin-pv-watcher-559c55b48d-ndg84
{"level":"info","ts":"2020-06-11T15:44:23.529Z","caller":"watcher/main.go:42","msg":"Failed to set flag:","error":"no such flag -logtostderr"}
W0611 15:44:23.530021       1 client_config.go:549] Neither --kubeconfig nor --master was specified.  Using the inClusterConfig.  This might not work.
{"level":"info","ts":"2020-06-11T15:44:23.531Z","caller":"config/config.go:103","msg":"Entry SetUpEvn"}
{"level":"info","ts":"2020-06-11T15:44:23.567Z","caller":"config/config.go:111","msg":"Exit SetUpEvn"}
{"level":"info","ts":"2020-06-11T15:44:23.570Z","caller":"watcher/watcher.go:114","msg":"WatchPersistentVolume"}
{"level":"info","ts":"2020-06-11T15:44:25.554Z","caller":"watcher/set_firewall_rules.go:41","msg":"UpdateFirewallRules","response":{"StatusCode":200,"Headers":{"Date":["Thu, 11 Jun 2020 15:44:24 GMT"],"Etag":["fb18ea33-946d-4821-9ebe-5457edfd6d5f"],"Ibm-Cos-Config-Api-Ver":["1.0"],"Ibm-Cos-Request-Id":["96fd24e6-21c5-47fc-94ca-bab46ba35ac1"]},"Result":null,"RawResult":null}}
{"level":"info","ts":"2020-06-11T15:44:25.555Z","caller":"watcher/watcher.go:150","msg":"Firewall rules for persistent volume updated successfully"}
{"level":"info","ts":"2020-06-11T15:44:25.576Z","caller":"watcher/watcher.go:158","msg":"Annotations updated successfully","for PV":"pvc-2a83ab2b-0ace-4270-b533-007a1460bfe9"}
```

