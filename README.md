# ibm-storage-odf-block-driver
The ibm-storage-odf-block-driver is a Prometheus Exporter to expose IBM storage information such as health, capacity and performance data as Prometheus metrics. It uses storage Rest API to fetch data and exports as metrics.

It is managed by [IBM storage odf operator](https://github.com/IBM/ibm-storage-odf-operator). Currently it only supports IBM FlashSystem.

The Exporter endpoint is POD:9100/metrics.

## Build image
1. Update the IMAGE_REPO,NAME_SPACE,DRIVER_IMAGE_VERSION in Makefile to setup the image repository. 
2. Run `make push-image` to build and publish image to your specified repository.

## Deploy
It is deployed by [IBM storage odf operator](https://github.com/IBM/ibm-storage-odf-operator).