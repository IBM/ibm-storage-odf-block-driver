package driver

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	log "k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	//"k8s.io/client-go/kubernetes/scheme"

	operatorapi "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	conditionutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

const (
	EnvNamespaceName = "WATCH_NAMESPACE"
	EnvClusterCRName = "FLASHSYSTEM_CLUSTERNAME"
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
	RoleCheckErrMessage    = "User role need to be Monitor"
	RestErrorMessage       = "Rest server hit unexpected error"
	ClusterErrMessage      = "Flash system cluster not online"
	ExporterReadyMessage   = "Flash system exporter is ready"
)

const INIT_POOL_ID = -1

var CacheManager DriverManager

type DriverManager struct {
	client.Client
	namespace  string
	systemName string
	ready      bool
	scPoolMap  map[string]string
}

func NewManager(scheme *runtime.Scheme, scPool map[string]string) (*DriverManager, error) {

	k8sclient, err := newClient(scheme)
	if err != nil {
		log.Errorf("fail to create k8s client, error: %v", err)
		return nil, err
	}

	CacheManager.Client = k8sclient

	// Flashsystem Cluster CR
	name, err := getFlashSystemClusterName()
	if err != nil {
		log.Errorf("Fail to get FlashSystemCluster CR Name, error: %v", err)
		return nil, err
	}

	// Namespace name
	namespace, err := getOperatorNamespace()
	if err != nil {
		log.Errorf("Fail to get operator namespace Name, error: %v", err)
		return nil, err
	}

	_, err = getFlashSystemClusterCR(k8sclient, name, namespace)
	if err != nil {
		log.Errorf("Fail to get FlashSystemCluster CR, error: %v", err)
		return nil, err
	}

	CacheManager.systemName = name
	CacheManager.namespace = namespace
	// CacheManager.ready = true
	CacheManager.Ready()
	CacheManager.scPoolMap = scPool

	return &CacheManager, nil
}

// Add helper function to expose the state for mockup
func (d *DriverManager) Ready() {
	d.ready = true
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

	fscluster, err := getFlashSystemClusterCR(k8sclient, d.systemName, d.namespace)
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
	fscluster, err := getFlashSystemClusterCR(d.Client, d.systemName, d.namespace)
	if err != nil {
		log.Errorf("Get flash system CR failed: %v", err)
		return err
	}

	t := metav1.Time{Time: time.Now()}
	evt := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", d.systemName, t.UnixNano()),
			Namespace: fscluster.Namespace,
			Labels:    conditionutil.GetLabels(d.systemName),
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

func getFlashSystemClusterCR(k8sclient client.Client, name string, namespace string) (*operatorapi.FlashSystemCluster, error) {
	fscluster := operatorapi.FlashSystemCluster{}
	err := k8sclient.Get(
		context.TODO(),
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		&fscluster,
	)
	if err != nil {
		log.Errorf("Fail to get FlashSystemCluster CR, error: %v", err)
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
