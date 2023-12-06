
LINT_VERSION="1.40.0"

REGISTRY=quay.io/ibmodffs
IMAGE_TAG=1.4.1
PLATFORM=linux/amd64,linux/ppc64le,linux/s390x
DRIVER_NAME=ibm-storage-odf-block-driver

DRIVER_IMAGE=$(REGISTRY)/$(DRIVER_NAME):$(IMAGE_TAG)
BUILD_COMMAND = docker buildx build -t $(DRIVER_IMAGE) --platform $(PLATFORM) -f ./Dockerfile .


.PHONY: all $(DRIVER_NAME)

all: $(DRIVER_NAME)

$(DRIVER_NAME):
	if [ ! -d ./vendor ]; then dep ensure -v; fi
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -ldflags '-extldflags "-static"' -o  ./build/_output/bin/${DRIVER_NAME} ./cmd/manager/main.go

.PHONY: deps
deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint --version)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: deps
	golangci-lint run -E gosec --timeout=6m    # Run `make lint-fix` may help to fix lint issues.

.PHONY: lint-fix
lint-fix: deps
	golangci-lint run --fix

.PHONY: build
build:
	go build ./cmd/manager/main.go

.PHONY: test
test:
	go test -race -covermode=atomic -coverprofile=cover.out ./pkg/...

build-image:
	$(BUILD_COMMAND)

push-image:
	$(BUILD_COMMAND) --push

clean: bin-clean

bin-clean:
	rm -rf ./build/_output/bin/*

add-copyright:
	hack/add-copyright.sh

check-copyright:
	hack/check-copyright.sh
