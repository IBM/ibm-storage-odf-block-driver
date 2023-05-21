# Known issues

###  Storage system status stuck on 'Progressing'
##### Problem: 
in some circumstances, after adding Flash system as external storage, some Flash storage system might get stuck on progressing state. 
##### Detected in version: 
ODF 4.13 using ODF-FS 1.4.0 
##### Problem Verification: 
On Openshift Console go to Storage -> Data Foundation -> storage systems. Some of the storage systems might be stuck forever with a status of: "Progressing" and never changes to "Available"

![Storage-system-in-progressing](https://ibm.box.com/s/41l4pu7letftqoqr6hqir3lzmpjxibfm "storage-system")
##### Workaround:
1. SSH into OCP cluster
2. Switch to openshift-storage namespace by running:  
$ oc project openshift-storage
3. List all pods in namespace by running:  
$ oc get pods  
Look for a pod with prefix: odf-operator-controller-manager*
4. Delete the pod by running:  
$ oc delete pod <df-operator-controller-manager-pod-name>  
5. The pod will be recreated automatically, verify pod creation by running:  
$ oc get pods  
6. storage system status should change to available after a while.


##### Links:
https://bugzilla.redhat.com/show_bug.cgi?id=2207619
https://jira.xiv.ibm.com/browse/ODF-448  

