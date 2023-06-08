# Configuring

Red Hat® OpenShift® Data Foundation (ODF) uses the ODF FlashSystem driver to integrate with your IBM FlashSystem® storage system. When using the OpenShift Data Foundation with your storage system ensure that all configuration needs are met by following the Red Hat ODF documentation set.

The ODF FlashSystem driver enable IBM Spectrum Virtualize family products to be used with Red Hat OpenShift Data Foundation.

For more information and documentation, see the following:

-   Support summary for enabled IBM Spectrum Virtualize family products and Red Hat OpenShift Data Foundation: [ODF FlashSystem driver support summary](../landing/odf_flashsystem_driver_support_matrix.html).
-   User information and release notes documentation for the Red Hat OpenShift Data Foundation: [OpenShift Data Foundation documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation).
-   General information about the OpenShift Data Foundation, a software-defined storage for containers: [OpenShift Data Foundation](https://www.redhat.com/en/technologies/cloud-computing/openshift-data-foundation).

## Configuration considerations

When creating block storage persistent volumes, be sure to select the storage class <storage_class_name> for best performance. The storage class allows a direct I/O path to the FlashSystem storage system.


