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
	clientmanagers "github.com/IBM/ibm-storage-odf-block-driver/pkg/managers"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/prome"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operatorapi "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	log "k8s.io/klog"
	"os"
	"os/signal"
	"syscall"
)

const (
	EnvNamespaceName = "WATCH_NAMESPACE"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(clientmanagers.Scheme))
	utilruntime.Must(operatorapi.AddToScheme(clientmanagers.Scheme))
}

func main() {
	log.InitFlags(nil)

	namespace, err := getOperatorNamespace()
	if err != nil {
		// todo tal - remove panic ? os.Exit(1)
		os.Exit(1)
	}

	systems, err := clientmanagers.GetManagers(namespace, make(map[string]*rest.FSRestClient))
	if err != nil || len(systems) == 0 {
		log.Error("Could not create managers")
		os.Exit(1)
	}

	// TODO: handle pod terminating signal
	go prome.RunExporter(systems, namespace)
	waitForSignal()
	// TODO: understand why do we need goto command instead of regular error\termination process ? remove goto_error

}

func waitForSignal() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Awaiting signal to exit")
	go func() {
		sig := <-sigs
		log.Infof("Received signal: %+v, clean up...", sig)
		done <- true
	}()

	// exiting
	<-done
	log.Info("Exiting")
}

func getOperatorNamespace() (string, error) {
	if value, ok := os.LookupEnv(EnvNamespaceName); ok {
		return value, nil
	} else {
		return "", fmt.Errorf("required env variable: '%s' isn't found", EnvNamespaceName)
	}
}
