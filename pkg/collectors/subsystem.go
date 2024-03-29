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
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "k8s.io/klog"

	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
)

const (
	// Column name for performance metrics
	VdiskReadBW       = "vdisk_r_mb"
	VdiskWriteBW      = "vdisk_w_mb"
	VdiskReadIOPS     = "vdisk_r_io"
	VdiskWriteIOPS    = "vdisk_w_io"
	VdiskLatency      = "vdisk_ms"
	VdiskReadLatency  = "vdisk_r_ms"
	VdiskWriteLatency = "vdisk_w_ms"

	VersionKey               = "code_level"
	ModelKey                 = "product_name"
	PhysicalTotalCapacityKey = "physical_capacity"
	PhysicalFreeCapacityKey  = "physical_free_capacity"
	//ReclaimableCapacity      = "total_reclaimable_capacity"

	// Metric name shown outside
	SystemReadIOPS     = "flashsystem_subsystem_rd_iops"
	SystemWriteIOPS    = "flashsystem_subsystem_wr_iops"
	SystemReadBytes    = "flashsystem_subsystem_rd_bytes"
	SystemWriteBytes   = "flashsystem_subsystem_wr_bytes"
	SystemLatency      = "flashsystem_subsystem_latency_seconds"
	SystemReadLatency  = "flashsystem_subsystem_rd_latency_seconds"
	SystemWriteLatency = "flashsystem_subsystem_wr_latency_seconds"

	SystemMetadata = "flashsystem_subsystem_metadata"
	SystemHealth   = "flashsystem_subsystem_health"
	SystemResponse = "flashsystem_subsystem_response"

	SystemPhysicalTotalCapacity = "flashsystem_subsystem_physical_total_capacity_bytes"
	SystemPhysicalFreeCapacity  = "flashsystem_subsystem_physical_free_capacity_bytes"
	SystemPhysicalUsedCapacity  = "flashsystem_subsystem_physical_used_capacity_bytes"
)

var (
	// Metadata label
	subsystemMetadataLabel = []string{"subsystem_name", "vendor", "model", "version", "is_internal_storage"}

	// Other label
	subsystemCommonLabel = []string{"subsystem_name"}

	systemMetricsMap = map[string]MetricLabel{
		SystemMetadata: {"System information", subsystemMetadataLabel},
		SystemHealth:   {"System health", subsystemCommonLabel},
		SystemResponse: {"System response", subsystemMetadataLabel},
	}

	perfMetricsMap = map[string]MetricLabel{
		SystemReadIOPS:     {"overall performance - read IOPS", subsystemCommonLabel},
		SystemWriteIOPS:    {"overall performance - write IOPS", subsystemCommonLabel},
		SystemReadBytes:    {"overall performance - read throughput bytes/s", subsystemCommonLabel},
		SystemWriteBytes:   {"overall performance - write throughput bytes/s", subsystemCommonLabel},
		SystemLatency:      {"overall performance - average latency seconds", subsystemCommonLabel},
		SystemReadLatency:  {"overall performance - read latency seconds", subsystemCommonLabel},
		SystemWriteLatency: {"overall performance - write latency seconds", subsystemCommonLabel},
	}

	// Raw metrics names to system metrics name map
	rawMetricsMap = map[string]string{
		VdiskReadBW:       SystemReadBytes,
		VdiskWriteBW:      SystemWriteBytes,
		VdiskReadIOPS:     SystemReadIOPS,
		VdiskWriteIOPS:    SystemWriteIOPS,
		VdiskLatency:      SystemLatency,
		VdiskReadLatency:  SystemReadLatency,
		VdiskWriteLatency: SystemWriteLatency,
	}

	// StorageSystemMetricsMap defines mapping
	StorageSystemMetricsMap = map[string]MetricLabel{
		SystemPhysicalTotalCapacity: {"System physical total capacity (byte)", subsystemCommonLabel},
		SystemPhysicalFreeCapacity:  {"System physical free capacity (byte)", subsystemCommonLabel},
		SystemPhysicalUsedCapacity:  {"System physical used capacity (byte)", subsystemCommonLabel},
	}

	// Unit conversion for raw metrics
	unitConvertMap = map[string]float64{
		VdiskReadBW:       1024 * 1024,
		VdiskWriteBW:      1024 * 1024,
		VdiskReadIOPS:     1,
		VdiskWriteIOPS:    1,
		VdiskLatency:      0.001,
		VdiskReadLatency:  0.001,
		VdiskWriteLatency: 0.001,
	}
)

