package rest

import (
	"fmt"
	"strings"

	log "k8s.io/klog"
)

const (
	VersionKey   = "code_level"
	UserRoleKey  = "usergrp_name"
	UserNameKey  = "name"
	ValidVersion = "8.3.1"
	ValidRole    = "Monitor"
)

func (c *FSRestClient) CheckVersion() (bool, error) {
	systeminfo, err := c.Lssystem()
	if err != nil {
		log.Error("get system version error")
		return false, fmt.Errorf("rest client error")
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

func (c *FSRestClient) CheckUserRole(name string) (bool, error) {
	userinfo, err := c.Lsuser()
	if err != nil {
		return false, fmt.Errorf("get user error due rest client error")
	}

	for _, user := range userinfo {
		username := user[UserNameKey].(string)
		if username == name {
			log.Infof("user name: %s, role: %v", username, user[UserRoleKey])
			switch user[UserRoleKey] {
			case "Administrator", "SecurityAdmin", "RestrictedAdmin":
				return true, nil
			}
			return false, nil
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
	log.Infof("parseVersion: [%s] => [%s]", s, v)
	return v
}
