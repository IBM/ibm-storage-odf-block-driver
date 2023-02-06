# Limitations

Refer to these limitations before working with the ODF FlashSystem driver.

The ODF FlashSystem driver contains the following limitations:

- Only x86 architecture is supported.
- When creating a storage class from within the Red Hat® ODF user-interface, installation of Ceph® RWO on FlashSystem storage systems is allowed. When working with FlashSystem storage systems, it is best to use a direct I/O path to the storage system. For more information, see [Configuration considerations](configuring.md#odf_config).
- The following individual reports are not currently generated:
    - Pool performance
    - Volume performance
    - Volume storage class capacity
- Reports are not generated for FlashSystem information and events.

