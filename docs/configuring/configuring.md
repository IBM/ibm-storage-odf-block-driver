# Configuring

Red Hat OpenShift Data Foundation® (ODF) uses the IBM® ODF FlashSystem driver to integrate with your IBM Storage Virtualize® family storage system.

When using Red Hat® ODF with your storage system, ensure that all configuration needs are met by following the Red Hat® ODF documentation.

The IBM® ODF FlashSystem driver utilizes the IBM® Block Storage CSI driver for all Red Hat OpenShift® cluster configuration changes, such as StorageClass and PersistentVolumeClaims creation, in use of IBM Storage Virtualize® family storage system products.

For more information and documentation, see the following:

-   Support summary for enabled IBM Storage Virtualize® family storage system products and Red Hat OpenShift Data Foundation®: [IBM® ODF FlashSystem driver support summary](../docs_general/odf_flashsystem_driver_support_matrix.html).
-   User information and release notes documentation for the Red Hat OpenShift Data Foundation®: [Red Hat OpenShift Data Foundation® documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation).
-   General information about Red Hat OpenShift Data Foundation®, a software-defined storage for containers: [Red Hat OpenShift Data Foundation®](https://www.redhat.com/en/technologies/cloud-computing/openshift-data-foundation).
-   IBM® Block Storage CSI driver configuration: [IBM® Block Storage CSI driver](https://www.ibm.com/docs/en/stg-block-csi-driver)

## Configuration considerations

When creating block storage persistent volumes, be sure to select the storage class <storage_class_name> for best performance. The storage class allows a direct I/O path to the IBM Storage Virtualize® family storage system.


