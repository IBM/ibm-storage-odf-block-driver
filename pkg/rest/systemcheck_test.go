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

	t.Run("Check User Administrator", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"Administrator"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if !valid {
			t.Errorf("Check user role should return true for role Administrator.")
		}
	})

	t.Run("Check User SecurityAdmin", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"SecurityAdmin"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if !valid {
			t.Errorf("Check user role should return true role  SecurityAdmin.")
		}
	})

	t.Run("Check User RestrictedAdmin", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"RestrictedAdmin"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if !valid {
			t.Errorf("Check user role should return true for role RestrictedAdmin.")
		}
	})

	t.Run("Check User Monitor", func(t *testing.T) {
		body = `[{"name":"u1"},{"role":"Monitor"},{"owner_id":""}]`
		valid, _ := c.CheckUserRole()
		if valid {
			t.Errorf("Check user role should return false for role Monitor.")
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
