/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package driver

import (
	"os"
	"context"
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"k8s.io/apimachinery/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	//"k8s.io/client-go/kubernetes/scheme"

	operatorapi "github.ibm.com/PuDong/ibm-storage-odf-operator/api/v1alpha1"
	conditionutil "github.ibm.com/PuDong/ibm-storage-odf-operator/controllers/util"
)

const (
	EnvNamespaceName		= "WATCH_NAMESPACE"
	EnvClusterCRName		= "FLASHSYSTEM_CLUSTERNAME"
)

// Reason
const (
	AuthFailure				= "AuthFailure"
	VersionCheckFailed		= "VersionCheckFailed"
	RoleCheckFailed			= "RoleCheckFailed"
	RestFailure				= "RestFailure"
	ClusterNotOnline 		= "ClusterNotOnline"
)

// Message
const (
	AuthFailureMessage		= "Authentication to flash system rest server failed"
	VersionCheckErrMessage	= "Flash system code level too low, need >= 8.3.1"
	RoleCheckErrMessage		= "User role need to be Monitor"
	RestErrorMessage		= "Rest server hit unexpected error"
	ClusterErrMessage		= "Flash system cluster not online"
)

var CacheManager DriverManager

type DriverManager struct {
	client.Client
	namespace	string
	systemName	string
	ready		bool
	scPoolMap	*map[string]string
}

func NewManager(scheme *runtime.Scheme, scPool *map[string]string) (*DriverManager, error) {

	k8sclient, err := newClient(scheme)
	if err != nil {
		log.Error("fail to create k8s client, error: %s", err)
		return nil, err
	}

	CacheManager.Client = k8sclient

	// Flashsystem Cluster CR
	name, err := getFlashSystemClusterName()
	if err != nil {
		log.Error("Fail to get FlashSystemCluster CR Name, error: %s", err)
		return nil, err
	}

	// Namespace name
	namespace, err := getOperatorNamespace()
	if err != nil {
		log.Error("Fail to get operator namespace Name, error: %s", err)
		return nil, err
	}

	_, err = getFlashSystemClusterCR(k8sclient, name, namespace)
	if err != nil {
		log.Error("Fail to get FlashSystemCluster CR, error: %s", err)
		return nil, err
	}

	CacheManager.systemName = name
	CacheManager.namespace = namespace
	CacheManager.ready = true
	CacheManager.scPoolMap = scPool

	return &CacheManager, nil
}

func GetManager() (*DriverManager, error) {

	if CacheManager.ready {
		return &CacheManager, nil
	}

	return nil, fmt.Errorf("Manager not ready")
}

func (d *DriverManager) GetSubsystemName() string {
	// CR Name is the subsystem name
	return d.systemName
}

func (d *DriverManager) GetNamespaceName() string {
	// CR Name is the subsystem name
	return d.namespace
}

func (d *DriverManager) GetSCNameByPoolName(poolName string) string {
	var scName string
	for sc, pool := range *d.scPoolMap {
		if pool == poolName {
			scName = sc
		}
	}

	return scName
}

func (d *DriverManager) UpdatePoolMap(scPool *map[string]string) {
	if !reflect.DeepEqual(*scPool, *d.scPoolMap) {
		d.scPoolMap = scPool
	}
}

func (d *DriverManager) GetPoolNames() []string {
	pools := []string{}
	for _, pool := range *d.scPoolMap {
		pools = append(pools, pool)
	}

	return pools
}

func (d *DriverManager) UpdateCondition(conditionType operatorapi.ConditionType, ready bool, reason string, message string) error {
	k8sclient := d.Client

	fscluster, err := getFlashSystemClusterCR(k8sclient, d.systemName, d.namespace)
	if err != nil {
		log.Error("Get flash system CR failed: %s", err)
		return err
	}

	// Update status ExporterReady
	if ready {
		conditionutil.SetStatusCondition(&fscluster.Status.Conditions, operatorapi.Condition{
			Type:    conditionType,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: message,
		})
	} else {
		log.Infof("Set error condition, reason: %s, message: %s", reason, message)
		conditionutil.SetStatusCondition(&fscluster.Status.Conditions, operatorapi.Condition{
			Type:    conditionType,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		})
	}

	err = k8sclient.Status().Update(context.TODO(), fscluster)
	if err != nil {
		log.Errorf("Fail to update FlashSystemCluster CR, error: %s", err)
		return err
	}

	return nil
}

func getFlashSystemClusterCR(k8sclient client.Client, name string, namespace string) (*operatorapi.FlashSystemCluster, error) {

	fscluster := operatorapi.FlashSystemCluster{}
	err := k8sclient.Get(
			context.TODO(),
			client.ObjectKey{
				Namespace:	namespace,
				Name:		name,
			},
			&fscluster,
		)
	if err != nil {
		log.Error("Fail to get FlashSystemCluster CR, error: %s", err)
		return nil, err
	}

	return &fscluster, nil
}

func newClient(scheme *runtime.Scheme) (client.Client, error) {
	restConfig := config.GetConfigOrDie()
	k8sclient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return k8sclient, nil
}

func getFlashSystemClusterName() (string, error) {
	if value, ok := os.LookupEnv(EnvClusterCRName); ok {
		return value, nil
	} else {
		return "", fmt.Errorf("Required env variable: '%s' isn't found", EnvClusterCRName)
	}
}

func getOperatorNamespace() (string, error) {
	if value, ok := os.LookupEnv(EnvNamespaceName); ok {
		return value, nil
	} else {
		return "", fmt.Errorf("Required env variable: '%s' isn't found", EnvNamespaceName)
	}
}
