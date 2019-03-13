FROM registry.svc.ci.openshift.org/openshift/release:golang-1.11 AS builder
WORKDIR /go/src/github.com/metalkube/mdns-publisher
COPY . .
RUN go get -v github.com/grandcat/zeroconf
WORKDIR /go/src/github.com/grandcat/zeroconf
RUN git remote add celebdor https://github.com/celebdor/zeroconf
RUN git fetch celebdor
RUN git checkout celebdor/register-svc-entry
RUN go get -v github.com/spf13/viper github.com/spf13/cobra github.com/sirupsen/logrus
WORKDIR /go/src/github.com/metalkube/mdns-publisher
RUN go build

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/metalkube/mdns-publisher/mdns-publisher /usr/bin/

ENTRYPOINT ["/usr/bin/mdns-publisher"]

LABEL io.k8s.display-name="mDNS-publisher" \
      io.k8s.description="Configurable mDNS service publisher" \
      maintainer="Antoni Segura Puimedon <antoni@redhat.com>"
