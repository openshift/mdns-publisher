FROM registry.svc.ci.openshift.org/openshift/release:golang-1.11 AS builder
WORKDIR /go/src/github.com/openshift-metalkube/mdns-publisher
COPY . .
RUN GO111MODULE=on go build

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift-metalkube/mdns-publisher/mdns-publisher /usr/bin/

ENTRYPOINT ["/usr/bin/mdns-publisher"]

LABEL io.k8s.display-name="mDNS-publisher" \
      io.k8s.description="Configurable mDNS service publisher" \
      maintainer="Antoni Segura Puimedon <antoni@redhat.com>"