type SystemInfo struct {
	Name              string
	Vendor            string
	Model             string
	Version           string
	isInternalStorage int
}

type SystemName struct {
	Name string
}

func (f *PerfCollector) initSubsystemDescs() {
	f.sysInfoDescriptors = make(map[string]*prometheus.Desc)
	f.sysPerfDescriptors = make(map[string]*prometheus.Desc)
	f.sysCapacityDescriptors = make(map[string]*prometheus.Desc)

	for metricName, metricLabel := range systemMetricsMap {
		f.sysInfoDescriptors[metricName] = prometheus.NewDesc(
			metricName,
			metricLabel.Name, metricLabel.Labels, nil,
		)
	}

	for metricName, metricLabel := range perfMetricsMap {
		f.sysPerfDescriptors[metricName] = prometheus.NewDesc(
			metricName,
			metricLabel.Name, metricLabel.Labels, nil,
		)
	}

	for metricName, metricLabel := range StorageSystemMetricsMap {
		f.sysCapacityDescriptors[metricName] = prometheus.NewDesc(
			metricName,
			metricLabel.Name, metricLabel.Labels, nil,
		)
	}
}

func (f *PerfCollector) collectSystemMetrics(ch chan<- prometheus.Metric, fsRestClient *rest.FSRestClient, poolsInfoList []PoolInfo) bool {

	// timer := prometheus.NewTimer(f.scrapeDuration)
	// defer timer.ObserveDuration()

	// f.totalScrapes.Inc()

	var statsResults rest.SystemStats
	var sysInfoResults rest.StorageSystem
	var systemInfo SystemInfo
	var systemName SystemName
	var err error

	manager := fsRestClient.DriverManager
	// Subsystem name is from CR
	systemName.Name = manager.GetSubsystemName()
	systemInfo.Name = manager.GetSubsystemName()

	// Get flash system results
	statsResults, err = fsRestClient.Lssystemstats()
	if err == nil {
		sysInfoResults, err = fsRestClient.Lssystem()
	}
	if err != nil {
		newSystemMetrics(ch, f.sysInfoDescriptors[SystemResponse], 0, &systemInfo)
		log.Error("fail to get system stats")
		return false
	} else {
		newSystemMetrics(ch, f.sysInfoDescriptors[SystemResponse], 1, &systemInfo)
	}

	// code level example: 8.3.1.2 (build 150.24.2008101830000)
	version := sysInfoResults[VersionKey].(string)
	versions := strings.Split(version, " ")
	systemInfo.Version = versions[0]

	// product_name: IBM FlashSystem 9200
	productStr := sysInfoResults[ModelKey].(string)
	names := strings.Split(productStr, " ")
	systemInfo.Vendor = names[0]
	model := strings.TrimPrefix(productStr, names[0])
	systemInfo.Model = strings.TrimSpace(model)
	systemInfo.isInternalStorage = isAllInternalStorage(poolsInfoList)
	newSystemMetrics(ch, f.sysInfoDescriptors[SystemMetadata], 0, &systemInfo)

	f.createSystemPhysicalCapacityMetrics(ch, sysInfoResults, systemName, poolsInfoList)

	// Determine the health 0 = OK, 1 = warning, 2 = error
	bReady, err := fsRestClient.CheckFlashsystemClusterState()
	status := 0.0
	if err != nil || !bReady {
		status = 1
	}
	newPerfMetrics(ch, f.sysInfoDescriptors[SystemHealth], status, &systemName)

	// Parse statsResults
	for _, m := range statsResults {
		metricName, ok := m["stat_name"]
		if !ok {
			log.Warningf("no stat_name in metric response: %v", m)
			continue
		}

		// Get metric descriptor name from rawMetricsMap
		metricDescName, ok := rawMetricsMap[metricName]
		if !ok {
			// log.Warningf("Not interested metric for %s", metricName)
			continue
		}

		// Get metric descriptor from sysPerfDescriptors
		metricDesc, ok := f.sysPerfDescriptors[metricDescName]
		if !ok {
			log.Errorf("metric mapping wrong: %s", metricName)
			continue
		}

		metricValue, err := strconv.ParseFloat(m["stat_current"], 64)
		if err != nil {
			log.Errorf("fail to convert metric %s from string %s to float", metricName, m["stat_current"])
			continue
		}

		convertFactor, ok := unitConvertMap[metricName]
		if ok {
			metricValue *= convertFactor
		}

		newPerfMetrics(ch, metricDesc, metricValue, &systemName)
	}

	return true
}

