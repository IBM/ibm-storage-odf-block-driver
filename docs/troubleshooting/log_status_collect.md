
{{site.data.keyword.attribute-definition-list}}

# Detecting errors and log collection


To collect status and logs related to the different components of the IBM® ODF FlashSystem driver, use this Red Hat Openshift® command:
```
oc adm must-gather --image=quay.io/ibmodffs/ibm-storage-odf-operator-must-gather:1.9.0 -- gather "ibm" "openshift-storage" "openshift-storage"
```
