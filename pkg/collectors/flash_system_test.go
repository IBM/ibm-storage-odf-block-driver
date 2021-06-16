package collectors

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

func poster(req *http.Request, c *rest.FSRestClient) ([]byte, int, error) {
	path := fmt.Sprintf("%v", req.URL)
	body := ""
	switch path {
	case "/lssystem":
		body = `{"code_level": "8.4.0.2 (build 152.23.2102111856000)","product_name":"IBM SAN Volume Controller"}`
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

var client = &rest.FSRestClient{PostRequester: rest.NewRequester(poster)}
var testCollector, _ = NewPerfCollector(client, "FS-system-name", "FS-ns")

func TestMetrics(t *testing.T) {
	// Mock the dependency
	getPoolMap = func() (operutil.ScPoolMap, error) {
		poolMap := operutil.ScPoolMap{ScPool: map[string]string{}}
		poolMap.ScPool["fs-sc-default"] = "Pool0"
		poolMap.ScPool["fs-sc-1"] = "Pool0"
		poolMap.ScPool["fs-sc-2"] = "Pool1"
		poolMap.ScPool["fs-sc-3"] = "Pool1"
		poolMap.ScPool["fs-sc-4"] = "Pool2"

		return poolMap, nil
	}

	driver.CacheManager.Ready()

	expected := `
	# HELP flashsystem_pool_capacity_usable Pool usable capacity (Byte)
	# TYPE flashsystem_pool_capacity_usable gauge
	flashsystem_pool_capacity_usable{pool_name="Pool0",subsystem_name="FS-system-name"} 1.0798621523968e+13
	flashsystem_pool_capacity_usable{pool_name="Pool1",subsystem_name="FS-system-name"} 1.0798621523968e+13
	flashsystem_pool_capacity_usable{pool_name="Pool2",subsystem_name="FS-system-name"} 1.0798621523968e+13
	# HELP flashsystem_pool_capacity_used Pool used capacity (byte)
	# TYPE flashsystem_pool_capacity_used gauge
	flashsystem_pool_capacity_used{pool_name="Pool0",subsystem_name="FS-system-name"} 1.073741824e+09
	flashsystem_pool_capacity_used{pool_name="Pool1",subsystem_name="FS-system-name"} 1.073741824e+09
	flashsystem_pool_capacity_used{pool_name="Pool2",subsystem_name="FS-system-name"} 1.073741824e+09
	# HELP flashsystem_pool_capacity_warning_threshold Pool capacatity warning threshold
	# TYPE flashsystem_pool_capacity_warning_threshold gauge
	flashsystem_pool_capacity_warning_threshold{pool_name="Pool0",subsystem_name="FS-system-name"} 80
	flashsystem_pool_capacity_warning_threshold{pool_name="Pool1",subsystem_name="FS-system-name"} 80
	flashsystem_pool_capacity_warning_threshold{pool_name="Pool2",subsystem_name="FS-system-name"} 60
	# HELP flashsystem_pool_efficiency_savings dedupe, thin provisioning, and compression savings
	# TYPE flashsystem_pool_efficiency_savings gauge
	flashsystem_pool_efficiency_savings{pool_name="Pool0",subsystem_name="FS-system-name"} 2.064998802432e+12
	flashsystem_pool_efficiency_savings{pool_name="Pool1",subsystem_name="FS-system-name"} 1.0505892864e+10
	flashsystem_pool_efficiency_savings{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_pool_efficiency_savings_compression compression savings
	# TYPE flashsystem_pool_efficiency_savings_compression gauge
	flashsystem_pool_efficiency_savings_compression{pool_name="Pool0",subsystem_name="FS-system-name"} 0
	flashsystem_pool_efficiency_savings_compression{pool_name="Pool1",subsystem_name="FS-system-name"} 0
	flashsystem_pool_efficiency_savings_compression{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_pool_efficiency_savings_dedup dedeup savings
	# TYPE flashsystem_pool_efficiency_savings_dedup gauge
	flashsystem_pool_efficiency_savings_dedup{pool_name="Pool0",subsystem_name="FS-system-name"} 0
	flashsystem_pool_efficiency_savings_dedup{pool_name="Pool1",subsystem_name="FS-system-name"} 0
	flashsystem_pool_efficiency_savings_dedup{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_pool_efficiency_savings_thin thin provisioning savings
	# TYPE flashsystem_pool_efficiency_savings_thin gauge
	flashsystem_pool_efficiency_savings_thin{pool_name="Pool0",subsystem_name="FS-system-name"} 2.064998802432e+12
	flashsystem_pool_efficiency_savings_thin{pool_name="Pool1",subsystem_name="FS-system-name"} 1.0505892864e+10
	flashsystem_pool_efficiency_savings_thin{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_pool_health Pool health status
	# TYPE flashsystem_pool_health gauge
	flashsystem_pool_health{pool_name="Pool0",subsystem_name="FS-system-name"} 0
	flashsystem_pool_health{pool_name="Pool1",subsystem_name="FS-system-name"} 2
	flashsystem_pool_health{pool_name="Pool2",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_pool_metadata Pool metadata
	# TYPE flashsystem_pool_metadata gauge
	flashsystem_pool_metadata{pool_id="0",pool_name="Pool0",storageclass="fs-sc-1,fs-sc-default",subsystem_name="FS-system-name"} 0
	flashsystem_pool_metadata{pool_id="1",pool_name="Pool1",storageclass="fs-sc-2,fs-sc-3",subsystem_name="FS-system-name"} 0
	flashsystem_pool_metadata{pool_id="2",pool_name="Pool2",storageclass="fs-sc-4",subsystem_name="FS-system-name"} 0
	# HELP flashsystem_subsystem_health System health
	# TYPE flashsystem_subsystem_health gauge
	flashsystem_subsystem_health{subsystem_name="FS-system-name"} 0
	# HELP flashsystem_subsystem_metadata System information
	# TYPE flashsystem_subsystem_metadata gauge
	flashsystem_subsystem_metadata{model="SAN Volume Controller",subsystem_name="FS-system-name",vendor="IBM",version="8.4.0.2"} 0
	# HELP flashsystem_subsystem_rd_bytes overall performance - read throughput MB/s
	# TYPE flashsystem_subsystem_rd_bytes gauge
	flashsystem_subsystem_rd_bytes{subsystem_name="FS-system-name"} 0
	`

	err := testutil.CollectAndCompare(testCollector, strings.NewReader(expected),
		"flashsystem_pool_capacity_usable", "flashsystem_pool_capacity_used", "flashsystem_pool_capacity_warning_threshold", "flashsystem_pool_efficiency_savings",
		"flashsystem_pool_efficiency_savings_compression", "flashsystem_pool_efficiency_savings_dedup", "flashsystem_pool_efficiency_savings_thin", "flashsystem_pool_metadata",
		"flashsystem_subsystem_health", "flashsystem_subsystem_metadata", "flashsystem_subsystem_rd_bytes", "flashsystem_pool_health")

	if err != nil {
		t.Errorf("unexpected metrics:\n %s", err)
	}
}
