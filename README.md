# ibm-storage-odf-block-driver
ibm-storage-odf-block-driver provides the flashsystem specific driver to connect backend flashsystem storage, pulling ODF required data info to ibm-storage-odf-operator.

## build image
1. Update the IMAGE_REPO,NAME_SPACE,DRIVER_IMAGE_VERSION in Makefile to setup the image repository. 
2. Run `make push-image` to make and publish image.
