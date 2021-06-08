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
package collectors

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
)

const (
	// Metric name defines
	PoolMetadata                     = "flashsystem_pool_metadata"
	PoolHealth                       = "flashsystem_pool_health"
	PoolWarningThreshold             = "flashsystem_pool_capacity_warning_threshold"
	PoolCapacityUsable               = "flashsystem_pool_capacity_usable"
	PoolCapacityUsed                 = "flashsystem_pool_capacity_used"
	PoolEfficiencySavings            = "flashsystem_pool_efficiency_savings"
	PoolEfficiencySavingsThin        = "flashsystem_pool_efficiency_savings_thin"
	PoolEfficiencySavingsDedup       = "flashsystem_pool_efficiency_savings_dedup"
	PoolEfficiencySavingsCompression = "flashsystem_pool_efficiency_savings_compression"

	// Pool state
	StateOnline   = "online"
	StateDegraded = "degraded"
	StateOffline  = "offline"
)

// Interested keys
const (
	DataReductionKey         = "data_reduction"
	MdiskIdKey               = "id"
	MdiskNameKey             = "name"
	PoolStatusKey            = "status"
	PhysicalFreeKey          = "physical_free_capacity"
	ReclaimableKey           = "reclaimable_capacity"
	PhysicalCapacityKey      = "physical_capacity"
	FreeCapacityKey          = "free_capacity"
	ChildPoolCapacityKey     = "child_mdisk_grp_capacity"
	RealCapacityKey          = "real_capacity"
	UsedBeforeDedupKey       = "used_capacity_before_reduction"
	UsedAfterDedupKey        = "used_capacity_after_reduction"
	DedupSavingsKey          = "deduplication_capacity_saving"
	VirtualCapacityKey       = "virtual_capacity"
	UncompressedKey          = "compression_uncompressed_capacity"
	CompressedKey            = "compression_compressed_capacity"
	CompressionActiveKey     = "compression_active"
	CapacityWarningThreshold = "warning"
)

var (
	// Pool Metadata label
	poolMetadataLabel = []string{
		"subsystem_name",
		"pool_id",
		"pool_name",
		"storageclass",
	}

	// Other metrics label
	poolLabelCommon = []string{
		"subsystem_name",
		"pool_name",
	}

	// Metric define mapping
	poolMetricsMap = map[string]MetricLabel{
		PoolMetadata:                     {"Pool metadata", poolMetadataLabel},
		PoolHealth:                       {"Pool health status", poolLabelCommon},
		PoolWarningThreshold:             {"Pool capacatity warning threshold", poolLabelCommon},
		PoolCapacityUsable:               {"Pool usable capacity (Byte)", poolLabelCommon},
		PoolCapacityUsed:                 {"Pool used capacity (byte)", poolLabelCommon},
		PoolEfficiencySavings:            {"dedupe, thin provisioning, and compression savings", poolLabelCommon},
		PoolEfficiencySavingsThin:        {"thin provisioning savings", poolLabelCommon},
		PoolEfficiencySavingsDedup:       {"dedeup savings", poolLabelCommon},
		PoolEfficiencySavingsCompression: {"compression savings", poolLabelCommon},
	}
)

type PoolInfo struct {
	SystemName               string
	PoolId                   int
	PoolName                 string
	StorageClass             string
	State                    string
	CapacityWarningThreshold string
}

func (f *PerfCollector) initPoolDescs() {
	f.poolDescriptors = make(map[string]*prometheus.Desc)

	for metricName, metricLabel := range poolMetricsMap {
		f.poolDescriptors[metricName] = prometheus.NewDesc(
			metricName,
			metricLabel.Name, metricLabel.Labels, nil,
		)
	}
}