func (f *PerfCollector) createSystemPhysicalCapacityMetrics(ch chan<- prometheus.Metric, sysInfoResults rest.StorageSystem,
	systemName SystemName, poolsInfoList []PoolInfo) {
	// [lssystem]: physical_capacity
	physicalTotalCapacity, err := strconv.ParseFloat(sysInfoResults[PhysicalTotalCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get system physical total capacity failed: %s", err)
		return
	}
	// [lssystem]: physical_free_capacity
	physicalUsableCapacity, err := strconv.ParseFloat(sysInfoResults[PhysicalFreeCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get system physical usable capacity failed: %s", err)
		return
	}
	physicalReclaimableCapacity, err := calcSystemReclaimableCapacity(poolsInfoList)
	//physicalReclaimableCapacity, err := strconv.ParseFloat(sysInfoResults[ReclaimableCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get system physical reclaimable capacity failed: %s", err)
		return
	}
	physicalUsedCapacity := physicalTotalCapacity - physicalUsableCapacity - physicalReclaimableCapacity

	physicalFreeCapacity := physicalTotalCapacity - physicalUsedCapacity
	// used = total - free
	log.Infof("system capacity total: %f, free: %f, used: %f", physicalTotalCapacity, physicalFreeCapacity, physicalUsedCapacity)

	newSystemCapacityMetrics(ch, f.sysCapacityDescriptors[SystemPhysicalTotalCapacity], physicalTotalCapacity, &systemName)
	newSystemCapacityMetrics(ch, f.sysCapacityDescriptors[SystemPhysicalUsedCapacity], physicalUsedCapacity, &systemName)
	newSystemCapacityMetrics(ch, f.sysCapacityDescriptors[SystemPhysicalFreeCapacity], physicalFreeCapacity, &systemName)
}

func isAllInternalStorage(poolsInfoList []PoolInfo) int {
	for _, pool := range poolsInfoList {
		if !pool.IsInternalStorage {
			return 0
		}
	}
	return 1
}

func calcSystemReclaimableCapacity(poolsInfoList []PoolInfo) (float64, error) {
	var totalSystemReclaimable float64
	for _, currentPool := range poolsInfoList {
		poolReclaimable, err := GetPoolReclaimablePhysicalCapacity(currentPool)
		if err != nil {
			log.Errorf("get pool reclaimable physical capacity failed: %v", err)
			return InvalidVal, err
		}
		totalSystemReclaimable += poolReclaimable
	}
	return totalSystemReclaimable, nil
}

func newSystemMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *SystemInfo) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.Name,
		info.Vendor,
		info.Model,
		info.Version,
		fmt.Sprintf("%d", info.isInternalStorage),
	)
}

func newPerfMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, systemName *SystemName) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		systemName.Name,
	)
}

func newSystemCapacityMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, systemName *SystemName) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		systemName.Name,
	)
}
