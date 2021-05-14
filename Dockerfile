# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
#COPY api/ api/
COPY vendor/ vendor/
COPY pkg/ pkg/
COPY pkg/collectors/ pkg/collectors/
COPY pkg/prome/ pkg/prome/
COPY pkg/server/ pkg/server/
COPY pkg/driver/ pkg/driver/
COPY pkg/rest/ pkg/rest/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod=vendor -a -o ibm-storage-odf-block-driver ./cmd/manager/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY --from=builder /workspace/ibm-storage-odf-block-driver .
USER nonroot:nonroot

ENTRYPOINT ["/ibm-storage-odf-block-driver"]
