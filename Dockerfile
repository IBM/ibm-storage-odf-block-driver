# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -o ibm-storage-odf-block-driver ./cmd/manager/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
MAINTAINER IBM Storage
LABEL vendor="IBM" \
  name="ibm-storage-odf-block-driver" \
  org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm storage odf driver" \  
  org.label-schema.vcs-url="https://github.com/IBM/ibm-storage-odf-block-driver" \
  org.label-schema.schema-version="1.0.0"

WORKDIR /

COPY --from=builder /workspace/ibm-storage-odf-block-driver .

ENTRYPOINT ["/ibm-storage-odf-block-driver"]