func (f *PerfCollector) collectPoolMetrics(ch chan<- prometheus.Metric) bool {
	// Get pool names
	manager, err := driver.GetManager()
	if err != nil {
		log.Errorf("get driver manager error: %s", err)
		return false
	}
	poolNames := manager.GetPoolNames()
	log.Infof("pool count: %d, pools: %v", len(poolNames), poolNames)

	pools, err := f.client.Lsmdiskgrp()
	if err != nil {
		log.Errorf("get pool list error: %v", err)
		return false
	}

	// Pool metrics
	for _, pool := range pools {
		poolId, _ := strconv.Atoi(pool[MdiskIdKey].(string))
		poolName := pool[MdiskNameKey].(string)
		if _, bHas := poolNames[poolName]; bHas {
			poolNames[poolName] = poolId
		} else {
			// Skip. Not used in StorageClass
			continue
		}

		scnames := manager.GetSCNameByPoolName(poolName)
		sort.Strings(scnames) // For testing to get unique value
		threshold := pool[CapacityWarningThreshold].(string)
		// The 0 means turn off the warning
		if threshold == "0" {
			threshold = "100"
		}
		poolInfo := PoolInfo{
			SystemName:               f.systemName,
			PoolId:                   poolId,
			PoolName:                 poolName,
			State:                    pool[PoolStatusKey].(string),
			CapacityWarningThreshold: threshold,
			StorageClass:             strings.Join(scnames, ","),
		}
		// metadata metrics
		poolMetaMetricDesc := f.poolDescriptors[PoolMetadata]
		log.Infof("subsystem: %s, pool id: %d, pool name: %s, state: %s, sc: %s, warning: %s",
			poolInfo.SystemName,
			poolInfo.PoolId,
			poolInfo.PoolName,
			poolInfo.State,
			poolInfo.StorageClass,
			poolInfo.CapacityWarningThreshold,
		)
		newPoolMetadataMetrics(ch, poolMetaMetricDesc, 0, &poolInfo)
		f.newPoolWarningThreshold(ch, &poolInfo)
		f.newPoolHealthMetrics(ch, &poolInfo)

		// Get pool 'data_reduction' as true/false
		drpool := "yes" == pool[DataReductionKey].(string)

		// pool_capacity_usable
		// [lsmdiskgrp]:physical_free_capacity + [lsmdiskgrp]:reclaimable_capacity
		physicalFree, err := strconv.ParseFloat(pool[PhysicalFreeKey].(string), 64)
		if err != nil {
			log.Errorf("get physical free failed: %s", err)
		}
		reclaimable, err := strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
		}
		usable := physicalFree + reclaimable
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], usable, &poolInfo)
		log.Infof("pool: %d capacity usable: %f", poolInfo.PoolId, usable)

		// pool_capacity_used
		// [lsmdiskgrp]:physical_capacity - pool_capacity_usable
		physical, err := strconv.ParseFloat(pool[PhysicalCapacityKey].(string), 64)
		if err != nil {
			log.Errorf("get physical capacity failed: %s", err)
		}
		used := physical - usable
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsed], used, &poolInfo)
		log.Infof("pool: %d, capacity used: %f", poolInfo.PoolId, used)

		// pool_efficiency_savings
		// virtualCap = [lsmdiskgrp]:virtual_capacity
		// realCap = [lsmdiskgrp]:physical_capacity – [lsmdiskgrp]:physical_free_capacity … for DRP
		//         = [lsmdiskgrp]:real_capacity … for non-DRP
		// Total Savings = Math.max(0, virtualCap – realCap)
		var totalSaving float64
		var realCapacity float64
		virtual, err := strconv.ParseFloat(pool[VirtualCapacityKey].(string), 64)
		if err != nil {
			log.Errorf("get virtual capacity failed: %s", err)
		}
		realCap, err := strconv.ParseFloat(pool[RealCapacityKey].(string), 64)
		if err != nil {
			log.Errorf("get real capacity failed: %s", err)
		}
		phyFree, err := strconv.ParseFloat(pool[PhysicalFreeKey].(string), 64)
		if err != nil {
			log.Errorf("get physical free capacity failed: %s", err)
		}
		if drpool {
			realCapacity = physical - phyFree
		} else {
			realCapacity = realCap
		}
		totalSaving = math.Max(0, virtual-realCapacity)
		log.Infof("pool: %d, total saving: %f", poolInfo.PoolId, totalSaving)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavings], totalSaving, &poolInfo)

		// pool_efficiency_savings_thin
		// drpSavings = Math.max(0, [lsmdiskgrp]:used_capacity_before_reduction – [lsmdiskgrp]:used_capacity_after_reduction + [lsmdiskgrp]:reclaimable_capacity – [lsmdiskgrp]:deduplication_capacity_saving)
		// drpCompressionSavings (for DRP) = drpSavings
		// For non-DRP, drpCompressionSavings = 0
		// Thin Provisioning Savings = Math.max(0, [lsmdiskgrp]:virtual_capacity – realCap – drpCompressionSavings – Math.max([lsmdiskgrp]:compression_uncompressed_capacity - [lsmdiskgrp]:compression_compressed_capacity, 0))
		var drpCompressionSavings float64
		usedBefore, err := strconv.ParseFloat(pool[UsedBeforeDedupKey].(string), 64)
		if err != nil {
			log.Errorf("get used_capacity_before_reduction failed: %s", err)
		}
		usedAfter, err := strconv.ParseFloat(pool[UsedAfterDedupKey].(string), 64)
		if err != nil {
			log.Errorf("get used_capacity_before_reduction failed: %s", err)
		}
		dedupSaving, err := strconv.ParseFloat(pool[DedupSavingsKey].(string), 64)
		if err != nil {
			log.Errorf("get deduplication_capacity_saving failed: %s", err)
		}
		tempValue := usedBefore - usedAfter + reclaimable - dedupSaving
		drpSavings := math.Max(0, tempValue)

		if drpool {
			drpCompressionSavings = drpSavings
		} else {
			drpCompressionSavings = 0
		}

		uncompressed, err := strconv.ParseFloat(pool[UncompressedKey].(string), 64)
		if err != nil {
			log.Errorf("get compression_uncompressed_capacity failed: %s", err)
		}
		compressed, err := strconv.ParseFloat(pool[CompressedKey].(string), 64)
		if err != nil {
			log.Errorf("get compression_compressed_capacity failed: %s", err)
		}
		comressDiff := uncompressed - compressed
		tempValue = virtual - realCapacity - drpCompressionSavings - math.Max(0, comressDiff)
		thinSavings := math.Max(0, tempValue)

		log.Infof("pool: %d, thin saving: %f", poolInfo.PoolId, thinSavings)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsThin], totalSaving, &poolInfo)

		// pool_efficiency_savings_dedup
		// [lsmdiskgrp]:deduplication_capacity_saving
		dedupSavings, err := strconv.ParseFloat(pool[DedupSavingsKey].(string), 64)
		if err != nil {
			log.Errorf("get deduplication_capacity_saving failed: %s", err)
		}
		log.Infof("pool: %d, dedup saving: %f", poolInfo.PoolId, dedupSavings)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsDedup], dedupSavings, &poolInfo)

		// pool_efficiency_savings_compression
		// If pool is not compressed, Compression Savings = 0
		// For new compression (drp pool),
		// Compression Savings = Math.max(0, drpTotalWrittenCapacity – (drpTotalWrittenCapacity – drpSavings))
		//					   = drpSavings
		// For old compression (non-drp pool),
		// Compression Savings = Math.max(0, ([lsmdiskgrp]:compression_uncompressed_capacity – [lsmdiskgrp]:compression_compressed_capacity)
		compressSavings := 0.0
		if "yes" == pool[CompressionActiveKey].(string) {
			if drpool {
				compressSavings = drpSavings
			} else {
				compressSavings = math.Max(0, comressDiff)
			}
		}
		log.Infof("pool: %d, compression saving: %f", poolInfo.PoolId, compressSavings)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsCompression], compressSavings, &poolInfo)
	}

	// Not found pool metrics
	for poolName, poolId := range poolNames {
		if poolId == driver.INIT_POOL_ID {
			scnames := manager.GetSCNameByPoolName(poolName)
			poolInfo := PoolInfo{
				SystemName:               f.systemName,
				PoolId:                   poolId,
				PoolName:                 poolName,
				State:                    "NotFound",
				CapacityWarningThreshold: "100",
				StorageClass:             strings.Join(scnames, ","),
			}

			log.Infof("subsystem: %s, pool id: %d, pool name: %s, state: %s, sc: %s, warning: %s",
				poolInfo.SystemName,
				poolInfo.PoolId,
				poolInfo.PoolName,
				poolInfo.State,
				poolInfo.StorageClass,
				poolInfo.CapacityWarningThreshold,
			)
			// poolMetaMetricDesc := f.poolDescriptors[PoolMetadata]
			// newPoolMetadataMetrics(ch, poolMetaMetricDesc, 0, &poolInfo)
		}
	}

	return true
}

func newPoolCapacityMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *PoolInfo) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.SystemName,
		info.PoolName,
	)
}

func newPoolMetadataMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *PoolInfo) {
	strPoolId := fmt.Sprintf("%d", info.PoolId)

	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.SystemName,
		strPoolId,
		info.PoolName,
		info.StorageClass,
	)
}

func (f *PerfCollector) newPoolWarningThreshold(ch chan<- prometheus.Metric, info *PoolInfo) {
	desc := f.poolDescriptors[PoolWarningThreshold]
	val, err := strconv.Atoi(info.CapacityWarningThreshold)
	if err != nil {
		val = 100
	}
	// No value set it to 100% to turn off the check
	if val == 0 {
		val = 100
	}
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		float64(val),
		info.SystemName,
		info.PoolName,
	)
}

func (f *PerfCollector) newPoolHealthMetrics(ch chan<- prometheus.Metric, info *PoolInfo) {
	desc := f.poolDescriptors[PoolHealth]
	val := 2.0
	if "online" == info.State {
		val = 0.0
	} else if "offline" == info.State {
		val = 1.0
	}
	log.Infof("pool: %d state: %s", info.PoolId, info.State)
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		val,
		info.SystemName,
		info.PoolName,
	)
}
