# Known issues

###  Storage system status stuck on 'Progressing'
##### Problem: 
In some specific circumstances, after adding FlashSystem as external storage, some Flash storage systems might get stuck on 'Progressing' state due to a status sync delay on RedHat ODF operator.
##### Detected in version: 
ODF 4.13 using ODF-FS 1.4.0 
##### Problem verification: 
On Openshift Console go to Storage -> Data Foundation -> storage systems. Some storage systems might be stuck forever with a status of: "Progressing" and never changes to "Available".

![Storage-system-in-progressing-github3](storage-system-in-progressing2.png "storage-system")
##### Workaround:
1. SSH into OCP cluster
2. Switch to openshift-storage namespace by running:  <br>
$ oc project openshift-storage
3. Find odf-controller-manager pod by running:  <br>
$ oc get pods | grep odf-operator-controller-manager
4. Delete the pod found in the previous step by running:  <br>
$ oc delete pod {odf-operator-controller-manager-*}  
5. The pod will be recreated automatically, verify pod creation by running:  <br>
$ oc get pods | grep odf-operator-controller-manager  
6. Storage system status should change to 'Available' after a few minutes


##### Links:
https://bugzilla.redhat.com/show_bug.cgi?id=2207619 <br>
https://jira.xiv.ibm.com/browse/ODF-448  

