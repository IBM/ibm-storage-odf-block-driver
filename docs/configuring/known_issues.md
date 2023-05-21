# Known issues

###  Storage system status stuck on 'Progressing'
##### Problem: 
in some circumstances, after adding Flash system as external storage, some Flash storage system might get stuck on progressing state. 
##### Detected in version: 
ODF 4.13 using ODF-FS 1.4.0 
##### Problem Verification: 
On Openshift Console go to Storage -> Data Foundation -> storage systems. Some storage systems might be stuck forever with a status of: "Progressing" and never changes to "Available"

![Storage-system-in-progressing](https://github.com/IBM/ibm-storage-odf-block-driver/blob/9e39da48732a08545ff44debc8db58d3366ddc48/docs/configuring/storage-system-in-progressing.png "storage-system")
##### Workaround:
1. SSH into OCP cluster
2. Switch to openshift-storage namespace by running:  <br>
$ oc project openshift-storage
3. List all pods in namespace by running:  <br>
$ oc get pods  
Look for a pod with prefix: odf-operator-controller-manager*
4. Delete the pod by running:  <br>
$ oc delete pod <df-operator-controller-manager-pod-name>  
5. The pod will be recreated automatically, verify pod creation by running:  <br>
$ oc get pods  
6. storage system status should change to available after a while.


##### Links:
https://bugzilla.redhat.com/show_bug.cgi?id=2207619 <br>
https://jira.xiv.ibm.com/browse/ODF-448  

