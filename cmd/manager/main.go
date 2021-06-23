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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	drivermanager "github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/prome"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operatorapi "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorapi.AddToScheme(scheme))
}

const (
	EnvUserName = "USERNAME"
	EnvPassword = "PASSWORD"
	EnvRestAddr = "REST_API_IP"
)

func getRestConfigFromEnv() (*rest.Config, error) {
	envVars := map[string]string{
		EnvUserName: "",
		EnvPassword: "",
		EnvRestAddr: "",
	}

	for k := range envVars {
		if value, ok := os.LookupEnv(k); ok {
			envVars[k] = value
		} else {
			return nil, fmt.Errorf("Required env variable: '%s' isn't found", k)
		}
	}

	restConfig := &rest.Config{
		Host:     envVars[EnvRestAddr],
		Username: envVars[EnvUserName],
		Password: envVars[EnvPassword],
	}

	return restConfig, nil
}

func main() {
	klog.Info("Try to read config file")

	// FIXME, demo of how to read pool
	scPoolMap, mistake := operutil.GetPoolConfigmapContent()
	if mistake != nil {
		log.Fatalf("Read ConfigMap failed, error: %s", mistake)
	} else {
		klog.Info("Pool ready", scPoolMap)
	}

	var err error
	var valid bool

	mgr, err := drivermanager.NewManager(scheme, scPoolMap.ScPool)
	if err != nil {
		log.Fatalf("Initialize mamager failed, error: %s", err)
	}

	restConfig, err := getRestConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	restClient, err := rest.NewFSRestClient(restConfig)
	if err != nil {
		// Update condition
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.AuthFailure, drivermanager.AuthFailureMessage)
		klog.Errorf("Fail to initialize rest client, error:%s", err)
		goto error_out
	}

	valid, err = restClient.CheckVersion()
	if err != nil {
		klog.Errorf("Flash system version check hit error: %s", err)
		// Update condition
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RestFailure, drivermanager.RestErrorMessage)
		goto error_out
	} else if !valid {
		klog.Error("Flash system version invalid")
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.VersionCheckFailed, drivermanager.VersionCheckErrMessage)
		goto error_out
	}

	// Print the user role in log.
	valid, err = restClient.CheckUserRole()
	if err != nil {
		klog.Errorf("Flash system user role check hit errors: %s", err)
		// Update condition
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RestFailure, drivermanager.RestErrorMessage)
		goto error_out
	} else if !valid {
		klog.Error("Flash system user role invalid")
		// Update condition
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RoleCheckFailed, drivermanager.RoleCheckErrMessage)
		goto error_out
	}

	// ready, err = restClient.CheckFlashsystemClusterState()
	// if err != nil {
	// 	klog.Errorf("Flash system cluster state check hit errors: %s", err)
	// 	// Update condition
	// 	var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.RestFailure, drivermanager.RestErrorMessage)
	// 	goto error_out
	// } else if !ready {
	// 	klog.Error("Flash system cluster is not online")
	// 	// Update condition
	// 	var _ = mgr.UpdateCondition(operatorapi.ExporterReady, false, drivermanager.ClusterNotOnline, drivermanager.ClusterErrMessage)
	// 	goto error_out
	// } else {
	// 	klog.Info("Flash system cluster ready")
	// }

	// Update ready condition
	{
		var _ = mgr.UpdateCondition(operatorapi.ExporterReady, true, "", "")
		var _ = mgr.UpdateCondition(operatorapi.StorageClusterReady, true, "", "")
		klog.Info("Exporter check done, ready to serve")
	}

	// TODO: handle pod terminating signal
	go prome.RunExporter(restClient, mgr.GetSubsystemName(), mgr.GetNamespaceName())
	// go prome.RunExporter(restClient, "FlashSystem", mgr.GetNamespaceName())

error_out:
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	klog.Info("Awaiting signal to exit")
	go func() {
		sig := <-sigs
		klog.Infof("Received signal: %+v, clean up...", sig)
		done <- true
	}()

	// exiting
	<-done
	klog.Info("Exiting")

}
