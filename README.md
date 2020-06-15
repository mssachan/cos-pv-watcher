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

**5. Create Secret**
```
vi deploy/sample-usage/secret.yaml
kubectl apply -f deploy/sample-usage/secret.yaml
```

**6. Create PVC**
```
vi deploy/sample-usage/pvc.yaml
kubectl apply -f deploy/sample-usage/pvc.yaml
```

**Sample watcher logs:**
```
$ kubectl logs -n kube-system ibmcloud-object-storage-plugin-pv-watcher-559c55b48d-ndg84
{"level":"info","ts":"2020-06-11T18:40:14.152Z","caller":"watcher/main.go:79","msg":"Failed to set flag:","error":"no such flag -logtostderr"}
W0611 18:40:14.152669       1 client_config.go:549] Neither --kubeconfig nor --master was specified.  Using the inClusterConfig.  This might not work.
{"level":"info","ts":"2020-06-11T18:40:14.197Z","caller":"watcher/watcher.go:117","msg":"WatchPersistentVolume"}
{"level":"info","ts":"2020-06-11T18:49:33.392Z","caller":"watcher/set_firewall_rules.go:41","msg":"UpdateFirewallRules","response":{"StatusCode":200,"Headers":{"Date":["Thu, 11 Jun 2020 18:49:32 GMT"],"Etag":["3dcaaf7e-fd1c-4112-8ecf-40c8d5ada6c2"],"Ibm-Cos-Config-Api-Ver":["1.0"],"Ibm-Cos-Request-Id":["75631e49-cc6e-4f31-8e96-5754a6cba4f5"]},"Result":null,"RawResult":null}}
{"level":"info","ts":"2020-06-11T18:49:33.392Z","caller":"watcher/watcher.go:152","msg":"Firewall rules for persistent volume updated successfully"}
{"level":"info","ts":"2020-06-11T18:49:33.404Z","caller":"watcher/watcher.go:160","msg":"Annotations updated successfully","for PV":"pvc-4e98c16a-f474-437e-b225-d48de0521972"}
```

