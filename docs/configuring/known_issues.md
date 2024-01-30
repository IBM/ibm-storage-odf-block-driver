# Known issues

###  Storage system status stuck on 'Progressing'
##### Problem: 
In some specific circumstances, after adding FlashSystem as external storage, some Flash storage systems might get stuck on 'Progressing' state due to a status sync delay on RedHat ODF operator.
##### Detected in version: 
ODF 4.13 using ODF-FS 1.4.x
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


### ODF-FS Console pod failing due to OOMKilled failure
##### Problem:
The ODF-FS Console pod fails continuously in a CrashLoopBackOff. The ODF-FS operator reports the failure reason is due to OOMKilled (error 137).

##### Detected in version:
All ODF versions running ODF-FS

##### Problem verification:
1. The ODF-FS Console pod failure reason can be extracted with the 'oc describe' command on the ODF-FS operator pod. Failure reason will be OOMKilled (error 137).
2. Any attempt to delete the ODF-FS Console pod fails in the same way.
3. The ODF-FS Console pod's log itself does not show any meaningfull information as the pod cannot start.

##### Workaround:
1. SSH into OCP cluster
2. Switch to openshift-storage namespace by running:  <br>
$ oc project openshift-storage
3. Download and edit the ODF-FS operator subscription by running:  <br>
$ oc edit subscription -n openshift-storage ibm-storage-odf-operator
4. Add the following lines in the <b>spec</b> section. Please note the indention, and that all should be <b>lower case</b>  <br>
<pre>
config:
    resources:
        limits:
            cpu:     50m
            memory:  1000Mi
        requests:
            cpu:     50m
            memory:  1000Mi
</pre>
5. Save and close the file. The subscription will be automatically applied to the cluster, and the ODF-FS pods will be redeployed
6. Confirm all the ODF-FS pods are functioning by running: <br>
$ oc get pods


##### Notes:
Changing the ODF-FS operator subscription will update the memory limit for all deployments in the subscription (ODF console, operator and sidecars). <br>
However, changing the ODF-FS operator subscription will be preserved through upgrades, so it won't need to be changed again when upgrading to a future version.

##### Links:
https://jira.xiv.ibm.com/browse/ODF-579
