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

package managers

import (
	"context"
	"fmt"
	drivermanager "github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operatorapi "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	log "k8s.io/klog"
)

const (
	SecretUsernameKey = "username"
	SecretPasswordKey = "password"
	SecretMgmtKey     = "management_address"
)

var Scheme = runtime.NewScheme()

func GetManagers(namespace string, currentSystems map[string]*rest.FSRestClient) (map[string]*rest.FSRestClient, error) {
	var newSystems = make(map[string]*rest.FSRestClient)

	fscMap, err := GetFscMap()
	if err != nil {
		log.Errorf("Read pool configmap failed, error: %v", err)
		return nil, err
	} else {
		log.Infof("Read pool configmap %v", fscMap)
	}

	for fscName, fscScSecretMap := range fscMap {
		if _, exist := currentSystems[fscName]; exist {
			log.Infof("Using existing manager for %s", fscName)
			newSystems[fscName] = currentSystems[fscName]
			// TODO - update storage credentials upon secret data change
			newSystems[fscName].DriverManager.UpdatePoolMap(fscScSecretMap.ScPoolMap)
		} else {
			log.Infof("Create new manager for %s", fscName)
			mgr, mgrErr := drivermanager.NewManager(Scheme, namespace, fscName, fscScSecretMap)
			if mgrErr != nil {
				log.Errorf("Initialize manager failed, error: %v", mgrErr)
				return nil, mgrErr
			}

			_, fscErr := mgr.GetFlashSystemClusterCR()
			if fscErr != nil {
				log.Errorf("Fail to get FlashSystemCluster CR, error: %v", fscErr)
				return nil, fscErr
			}

			restConfig, SecretErr := GetStorageCredentials(&mgr)
			if SecretErr != nil {
				log.Errorf("Fail to get FlashSystemCluster secret, error: %v", SecretErr)
				return nil, SecretErr
			}

			restClient, restErr := rest.NewFSRestClient(restConfig, &mgr)
			if err := checkRestClientState(restClient, restErr); err != nil {
				return nil, err
			}

			newSystems[fscName] = restClient
		}
	}

	return newSystems, nil
}

var GetStorageCredentials = func(d *drivermanager.DriverManager) (*rest.Config, error) {
	secret := &corev1.Secret{}
	err := d.Client.Get(context.TODO(),
		types.NamespacedName{
			Namespace: d.GetNamespaceName(),
			Name:      d.GetSecretName()},
		secret)
	if err != nil {
		return &rest.Config{}, err
	}

	restConfig := &rest.Config{
		Host:     string(secret.Data[SecretMgmtKey]),
		Username: string(secret.Data[SecretUsernameKey]),
		Password: string(secret.Data[SecretPasswordKey]),
	}

	return restConfig, nil
}

var GetFscMap = func() (map[string]operutil.FlashSystemClusterMapContent, error) {
	return operutil.ReadPoolConfigMapFile()
}

func checkRestClientState(restClient *rest.FSRestClient, err error) error {
	mgr := restClient.DriverManager
	if err != nil {
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.AuthFailure, drivermanager.AuthFailureMessage)
		log.Errorf("Fail to initialize rest client for %s, error: %s", mgr.GetSubsystemName(), err)
		return err
	}

	var valid bool
	valid, err = restClient.CheckVersion()
	if err != nil {
		log.Errorf("Flash system version check hit error: %s", err)
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RestFailure, drivermanager.RestErrorMessage)
		return err
	} else if !valid {
		log.Error("Flash system version invalid")
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.VersionCheckFailed, drivermanager.VersionCheckErrMessage)
		return fmt.Errorf("flash system version invalid")
	}

	// Print the user role in log.
	valid, err = restClient.CheckUserRole()
	if err != nil {
		log.Errorf("Flash system user role check hit errors: %s", err)
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RestFailure, drivermanager.RestErrorMessage)
		return err
	} else if !valid {
		log.Error("Flash system user role invalid")
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RoleCheckFailed, drivermanager.RoleCheckErrMessage)
		return fmt.Errorf("flash system user role invalid")
	}

	// Update ready condition
	{
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, true, "", "")
		var _ = mgr.UpdateCondition(operatorapi.StorageClusterReady, true, "", "")
		log.Info("Exporter check done, ready to serve")
		return nil
	}
}
