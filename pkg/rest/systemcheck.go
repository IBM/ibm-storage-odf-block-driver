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

package rest

import (
	"fmt"
	"strings"

	log "k8s.io/klog"
)

const (
	VersionKey   = "code_level"
	UserRoleKey  = "role"
	ValidVersion = "8.3.1"
)

func (c *FSRestClient) CheckVersion() (bool, error) {
	systeminfo, err := c.Lssystem()
	if err != nil {
		log.Errorf("get flash system version error: %v", err)
		return false, err
	}

	version := systeminfo[VersionKey].(string)
	versions := strings.Split(version, " ")
	// Compare
	validversion := normalizeVersion(ValidVersion, 2, 4)
	systemversion := normalizeVersion(versions[0], 2, 4)

	bValid := systemversion >= validversion
	if !bValid {
		log.Errorf("Unsupported version %s. Supported version above than %s", version, ValidVersion)
	}
	return bValid, nil
}

func (c *FSRestClient) CheckUserRole() (bool, error) {
	userinfo, err := c.Lscurrentuser()
	if err != nil {
		return false, err
	}

	for _, info := range userinfo {
		role, bHas := info[UserRoleKey]
		if bHas {
			switch role {
			case "Administrator", "SecurityAdmin", "RestrictedAdmin":
				return true, nil
			}
			log.Infof("The current user role is %v.", role)
		}
	}
	return false, nil
}

func isHealth(status string) bool {
	switch status {
	case "starting", "service", "pending", "offline", "flushing", "deleting", "adding":
		return false
	}
	return true
}

func (c *FSRestClient) CheckFlashsystemClusterState() (bool, error) {
	nodes, err := c.Lsnode()
	if err != nil {
		return false, err
	}

	iogrps := map[string]int{}
	for _, node := range nodes {
		if !isHealth(node["status"]) {
			log.Infof("The node %s id %s status %s is unhealthy.", node["name"], node["id"], node["status"])
			return false, nil
		}
		iogrps[node["IO_group_name"]]++
	}

	// Check grpName io_grp0-3 to ensure the node_count is 1, in not HA mode
	for grpName, nodeCnt := range iogrps {
		if nodeCnt == 1 && strings.HasPrefix(grpName, "io_grp") {
			log.Infof("The iogrp %s node count is 1.", grpName)
			return false, nil
		}
	}

	return true, nil
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
	return v
}
