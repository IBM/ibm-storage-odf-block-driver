# ibm-storage-odf-block-driver
The ibm-storage-odf-block-driver is a Prometheus Exporter to expose the IBM flashsystem internal data as Prometheus metrics. When the Prometheus server scrape the metrics it will call the backend Flashsystem Rest API to fetch data and convert the metric.

The Exporter exposed endpoint is POD:9100/metrics.

## Build image
1. Update the IMAGE_REPO,NAME_SPACE,DRIVER_IMAGE_VERSION in Makefile to setup the image repository. 
2. Run `make push-image` to build and publish image to your specified repository.

## Deploy
This exporter is deployed by [IBM storage odf operator](https://github.com/IBM/ibm-storage-odf-operator)
