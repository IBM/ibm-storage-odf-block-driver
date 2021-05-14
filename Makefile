# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DRIVER_NAME=ibm-storage-odf-block-driver

IMAGE_REPO=9.110.70.75/sandbox
DRIVER_IMAGE_VERSION=v0.0.10

DRIVER_IMAGE=$(IMAGE_REPO)/$(DRIVER_NAME)

.PHONY: all $(DRIVER_IMAGE) 

all: $(DRIVER_IMAGE) 

$(DRIVER_IMAGE):
	if [ ! -d ./vendor ]; then dep ensure -v; fi
	CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -ldflags '-extldflags "-static"' -o  ./build/_output/bin/${DRIVER_NAME} ./cmd/manager/main.go

build-image: 
	docker build --network=host -t $(DRIVER_IMAGE):$(DRIVER_IMAGE_VERSION) -f ./Dockerfile .	

push-image: build-image
	docker push $(DRIVER_IMAGE):$(DRIVER_IMAGE_VERSION)

clean: bin-clean

bin-clean:
	rm -rf ./build/_output/bin/*
