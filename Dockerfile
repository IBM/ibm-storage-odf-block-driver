# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ibm-storage-odf-block-driver ./cmd/manager/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
MAINTAINER IBM Storage
LABEL vendor="IBM" \
  name="ibm-storage-odf-block-driver" \
  org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm storage odf driver" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.schema-version="0.2.0"

WORKDIR /

COPY --from=builder /workspace/ibm-storage-odf-block-driver .

ENTRYPOINT ["/ibm-storage-odf-block-driver"]
