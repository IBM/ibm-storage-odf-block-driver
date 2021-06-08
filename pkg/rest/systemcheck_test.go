package rest

import (
	"net/http"
	"testing"
)

var body string
var c = FSRestClient{PostRequester: &Requester{
	poster: func(req *http.Request, c *FSRestClient) ([]byte, int, error) {
		return []byte(body), 200, nil
	},
}}

func TestNormalizeVersion(t *testing.T) {

	// Happy path
	t.Run("Check valid version", func(t *testing.T) {
		body = `{"code_level": "8.4.0.2 (build 152.23.2102111856000)"}`
		valid, err := c.CheckVersion()
		if err != nil || !valid {
			t.Errorf("Check version should return true.")
		}
	})

	// Unhappy path
	t.Run("Check invalid Version", func(t *testing.T) {
		body = `{"code_level": "8.1.0.2 (build 152.23.2102111856000)"}`
		valid, _ := c.CheckVersion()
		if valid {
			t.Errorf("Check version should return false.")
		}
	})
}

func TestUserRole(t *testing.T) {

	t.Run("Check User true", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"RestrictedAdmin"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if !valid {
			t.Errorf("Check user role should return true for u1.")
		}
	})

	t.Run("Check User false", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"Monitor"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if valid {
			t.Errorf("Check user role should return false for ux.")
		}
	})
}

func TestCheckFlashsystemClusterState(t *testing.T) {
	n1 := `{"name":"node1","id":"1","status":"online","IO_group_name":"io_grp0"}`
	n2 := `{"name":"node2","id":"2","status":"online","IO_group_name":"io_grp0"}`
	n3 := `{"name":"node3","id":"3","status":"starting","IO_group_name":"io_grp1"}`
	n4 := `{"name":"node4","id":"4","status":"online","IO_group_name":"io_grp1"}`
	n5 := `{"name":"node5","id":"5","status":"online","IO_group_name":"io_grp2"}`

	t.Run("Check cluster state: node online, iogrp health", func(t *testing.T) {
		body = "[" + n1 + "," + n2 + "]"
		valid, _ := c.CheckFlashsystemClusterState()
		if !valid {
			t.Errorf("CheckFlashsystemClusterState should return true for node online and iogrp health.")
		}
	})

	t.Run("Check cluster state: node ofline, iogrp health", func(t *testing.T) {
		body = "[" + n1 + "," + n2 + "," + n3 + "," + n4 + "]"
		valid, _ := c.CheckFlashsystemClusterState()
		if valid {
			t.Errorf("CheckFlashsystemClusterState should return false for node online and iogrp health.")
		}
	})

	t.Run("Check cluster state: node online, iogrp Unhealth", func(t *testing.T) {
		body = "[" + n1 + "," + n2 + "," + n5 + "]"
		valid, _ := c.CheckFlashsystemClusterState()
		if valid {
			t.Errorf("CheckFlashsystemClusterState should return false for node online and iogrp Unhealth.")
		}
	})
}
