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

package collectors

import (
	"fmt"
	drivermanager "github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
	clientmanagers "github.com/IBM/ibm-storage-odf-block-driver/pkg/managers"

	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

func poster(req *http.Request, c *rest.FSRestClient) ([]byte, int, error) {
	path := fmt.Sprintf("%v", req.URL)
	body := ""
	switch path {
	case "/lssystem":
		body = `{"code_level": "8.4.0.2 (build 152.23.2102111856000)","product_name":"IBM SAN Volume Controller", "physical_capacity":"70727768211456", "physical_free_capacity":"37416452751360", "total_reclaimable_capacity":"32564"}`
	case "/lssystemstats":
		body = `[
			{
				"stat_name": "vdisk_r_mb",
				"stat_current": "0",
				"stat_peak": "0",
				"stat_peak_time": "210604162102"
			},
			{
				"stat_name": "vdisk_w_mb",
				"stat_current": "0",
				"stat_peak": "16",
				"stat_peak_time": "210604161947"
			},
			{
				"stat_name": "vdisk_r_io",
				"stat_current": "0",
				"stat_peak": "1",
				"stat_peak_time": "210604161657"
			},
			{
				"stat_name": "vdisk_w_io",
				"stat_current": "11",
				"stat_peak": "176",
				"stat_peak_time": "210604161812"
			},
			{
				"stat_name": "vdisk_ms",
				"stat_current": "10",
				"stat_peak": "20",
				"stat_peak_time": "210716134213"
			},
			{
				"stat_name": "vdisk_r_ms",
				"stat_current": "0",
				"stat_peak": "0",
				"stat_peak_time": "210604162102"
			},
			{
				"stat_name": "vdisk_w_ms",
				"stat_current": "1",
				"stat_peak": "690",
				"stat_peak_time": "210604161807"
			}
		]`
	case "/lsnode":
		body = `[
			{"name":"node1","id":"1","status":"online","IO_group_name":"io_grp0"},
			{"name":"node2","id":"2","status":"online","IO_group_name":"io_grp0"}
		]`
	case "/lsmdiskgrp":
		body = `[
			{
				"id": "0",
				"name": "Pool0",
				"status": "online",
				"mdisk_count": "6",
				"vdisk_count": "14",
				"capacity": "7882338729984",
				"extent_size": "1024",
				"free_capacity": "6386616369152",
				"virtual_capacity": "3554085437440",
				"used_capacity": "1446814941184",
				"real_capacity": "1489086635008",
				"overallocation": "45",
				"warning": "80",
				"easy_tier": "auto",
				"easy_tier_status": "balanced",
				"gui_id": "0",
				"gui_iogrp_id": "",
				"compression_active": "no",
				"compression_virtual_capacity": "0",
				"compression_compressed_capacity": "0",
				"compression_uncompressed_capacity": "0",
				"parent_mdisk_grp_id": "0",
				"parent_mdisk_grp_name": "Pool0",
				"child_mdisk_grp_count": "0",
				"child_mdisk_grp_capacity": "0",
				"type": "parent",
				"encrypt": "no",
				"owner_type": "none",
				"owner_id": "",
				"owner_name": "",
				"site_id": "",
				"site_name": "",
				"has_encryption_key": "no",
				"child_has_encryption_key": "no",
				"data_reduction": "no",
				"used_capacity_before_reduction": "0",
				"used_capacity_after_reduction": "0",
				"overhead_capacity": "0",
				"compression_opportunity": "0",
				"deduplication_opportunity": "0",
				"deduplication_capacity_saving": "0",
				"reclaimable_capacity": "0",
				"physical_capacity": "10799695265792",
				"physical_free_capacity": "10798621523968",
				"shared_resources": "yes",
				"vdisk_protection_enabled": "yes",
				"vdisk_protection_status": "inactive",
				"easy_tier_fcm_over_allocation_max": "",
				"auto_expand": "no",
				"auto_expand_max_capacity": "0"
			},
			{
				"id": "1",
				"name": "Pool1",
				"status": "offline",
				"mdisk_count": "1",
				"vdisk_count": "2",
				"capacity": "644245094400",
				"extent_size": "1024",
				"free_capacity": "41875931136",
				"virtual_capacity": "612032839680",
				"used_capacity": "601296207872",
				"real_capacity": "601526946816",
				"overallocation": "95",
				"warning": "80",
				"easy_tier": "auto",
				"easy_tier_status": "balanced",
				"gui_id": "0",
				"gui_iogrp_id": "",
				"compression_active": "no",
				"compression_virtual_capacity": "0",
				"compression_compressed_capacity": "0",
				"compression_uncompressed_capacity": "0",
				"parent_mdisk_grp_id": "1",
				"parent_mdisk_grp_name": "Pool1",
				"child_mdisk_grp_count": "0",
				"child_mdisk_grp_capacity": "0",
				"type": "parent",
				"encrypt": "no",
				"owner_type": "none",
				"owner_id": "",
				"owner_name": "",
				"site_id": "",
				"site_name": "",
				"has_encryption_key": "no",
				"child_has_encryption_key": "no",
				"data_reduction": "no",
				"used_capacity_before_reduction": "0",
				"used_capacity_after_reduction": "0",
				"overhead_capacity": "0",
				"compression_opportunity": "0",
				"deduplication_opportunity": "0",
				"deduplication_capacity_saving": "0",
				"reclaimable_capacity": "0",
				"physical_capacity": "10799695265792",
				"physical_free_capacity": "10798621523968",
				"shared_resources": "yes",
				"vdisk_protection_enabled": "yes",
				"vdisk_protection_status": "inactive",
				"easy_tier_fcm_over_allocation_max": "",
				"auto_expand": "no",
				"auto_expand_max_capacity": "0"
			},
			{
				"id": "2",
				"name": "Pool2",
				"status": "online",
				"mdisk_count": "1",
				"vdisk_count": "0",
				"capacity": "644245094400",
				"extent_size": "1024",
				"free_capacity": "644245094400",
				"virtual_capacity": "0",
				"used_capacity": "0",
				"real_capacity": "0",
				"overallocation": "0",
				"warning": "60",
				"easy_tier": "auto",
				"easy_tier_status": "balanced",
				"gui_id": "0",
				"gui_iogrp_id": "",
				"compression_active": "no",
				"compression_virtual_capacity": "0",
				"compression_compressed_capacity": "0",
				"compression_uncompressed_capacity": "0",
				"parent_mdisk_grp_id": "2",
				"parent_mdisk_grp_name": "Pool2",
				"child_mdisk_grp_count": "0",
				"child_mdisk_grp_capacity": "0",
				"type": "parent",
				"encrypt": "no",
				"owner_type": "none",
				"owner_id": "",
				"owner_name": "",
				"site_id": "",
				"site_name": "",
				"has_encryption_key": "no",
				"child_has_encryption_key": "no",
				"data_reduction": "no",
				"used_capacity_before_reduction": "0",
				"used_capacity_after_reduction": "0",
				"overhead_capacity": "0",
				"compression_opportunity": "0",
				"deduplication_opportunity": "0",
				"deduplication_capacity_saving": "0",
				"reclaimable_capacity": "0",
				"physical_capacity": "10799695265792",
				"physical_free_capacity": "10798621523968",
				"shared_resources": "yes",
				"vdisk_protection_enabled": "yes",
				"vdisk_protection_status": "inactive",
				"easy_tier_fcm_over_allocation_max": "",
				"auto_expand": "no",
				"auto_expand_max_capacity": "0"
			}
		]`
	}
	return []byte(body), 200, nil
}

func posterSecondSystem(req *http.Request, c *rest.FSRestClient) ([]byte, int, error) {
	path := fmt.Sprintf("%v", req.URL)
	body := ""
	switch path {
	case "/lssystem":
		body = `{"code_level": "8.5.2.0 (build 161.15.2208121040000)","product_name":"IBM FlashSystem 9200", "physical_capacity":"76427768211456", "physical_free_capacity":"28416452751360", "total_reclaimable_capacity":"40564"}`
	case "/lssystemstats":
		body = `[
			{
				"stat_name": "vdisk_r_mb",
				"stat_current": "5",
				"stat_peak": "0",
				"stat_peak_time": "210604162102"
			},
			{
				"stat_name": "vdisk_w_mb",
				"stat_current": "1024",
				"stat_peak": "16",
				"stat_peak_time": "210604161947"
			},
			{
				"stat_name": "vdisk_r_io",
				"stat_current": "2",
				"stat_peak": "4",
				"stat_peak_time": "210604161657"
			},
			{
				"stat_name": "vdisk_w_io",
				"stat_current": "13",
				"stat_peak": "176",
				"stat_peak_time": "210604161812"
			},
			{
				"stat_name": "vdisk_ms",
				"stat_current": "30",
				"stat_peak": "20",
				"stat_peak_time": "210716134213"
			},
			{
				"stat_name": "vdisk_r_ms",
				"stat_current": "5",
				"stat_peak": "0",
				"stat_peak_time": "210604162102"
			},
			{
				"stat_name": "vdisk_w_ms",
				"stat_current": "2",
				"stat_peak": "690",
				"stat_peak_time": "210604161807"
			}
		]`
	case "/lsnode":
		body = `[
			{"name":"node1","id":"1","status":"online","IO_group_name":"io_grp0"},
			{"name":"node2","id":"2","status":"online","IO_group_name":"io_grp0"}
		]`
	case "/lsmdiskgrp":
		body = `[
			{
				"id": "5",
				"name": "Pool5",
				"status": "online",
				"mdisk_count": "2",
				"vdisk_count": "0",
				"capacity": "4244114883215",
				"extent_size": "1024",
				"free_capacity": "3936251627438",
				"virtual_capacity": "0",
				"used_capacity": "0",
				"real_capacity": "0",
				"overallocation": "0",
				"warning": "80",
				"easy_tier": "auto",
				"easy_tier_status": "balanced",
				"gui_id": "0",
				"gui_iogrp_id": "",
				"compression_active": "no",
				"compression_virtual_capacity": "0",
				"compression_compressed_capacity": "0",
				"compression_uncompressed_capacity": "0",
				"parent_mdisk_grp_id": "5",
				"parent_mdisk_grp_name": "Pool5",
				"child_mdisk_grp_count": "0",
				"child_mdisk_grp_capacity": "0",
				"type": "parent",
				"encrypt": "no",
				"owner_type": "none",
				"owner_id": "",
				"owner_name": "",
				"site_id": "",
				"site_name": "",
				"has_encryption_key": "no",
				"child_has_encryption_key": "no",
				"data_reduction": "no",
				"used_capacity_before_reduction": "0",
				"used_capacity_after_reduction": "0",
				"overhead_capacity": "0",
				"compression_opportunity": "0",
				"deduplication_opportunity": "0",
				"deduplication_capacity_saving": "0",
				"reclaimable_capacity": "40",
				"physical_capacity": "1649267441664",
				"physical_free_capacity": "1594291860275",
				"shared_resources": "yes",
				"vdisk_protection_enabled": "yes",
				"vdisk_protection_status": "inactive",
				"easy_tier_fcm_over_allocation_max": "",
				"auto_expand": "no",
				"auto_expand_max_capacity": "0"
			},
			{
				"id": "6",
				"name": "Pool6",
				"status": "offline",
				"mdisk_count": "1",
				"vdisk_count": "2",
				"capacity": "644245094400",
				"extent_size": "1024",
				"free_capacity": "41875931136",
				"virtual_capacity": "612032839680",
				"used_capacity": "601296207872",
				"real_capacity": "601526946816",
				"overallocation": "95",
				"warning": "80",
				"easy_tier": "auto",
				"easy_tier_status": "balanced",
				"gui_id": "0",
				"gui_iogrp_id": "",
				"compression_active": "no",
				"compression_virtual_capacity": "0",
				"compression_compressed_capacity": "0",
				"compression_uncompressed_capacity": "0",
				"parent_mdisk_grp_id": "1",
				"parent_mdisk_grp_name": "Pool1",
				"child_mdisk_grp_count": "0",
				"child_mdisk_grp_capacity": "0",
				"type": "parent",
				"encrypt": "no",
				"owner_type": "none",
				"owner_id": "",
				"owner_name": "",
				"site_id": "",
				"site_name": "",
				"has_encryption_key": "no",
				"child_has_encryption_key": "no",
				"data_reduction": "no",
				"used_capacity_before_reduction": "0",
				"used_capacity_after_reduction": "0",
				"overhead_capacity": "0",
				"compression_opportunity": "0",
				"deduplication_opportunity": "0",
				"deduplication_capacity_saving": "0",
				"reclaimable_capacity": "0",
				"physical_capacity": "10799695265792",
				"physical_free_capacity": "10798621523968",
				"shared_resources": "yes",
				"vdisk_protection_enabled": "yes",
				"vdisk_protection_status": "inactive",
				"easy_tier_fcm_over_allocation_max": "",
				"auto_expand": "no",
				"auto_expand_max_capacity": "0"
			}
		]`
	}
	return []byte(body), 200, nil
}

var restConfig1 = rest.Config{
	Host:     "FS-Host",
	Username: "FS-Username",
	Password: "FS-Password",
}

var restConfig2 = rest.Config{
	Host:     "FS-Host-second",
	Username: "FS-Username-second",
	Password: "FS-Password-second",
}

var manager1 = drivermanager.DriverManager{SystemName: "FS-system-name"}
var manager2 = drivermanager.DriverManager{SystemName: "FS-system-name-second"}
var client1 = &rest.FSRestClient{PostRequester: rest.NewRequester(poster), DriverManager: &manager1, RestConfig: restConfig1}
var client2 = &rest.FSRestClient{PostRequester: rest.NewRequester(posterSecondSystem), DriverManager: &manager2, RestConfig: restConfig2}

var testCollector, _ = NewPerfCollector(map[string]*rest.FSRestClient{"FS-system-name": client1,
	"FS-system-name-second": client2}, "FS-ns")

func TestMetrics(t *testing.T) {
	// Mock the dependency
	clientmanagers.GetStorageCredentials = func(client *drivermanager.DriverManager) (rest.Config, error) {
		if client.SystemName == "FS-system-name" {
			return restConfig1, nil
		}
		return restConfig2, nil
	}

	clientmanagers.GetFscMap = func() (map[string]operutil.FlashSystemClusterMapContent, error) {
		fscScSecretMap := operutil.FlashSystemClusterMapContent{ScPoolMap: map[string]string{}, Secret: "FC-secret"}
		fscScSecretMap.ScPoolMap["fs-sc-default"] = "Pool0"
		fscScSecretMap.ScPoolMap["fs-sc-1"] = "Pool0"
		fscScSecretMap.ScPoolMap["fs-sc-2"] = "Pool1"
		fscScSecretMap.ScPoolMap["fs-sc-3"] = "Pool1"
		fscScSecretMap.ScPoolMap["fs-sc-4"] = "Pool2"

		fscScSecretMapSecond := operutil.FlashSystemClusterMapContent{ScPoolMap: map[string]string{}, Secret: "FC-secret-second"}
		fscScSecretMapSecond.ScPoolMap["fs-second-sc-1"] = "Pool5"
		fscScSecretMapSecond.ScPoolMap["fs-second-sc-2"] = "Pool6"

		return map[string]operutil.FlashSystemClusterMapContent{"FS-system-name": fscScSecretMap, "FS-system-name-second": fscScSecretMapSecond}, nil
	}

	testCollector.systems["FS-system-name"].DriverManager.Ready()
	testCollector.systems["FS-system-name-second"].DriverManager.Ready()

	expected := `

	# HELP flashsystem_pool_logical_capacity_usable_bytes Pool logical usable capacity (byte)
	# TYPE flashsystem_pool_logical_capacity_usable_bytes gauge
	flashsystem_pool_logical_capacity_usable_bytes{pool_name="Pool0",subsystem_name="FS-system-name"} 6.386616369152e+12
	flashsystem_pool_logical_capacity_usable_bytes{pool_name="Pool1",subsystem_name="FS-system-name"} 4.1875931136e+10
	flashsystem_pool_logical_capacity_usable_bytes{pool_name="Pool2",subsystem_name="FS-system-name"} 6.442450944e+11
	flashsystem_pool_logical_capacity_usable_bytes{pool_name="Pool5",subsystem_name="FS-system-name-second"} 3.936251627478e+12
	flashsystem_pool_logical_capacity_usable_bytes{pool_name="Pool6",subsystem_name="FS-system-name-second"} 4.1875931136e+10

	# HELP flashsystem_pool_logical_capacity_used_bytes Pool logical used capacity (byte)
	# TYPE flashsystem_pool_logical_capacity_used_bytes gauge
	flashsystem_pool_logical_capacity_used_bytes{pool_name="Pool0",subsystem_name="FS-system-name"} 1.495722360832e+12
	flashsystem_pool_logical_capacity_used_bytes{pool_name="Pool1",subsystem_name="FS-system-name"} 6.02369163264e+11
	flashsystem_pool_logical_capacity_used_bytes{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	flashsystem_pool_logical_capacity_used_bytes{pool_name="Pool5",subsystem_name="FS-system-name-second"} 3.07863255737e+11
	flashsystem_pool_logical_capacity_used_bytes{pool_name="Pool6",subsystem_name="FS-system-name-second"} 6.02369163264e+11

	# HELP flashsystem_pool_capacity_usable_bytes Pool usable capacity (Byte)
	# TYPE flashsystem_pool_capacity_usable_bytes gauge
	flashsystem_pool_capacity_usable_bytes{pool_name="Pool0",subsystem_name="FS-system-name"} 1.0798621523968e+13
	flashsystem_pool_capacity_usable_bytes{pool_name="Pool1",subsystem_name="FS-system-name"} 1.0798621523968e+13
	flashsystem_pool_capacity_usable_bytes{pool_name="Pool2",subsystem_name="FS-system-name"} 1.0798621523968e+13
	flashsystem_pool_capacity_usable_bytes{pool_name="Pool5",subsystem_name="FS-system-name-second"} 1.594291860315e+12
	flashsystem_pool_capacity_usable_bytes{pool_name="Pool6",subsystem_name="FS-system-name-second"} -1

	# HELP flashsystem_pool_capacity_used_bytes Pool used capacity (byte)
	# TYPE flashsystem_pool_capacity_used_bytes gauge
	flashsystem_pool_capacity_used_bytes{pool_name="Pool0",subsystem_name="FS-system-name"} 1.073741824e+09
	flashsystem_pool_capacity_used_bytes{pool_name="Pool1",subsystem_name="FS-system-name"} 1.073741824e+09
	flashsystem_pool_capacity_used_bytes{pool_name="Pool2",subsystem_name="FS-system-name"} 1.073741824e+09
	flashsystem_pool_capacity_used_bytes{pool_name="Pool5",subsystem_name="FS-system-name-second"} 5.4975581349e+10
	flashsystem_pool_capacity_used_bytes{pool_name="Pool6",subsystem_name="FS-system-name-second"} -1

	# HELP flashsystem_capacity_warning_threshold Pool capacity warning threshold
	# TYPE flashsystem_capacity_warning_threshold gauge
	flashsystem_capacity_warning_threshold{pool_name="Pool0",subsystem_name="FS-system-name"} 80
	flashsystem_capacity_warning_threshold{pool_name="Pool1",subsystem_name="FS-system-name"} 80
	flashsystem_capacity_warning_threshold{pool_name="Pool2",subsystem_name="FS-system-name"} 60
	flashsystem_capacity_warning_threshold{pool_name="Pool5",subsystem_name="FS-system-name-second"} 80
	flashsystem_capacity_warning_threshold{pool_name="Pool6",subsystem_name="FS-system-name-second"} 80

	# HELP flashsystem_pool_health Pool health status
	# TYPE flashsystem_pool_health gauge
	flashsystem_pool_health{pool_name="Pool0",subsystem_name="FS-system-name"} 0
	flashsystem_pool_health{pool_name="Pool1",subsystem_name="FS-system-name"} 2
	flashsystem_pool_health{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	flashsystem_pool_health{pool_name="Pool5",subsystem_name="FS-system-name-second"} 0
	flashsystem_pool_health{pool_name="Pool6",subsystem_name="FS-system-name-second"} 2

	# HELP flashsystem_pool_savings_bytes dedupe, thin provisioning, and compression savings
	# TYPE flashsystem_pool_savings_bytes gauge
	flashsystem_pool_savings_bytes{pool_name="Pool0",subsystem_name="FS-system-name"} 2.064998802432e+12
	flashsystem_pool_savings_bytes{pool_name="Pool1",subsystem_name="FS-system-name"} 1.0505892864e+10
	flashsystem_pool_savings_bytes{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	flashsystem_pool_savings_bytes{pool_name="Pool5",subsystem_name="FS-system-name-second"} 0
	flashsystem_pool_savings_bytes{pool_name="Pool6",subsystem_name="FS-system-name-second"} 1.0505892864e+10

	# HELP flashsystem_pool_metadata Pool metadata
	# TYPE flashsystem_pool_metadata gauge
	flashsystem_pool_metadata{pool_id="0",pool_name="Pool0",storageclass="fs-sc-1,fs-sc-default",subsystem_name="FS-system-name"} 0
	flashsystem_pool_metadata{pool_id="1",pool_name="Pool1",storageclass="fs-sc-2,fs-sc-3",subsystem_name="FS-system-name"} 0
	flashsystem_pool_metadata{pool_id="2",pool_name="Pool2",storageclass="fs-sc-4",subsystem_name="FS-system-name"} 0
	flashsystem_pool_metadata{pool_id="5",pool_name="Pool5",storageclass="fs-second-sc-1",subsystem_name="FS-system-name-second"} 0
	flashsystem_pool_metadata{pool_id="6",pool_name="Pool6",storageclass="fs-second-sc-2",subsystem_name="FS-system-name-second"} 0

	# HELP flashsystem_subsystem_metadata System information
	# TYPE flashsystem_subsystem_metadata gauge
	flashsystem_subsystem_metadata{model="SAN Volume Controller",subsystem_name="FS-system-name",vendor="IBM",version="8.4.0.2"} 0
	flashsystem_subsystem_metadata{model="FlashSystem 9200",subsystem_name="FS-system-name-second",vendor="IBM",version="8.5.2.0"} 0

	# HELP flashsystem_subsystem_health System health
	# TYPE flashsystem_subsystem_health gauge
	flashsystem_subsystem_health{subsystem_name="FS-system-name"} 0
	flashsystem_subsystem_health{subsystem_name="FS-system-name-second"} 0
	
	# HELP flashsystem_subsystem_response System response
    # TYPE flashsystem_subsystem_response gauge
	flashsystem_subsystem_response{model="",subsystem_name="FS-system-name",vendor="",version=""} 1
    flashsystem_subsystem_response{model="",subsystem_name="FS-system-name-second",vendor="",version=""} 1

	# HELP flashsystem_subsystem_latency_seconds overall performance - average latency seconds
	# TYPE flashsystem_subsystem_latency_seconds gauge
	flashsystem_subsystem_latency_seconds{subsystem_name="FS-system-name"} 0.01
	flashsystem_subsystem_latency_seconds{subsystem_name="FS-system-name-second"} 0.03

	# HELP flashsystem_subsystem_rd_latency_seconds overall performance - read latency seconds
	# TYPE flashsystem_subsystem_rd_latency_seconds gauge
	flashsystem_subsystem_rd_latency_seconds{subsystem_name="FS-system-name"} 0
	flashsystem_subsystem_rd_latency_seconds{subsystem_name="FS-system-name-second"} 0.005

	# HELP flashsystem_subsystem_wr_latency_seconds overall performance - write latency seconds
	# TYPE flashsystem_subsystem_wr_latency_seconds gauge
	flashsystem_subsystem_wr_latency_seconds{subsystem_name="FS-system-name"} 0.001
	flashsystem_subsystem_wr_latency_seconds{subsystem_name="FS-system-name-second"} 0.002

	# HELP flashsystem_subsystem_rd_bytes overall performance - read throughput bytes/s
	# TYPE flashsystem_subsystem_rd_bytes gauge
	flashsystem_subsystem_rd_bytes{subsystem_name="FS-system-name"} 0
	flashsystem_subsystem_rd_bytes{subsystem_name="FS-system-name-second"} 5.24288e+06

	# HELP flashsystem_subsystem_wr_bytes overall performance - write throughput bytes/s
	# TYPE flashsystem_subsystem_wr_bytes gauge
	flashsystem_subsystem_wr_bytes{subsystem_name="FS-system-name"} 0
	flashsystem_subsystem_wr_bytes{subsystem_name="FS-system-name-second"} 1.073741824e+9

	# HELP flashsystem_subsystem_rd_iops overall performance - read IOPS
	# TYPE flashsystem_subsystem_rd_iops gauge
	flashsystem_subsystem_rd_iops{subsystem_name="FS-system-name"} 0
	flashsystem_subsystem_rd_iops{subsystem_name="FS-system-name-second"} 2

	# HELP flashsystem_subsystem_wr_iops overall performance - write IOPS
	# TYPE flashsystem_subsystem_wr_iops gauge
	flashsystem_subsystem_wr_iops{subsystem_name="FS-system-name"} 11
	flashsystem_subsystem_wr_iops{subsystem_name="FS-system-name-second"} 13

	# HELP flashsystem_subsystem_physical_free_capacity_bytes System physical free capacity (byte)
	# TYPE flashsystem_subsystem_physical_free_capacity_bytes gauge
	flashsystem_subsystem_physical_free_capacity_bytes{subsystem_name="FS-system-name"} 3.7416452783924e+13
    flashsystem_subsystem_physical_free_capacity_bytes{subsystem_name="FS-system-name-second"} 2.8416452791924e+13

	# HELP flashsystem_subsystem_physical_total_capacity_bytes System physical total capacity (byte)
    # TYPE flashsystem_subsystem_physical_total_capacity_bytes gauge
    flashsystem_subsystem_physical_total_capacity_bytes{subsystem_name="FS-system-name"} 7.0727768211456e+13
    flashsystem_subsystem_physical_total_capacity_bytes{subsystem_name="FS-system-name-second"} 7.6427768211456e+13
    
	# HELP flashsystem_subsystem_physical_used_capacity_bytes System physical used capacity (byte)
    # TYPE flashsystem_subsystem_physical_used_capacity_bytes gauge
    flashsystem_subsystem_physical_used_capacity_bytes{subsystem_name="FS-system-name"} 3.3311315427532e+13
    flashsystem_subsystem_physical_used_capacity_bytes{subsystem_name="FS-system-name-second"} 4.8011315419532e+13
	`

	err := testutil.CollectAndCompare(testCollector, strings.NewReader(expected),
		SystemReadIOPS, SystemWriteIOPS, SystemReadBytes, SystemWriteBytes, SystemLatency, SystemReadLatency,
		SystemWriteLatency, SystemMetadata, SystemHealth, SystemResponse, SystemPhysicalTotalCapacity,
		SystemPhysicalUsedCapacity, SystemPhysicalFreeCapacity,
		PoolMetadata, PoolHealth, PoolWarningThreshold, PoolCapacityUsable, PoolCapacityUsed, PoolEfficiencySavings,
		PoolLogicalCapacityUsable, PoolLogicalCapacityUsed)

	if err != nil {
		t.Errorf("unexpected metrics:\n %s", err)
	}
}
