ARG GO_BUILDER=brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.24
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:7c5495d5fad59aaee12abc3cbbd2b283818ee1e814b00dbc7f25bf2d14fa4f0c

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/openshift-pipelines/tekton-assist
COPY . .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done

ENV GOEXPERIMENT=strictfipsruntime
RUN git rev-parse HEAD > /tmp/HEAD
RUN go build -ldflags="-X 'knative.dev/pkg/changeset.rev=$(cat /tmp/HEAD)'" -mod=vendor -tags disable_gcp,strictfipsruntime -v -o /tmp/tekton-assist \
    ./cmd/diagnose

FROM $RUNTIME
ARG VERSION=tekton-assist-main

COPY --from=builder /tmp/tekton-assist /ko-app/tekton-assist

LABEL \
      com.redhat.component="openshift-pipelines-tekton-assist-rhel8-container" \
      name="openshift-pipelines/tekton-assist-rhel8" \
      version=$VERSION \
      summary="Red Hat OpenShift Pipelines Tekton Assistant" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Tekton Assistant" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Tekton Assistant" \
      io.k8s.description="Red Hat OpenShift Pipelines Tekton Assistant" \
      io.openshift.tags="pipelines,tekton,openshift"

RUN microdnf install -y shadow-utils
RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT ["/ko-app/tekton-assist"]
