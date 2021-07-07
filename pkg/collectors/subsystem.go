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
	VdiskReadLatency  = "vdisk_r_ms"
	VdiskWriteLatency = "vdisk_w_ms"

	VersionKey = "code_level"
	ModelKey   = "product_name"

	// Metric name shown outside
	SystemReadIOPS     = "flashsystem_subsystem_rd_iops"
	SystemWriteIOPS    = "flashsystem_subsystem_wr_iops"
	SystemReadBytes    = "flashsystem_subsystem_rd_bytes"
	SystemWriteBytes   = "flashsystem_subsystem_wr_bytes"
	SystemReadLatency  = "flashsystem_subsystem_rd_latency_seconds"
	SystemWriteLatency = "flashsystem_subsystem_wr_latency_seconds"

	SystemMetadata = "flashsystem_subsystem_metadata"
	SystemHealth   = "flashsystem_subsystem_health"
)

var (
	// Metadata label
	subsystemMetadataLabel = []string{"subsystem_name", "vendor", "model", "version"}

	// Other label
	subsystemCommonLabel = []string{"subsystem_name"}

	systemMetricsMap = map[string]MetricLabel{
		SystemMetadata: {"System information", subsystemMetadataLabel},
		SystemHealth:   {"System health", subsystemCommonLabel},
	}

	perfMetricsMap = map[string]MetricLabel{
		SystemReadIOPS:     {"overall performance - read IOPS", subsystemCommonLabel},
		SystemWriteIOPS:    {"overall performance - write IOPS", subsystemCommonLabel},
		SystemReadBytes:    {"overall performance - read throughput bytes/s", subsystemCommonLabel},
		SystemWriteBytes:   {"overall performance - write throughput bytes/s", subsystemCommonLabel},
		SystemReadLatency:  {"overall performance - read latency seconds", subsystemCommonLabel},
		SystemWriteLatency: {"overall performance - write latency seconds", subsystemCommonLabel},
	}

	// Raw metrics names to system metrics name map
	rawMetricsMap = map[string]string{
		VdiskReadBW:       SystemReadBytes,
		VdiskWriteBW:      SystemWriteBytes,
		VdiskReadIOPS:     SystemReadIOPS,
		VdiskWriteIOPS:    SystemWriteIOPS,
		VdiskReadLatency:  SystemReadLatency,
		VdiskWriteLatency: SystemWriteLatency,
	}

	// Unit conversion for raw metrics
	unitConvertMap = map[string]float64{
		VdiskReadBW:       1024 * 1024,
		VdiskWriteBW:      1024 * 1024,
		VdiskReadIOPS:     1,
		VdiskWriteIOPS:    1,
		VdiskReadLatency:  0.001,
		VdiskWriteLatency: 0.001,
	}
)

type SystemInfo struct {
	Name    string
	Vendor  string
	Model   string
	Version string
}

type SystemName struct {
	Name string
}

func (f *PerfCollector) initSubsystemDescs() {
	f.sysInfoDescriptors = make(map[string]*prometheus.Desc)
	f.sysPerfDescriptors = make(map[string]*prometheus.Desc)

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
}

func (f *PerfCollector) collectSystemMetrics(ch chan<- prometheus.Metric) bool {

	// timer := prometheus.NewTimer(f.scrapeDuration)
	// defer timer.ObserveDuration()

	// f.totalScrapes.Inc()
	f.sequenceNumber++

	var statsResults rest.SystemStats
	var sysInfoResults rest.StorageSystem
	var systemInfo SystemInfo
	var systemName SystemName
	var err error

	// Subsystem name is from CR
	systemName.Name = f.systemName

	// Get flash system results
	statsResults, err = f.client.Lssystemstats()
	if err == nil {
		sysInfoResults, err = f.client.Lssystem()
	}
	if err != nil {
		f.up.Set(0)
		log.Errorf("fail metrics pulling in round %d", f.sequenceNumber)
		return false
	} else {
		f.up.Set(1)
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
	systemInfo.Name = f.systemName

	newSystemMetrics(ch, f.sysInfoDescriptors[SystemMetadata], 0, &systemInfo)
	// Determine the health 0 = OK, 1 = warning, 2 = error
	bReady, err := f.client.CheckFlashsystemClusterState()
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
			log.Fatalf("metric mapping wrong: %s", metricName)
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

func newSystemMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *SystemInfo) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.Name,
		info.Vendor,
		info.Model,
		info.Version,
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
