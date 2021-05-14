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
package prome

import (
	"fmt"
	"strings"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/rest"
	collector "github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/collectors"
)

const (
	VersionKey		= "code_level"
	UserRoleKey		= "usergrp_name"
	UserNameKey		= "name"
	ValidVersion	= "8.3.1"
	ValidRole		= "Monitor"
)

func RunExporter(restClient *rest.FSRestClient, subsystemName string, namespace string) {

	c, err := collector.NewPerfCollector(restClient, subsystemName, namespace)
	if err != nil {
		log.Warnf("NewFSPerfCollector fails, err:%s", err)
	}

	prometheus.MustRegister(c)
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Promethues Exporter</title></head>
            <body>
            <h1>FlashSystem Overall Perf Promethues Exporter </h1>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	log.Info("Beginning to serve on port :9100")
	log.Fatal(http.ListenAndServe(":9100", nil))
}

func CheckVersion(client *rest.FSRestClient) (bool, error) {
	var systeminfo rest.StorageSystem
	var valid bool
	var err error
	errCount := 0

	for i := 0; i < 2; i++ {
		systeminfo, err = client.Lssystem()
		if err != nil {
			client.Reconnect()
			log.Errorf("fails to do Lssystem, err:%s", err)
			errCount = errCount + 1
			continue
		} else {
			break
		}
	}

	if errCount == 2 {
		log.Error("get system version error")
		return false, fmt.Errorf("rest client error")
	}

	codelevel, _ := systeminfo[VersionKey]
	version := fmt.Sprintf("%v", codelevel)
	versions := strings.Split(version, " ")
	// Compare
	validversion := normalizeVersion(ValidVersion, 2, 4)
	systemversion := normalizeVersion(versions[0], 2, 4)
	if systemversion >= validversion {
		valid = true
	} else {
		valid = false
	}

	return valid, nil
}

func CheckUserRole(name string, client *rest.FSRestClient) (bool, error) {
	var userinfo rest.Users
	var valid bool
	var err error
	errCount := 0

	for i := 0; i < 2; i++ {
		userinfo, err = client.Lsuser()
		if err != nil {
			client.Reconnect()
			errCount = errCount + 1
			log.Errorf("fails to do Lsuser, err:%s", err)
			continue
		} else {
			break
		}
	}

	if errCount == 2 {
		return valid, fmt.Errorf("rest client error")
	}

	for _, user := range userinfo {
		username := fmt.Sprintf("%v", user[UserNameKey])
		if username == name {
			role, _ := user[UserRoleKey]
			rolename := fmt.Sprintf("%v", role)
			log.Infof("user name: %s, role: %s", username, rolename)
			valid = true
			//if rolename != ValidRole {
			//	valid = false
			//} else {
			//}
		}
	}

	return valid, nil
}

func CheckFlashsystemClusterState(client *rest.FSRestClient) (bool, error) {
	ready := true
	var err error

	var nodes rest.Nodes
	for i := 0; i < 2; i++ {
		nodes, err = client.Lsnode()
		if err != nil {
			err = client.Reconnect()
			if err != nil {
				log.Errorf("fails to authenticate rest server, err:%s", err)
				return false, err
			}
		} else {
			break
		}
	}

	for _, node := range nodes {
		status, ok := node["status"]
		if !ok || "online" != status {
			ready = false
			break
		}
	}

	return ready, nil
}

func normalizeVersion(s string, width, parts int) string {
	strList := strings.Split(s, ".")
	v := ""
	for _, value := range strList {
		v += fmt.Sprintf("%0*s", width, value)
	}
	for i := len(strList); i < parts; i++ {
		v += fmt.Sprintf("%0*s", width, "")
	}
	log.Infof("parseVersion: [%s] => [%s]", s, v)
	return v
}
