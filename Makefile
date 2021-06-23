
LINT_VERSION="1.40.0"

IMAGE_REPO="docker.io"
NAME_SPACE="ibmcom"
DRIVER_IMAGE_VERSION=latest

DRIVER_NAME=ibm-storage-odf-block-driver

DRIVER_IMAGE=$(IMAGE_REPO)/${NAME_SPACE}/$(DRIVER_NAME)

.PHONY: all $(DRIVER_IMAGE) 

all: $(DRIVER_IMAGE) 

$(DRIVER_IMAGE):
	if [ ! -d ./vendor ]; then dep ensure -v; fi
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -ldflags '-extldflags "-static"' -o  ./build/_output/bin/${DRIVER_NAME} ./cmd/manager/main.go

.PHONY: deps
deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint --version)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: deps
	golangci-lint run # Run `make lint-fix` may help to fix lint issues.

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
	docker build -t $(DRIVER_IMAGE):$(DRIVER_IMAGE_VERSION) -f ./Dockerfile .	

push-image: build-image
	docker push $(DRIVER_IMAGE):$(DRIVER_IMAGE_VERSION)

clean: bin-clean

bin-clean:
	rm -rf ./build/_output/bin/*

add-copyright:
	hack/add-copyright.sh

check-copyright:
	hack/check-copyright.sh
	