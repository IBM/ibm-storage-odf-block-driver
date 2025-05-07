# Limitations

Refer to these limitations before working with the IBM® ODF FlashSystem driver.

The IBM® ODF FlashSystem driver contains the following limitations:

- Only x86 architecture is supported.
- When creating a storage class from within the Red Hat OpenShift Data Foundation® user-interface, installation of Ceph® RWO on FlashSystem storage systems is allowed. When working with FlashSystem storage systems, it is best to use a direct I/O path to the storage system. For more information, see [Configuration considerations](configuring.md#odf_config).
- The following individual reports are not currently generated:
    - Pool performance
    - Volume performance
    - Volume storage class capacity
- Reports are not generated for IBM FlashSystem® information and events.
- IBM® ODF FlashSystem driver will not operate correctly if prior to installation an instance of IBM Block Storage CSI driver is installed on the cluster.
