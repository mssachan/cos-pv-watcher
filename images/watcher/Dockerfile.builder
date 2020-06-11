FROM golang:1.13.5
ADD . /go/src/github.com/IBM/cos-pv-watcher
RUN set -ex; cd /go/src/github.com/IBM/cos-pv-watcher/ && CGO_ENABLED=0 go install -v github.com/IBM/cos-pv-watcher/cmd/watcher
RUN set -ex; tar cvC / ./etc/ssl  | gzip -n > /root/ca-certs.tar.gz
RUN set -ex; tar cvC /go/ ./bin | gzip -9 > /root/watcher.tar.gz
