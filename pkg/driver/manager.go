/**
 * Copyright contributors to the ibm-storage-odf-block-driver project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package driver

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	log "k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	operatorapi "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	conditionutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

// Reason
const (
	AuthFailure        = "AuthFailure"
	AuthSuccess        = "AuthSuccess"
	VersionCheckFailed = "VersionCheckFailed"
	RoleCheckFailed    = "RoleCheckFailed"
	RestFailure        = "RestFailure"
	ClusterNotOnline   = "ClusterNotOnline"
)

// Message
const (
	AuthFailureMessage     = "Authentication to flash system rest server failed"
	AuthSuccessMessage     = "Authentication to flash system rest server succeed"
	VersionCheckErrMessage = "Flash system code level too low, need >= 8.3.1"
	RoleCheckErrMessage    = "User role need to be Administrator, SecurityAdmin or RestrictedAdmin"
	RestErrorMessage       = "Rest server hit unexpected error"
	ClusterErrMessage      = "Flash system cluster is not online"
	ExporterReadyMessage   = "Flash system exporter is ready"
)

const INIT_POOL_ID = -1

var K8SClient client.Client = nil

type DriverManager struct {
	client.Client
	namespace  string
	SystemName string
	ready      bool
	scPoolMap  map[string]string
	secretName string
}

func NewManager(scheme *runtime.Scheme, namespace string, fscName string, fscScSecretMap operutil.FlashSystemClusterMapContent) (DriverManager, error) {
	var manager DriverManager

	k8sClient, err := getK8sClient(scheme)
	if err != nil {
		log.Errorf("fail to create k8s client, error: %v", err)
		return DriverManager{}, err
	}

	manager.Client = k8sClient
	manager.SystemName = fscName
	manager.namespace = namespace
	manager.scPoolMap = fscScSecretMap.ScPoolMap
	manager.secretName = fscScSecretMap.Secret
	manager.Ready()

	return manager, nil
}

// Add helper function to expose the state for mockup
func (d *DriverManager) Ready() {
	d.ready = true
}

func (d *DriverManager) GetClient() client.Client {
	return d.Client
}

func (d *DriverManager) GetSubsystemName() string {
	// CR Name is the subsystem name
	return d.SystemName
}

func (d *DriverManager) GetNamespaceName() string {
	// CR Name is the subsystem name
	return d.namespace
}

func (d *DriverManager) GetSecretName() string {
	// CR Name is the subsystem name
	return d.secretName
}

func (d *DriverManager) GetSCNameByPoolName(poolName string) []string {
	scNames := []string{}
	for sc, pool := range d.scPoolMap {
		if pool == poolName {
			scNames = append(scNames, sc)
		}
	}

	return scNames
}

func (d *DriverManager) UpdatePoolMap(scPool map[string]string) {
	if !reflect.DeepEqual(scPool, d.scPoolMap) {
		d.scPoolMap = scPool
	}
}

func (d *DriverManager) GetPoolNames() map[string]int {
	poolNames := map[string]int{}
	for _, pool := range d.scPoolMap {
		poolNames[pool] = INIT_POOL_ID
	}

	return poolNames
}

func (d *DriverManager) UpdateCondition(conditionType operatorapi.ConditionType, ready bool, reason string, message string) error {
	k8sclient := d.Client

	fscluster, err := d.GetFlashSystemClusterCR()
	if err != nil {
		log.Errorf("Get flash system CR failed: %v", err)
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
		if operatorapi.ExporterReady == conditionType {
			_ = d.SendK8sEvent(corev1.EventTypeNormal, fmt.Sprintf("%v", conditionType), ExporterReadyMessage)
		}
	} else {
		log.Infof("Set error condition, reason: %s, message: %s", reason, message)
		conditionutil.SetStatusCondition(&fscluster.Status.Conditions, operatorapi.Condition{
			Type:    conditionType,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		})

		_ = d.SendK8sEvent(corev1.EventTypeWarning, reason, message)
	}

	err = k8sclient.Status().Update(context.TODO(), fscluster)
	if err != nil {
		log.Errorf("Fail to update FlashSystemCluster CR, error: %s", err)
		return err
	}

	return nil
}

func (d *DriverManager) SendK8sEvent(eventtype, reason, message string) error {
	fscluster, err := d.GetFlashSystemClusterCR()
	if err != nil {
		log.Errorf("Get flash system CR failed: %v", err)
		return err
	}

	t := metav1.Time{Time: time.Now()}
	evt := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", d.SystemName, t.UnixNano()),
			Namespace: fscluster.Namespace,
			Labels:    conditionutil.GetLabels(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:            fscluster.Kind,
			Namespace:       fscluster.Namespace,
			Name:            fscluster.Name,
			UID:             fscluster.UID,
			ResourceVersion: fscluster.ResourceVersion,
			APIVersion:      fscluster.APIVersion,
		},
		Reason:         reason,
		Message:        message,
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           eventtype,
	}

	err = d.Client.Create(context.TODO(), evt)
	if err != nil {
		log.Errorf("failed to SendK8sEvent reason: %s, message: %s, error: \n %v\n", reason, message, err)
	}
	return err
}

func (d *DriverManager) GetFlashSystemClusterCR() (*operatorapi.FlashSystemCluster, error) {
	fscluster := operatorapi.FlashSystemCluster{}
	err := d.Client.Get(
		context.TODO(),
		client.ObjectKey{
			Namespace: d.namespace,
			Name:      d.SystemName,
		},
		&fscluster,
	)
	if err != nil {
		log.Errorf("Fail to get FlashSystemCluster CR %s, error: %v", d.SystemName, err)
		return nil, err
	}
	return &fscluster, nil
}

func getK8sClient(scheme *runtime.Scheme) (client.Client, error) {
	if K8SClient != nil {
		return K8SClient, nil
	}
	restConfig := config.GetConfigOrDie()
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	K8SClient = k8sClient
	return k8sClient, nil
}
