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
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "k8s.io/klog"

	"github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
)

type Pool map[string]interface{}

const (
	// Metric name defines
	PoolMetadata              = "flashsystem_pool_metadata"
	PoolHealth                = "flashsystem_pool_health"
	PoolWarningThreshold      = "flashsystem_capacity_warning_threshold"
	PoolCapacityUsable        = "flashsystem_pool_capacity_usable_bytes"
	PoolCapacityUsed          = "flashsystem_pool_capacity_used_bytes"
	PoolPhysicalCapacity      = "flashsystem_pool_capacity_bytes"
	PoolLogicalCapacityUsable = "flashsystem_pool_logical_capacity_usable_bytes"
	PoolLogicalCapacity       = "flashsystem_pool_logical_capacity_bytes"
	PoolLogicalCapacityUsed   = "flashsystem_pool_logical_capacity_used_bytes"
	PoolEfficiencySavings     = "flashsystem_pool_savings_bytes"
	// PoolEfficiencySavingsThin        = "flashsystem_pool_savings_thin_bytes"
	// PoolEfficiencySavingsDedup       = "flashsystem_pool_savings_dedup_bytes"
	// PoolEfficiencySavingsCompression = "flashsystem_pool_savings_compression_bytes"

	// Pool state
	StateOnline   = "online"
	StateDegraded = "degraded"
	StateOffline  = "offline"
)

// Interested keys
const (
	DataReductionKey           = "data_reduction"
	MdiskIdKey                 = "id"
	MdiskEffectiveUsedCapacity = "effective_used_capacity"
	ParentMdiskIdKey           = "parent_mdisk_grp_id"
	MdiskGroupNameKey          = "mdisk_grp_name"
	MdiskNameKey               = "name"
	PoolStatusKey              = "status"
	PhysicalFreeKey            = "physical_free_capacity"
	ReclaimableKey             = "reclaimable_capacity"
	PhysicalCapacityKey        = "physical_capacity"
	CompressionEnabledKey      = "compression_active"
	CapacityKey                = "capacity"
	FreeCapacityKey            = "free_capacity"
	ChildPoolCapacityKey       = "child_mdisk_grp_capacity"
	ControllerNameKey          = "controller_name"
	DiskModekey                = "mode"
	RealCapacityKey            = "real_capacity"
	UsedBeforeDedupKey         = "used_capacity_before_reduction"
	UsedAfterDedupKey          = "used_capacity_after_reduction"
	DedupSavingsKey            = "deduplication_capacity_saving"
	VirtualCapacityKey         = "virtual_capacity"
	UncompressedKey            = "compression_uncompressed_capacity"
	CompressedKey              = "compression_compressed_capacity"
	CapacityWarningThreshold   = "warning"
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
		PoolMetadata:              {"Pool metadata", poolMetadataLabel},
		PoolHealth:                {"Pool health status", poolLabelCommon},
		PoolWarningThreshold:      {"Pool capacity warning threshold", poolLabelCommon},
		PoolCapacityUsable:        {"Pool usable capacity (byte)", poolLabelCommon},
		PoolCapacityUsed:          {"Pool used capacity (byte)", poolLabelCommon},
		PoolPhysicalCapacity:      {"Pool total capacity (bytes)", poolLabelCommon},
		PoolLogicalCapacity:       {"Pool total logical capacity (byte)", poolLabelCommon},
		PoolLogicalCapacityUsable: {"Pool logical usable capacity (byte)", poolLabelCommon},
		PoolLogicalCapacityUsed:   {"Pool logical used capacity (byte)", poolLabelCommon},
		PoolEfficiencySavings:     {"dedupe, thin provisioning, and compression savings", poolLabelCommon},
		// PoolEfficiencySavingsThin:        {"thin provisioning savings", poolLabelCommon},
		// PoolEfficiencySavingsDedup:       {"dedeup savings", poolLabelCommon},
		// PoolEfficiencySavingsCompression: {"compression savings", poolLabelCommon},
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

func isPoolInternalMDisks(poolName string, disksList rest.MDisksList) (bool, error) {
	var internalPool bool
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			if disk[ControllerNameKey].(string) != "" {
				internalPool = false
			} else {
				internalPool = true
			}
		}
	}
	return internalPool, nil
}

func isPoolArrayMode(poolName string, disksList rest.MDisksList) (bool, error) {
	var arrayMode bool
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			if disk[DiskModekey].(string) != "array" {
				arrayMode = false
			} else {
				arrayMode = true
			}
		}
	}
	return arrayMode, nil
}

func (f *PerfCollector) CalcReducedReclaimableCapacityForPool(pool Pool, fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) (float64, error) {
	var totalDisksCapacities float64
	var midSum float64
	disksInPool := mapMDisksToPool(pool[MdiskNameKey].(string), mDisksList)
	reclaimable, err := strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
	if err != nil {
		log.Errorf("get pool reclaimable capacity failed: %s", err)
		return -1, err
	}

	for _, diskID := range disksInPool {
		PC, EU, physicalFree, err := calcCapacityForSingleDisk(fsRestClient, diskID)
		if err != nil {
			log.Errorf("get single disk capacity failed: %s", err)
			return -1, err
		}
		PU := math.Max(0, PC-physicalFree)
		diskRatio := PC * PU / EU

		totalDisksCapacities += PC
		midSum += diskRatio

		log.Infof("Disk ID: %d, PhysicalCapacity PC: %f, MdiskEffectiveUsedCapacity EU: %f, PU: %f, "+
			"diskRatio: %f, totalDisksCapacities: %f, midSum: %f",
			diskID, PC, EU, PU, diskRatio, totalDisksCapacities, midSum)
	}

	if totalDisksCapacities == 0 || midSum == 0 {
		return 0, nil
	} else {
		return reclaimable / totalDisksCapacities * midSum, nil
	}
}

func mapMDisksToPool(poolName string, disksList rest.MDisksList) []int {
	var disksInPool []int
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			diskId, _ := strconv.Atoi(disk[MdiskIdKey].(string))
			log.Infof("found disk name %s with ID %d", disk[MdiskGroupNameKey].(string), diskId)
			disksInPool = append(disksInPool, diskId)
		} else {
			continue
		}
	}
	return disksInPool
}

func calcCapacityForSingleDisk(fsRestClient *rest.FSRestClient, diskID int) (float64, float64, float64, error) {
	MDisksInfo, err := fsRestClient.LsSingleMDisk(diskID)
	if err != nil {
		log.Errorf("get disk list error: %v", err)
		return -1, -1, -1, err
	}

	PC, err := strconv.ParseFloat(MDisksInfo[PhysicalCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get disk physical capacity failed: %s", err)
		return -1, -1, -1, err
	}
	EU, err := strconv.ParseFloat(MDisksInfo[MdiskEffectiveUsedCapacity].(string), 64)
	if err != nil {
		if MDisksInfo[MdiskEffectiveUsedCapacity].(string) == "" {
			EU = 1
		} else {
			log.Errorf("get disk physical effective used capacity failed: %s", err)
			return -1, -1, -1, err
		}
	}

	physicalFree, err := strconv.ParseFloat(MDisksInfo[PhysicalFreeKey].(string), 64)
	if err != nil {
		log.Errorf("get disk physical free capacity failed: %s", err)
		return -1, -1, -1, err
	}

	return PC, EU, physicalFree, nil
}

func (f *PerfCollector) collectPoolMetrics(ch chan<- prometheus.Metric, fsRestClient *rest.FSRestClient) bool {
	// Get pool names
	manager := fsRestClient.DriverManager
	poolNames := manager.GetPoolNames()
	// log.Infof("pool count: %d, pools: %v", len(poolNames), poolNames)

	pools, err := fsRestClient.Lsmdiskgrp()
	if err != nil {
		log.Errorf("get pool list error: %v", err)
		return false
	}

	mDisksList, err := fsRestClient.LsAllMDisk()
	if err != nil {
		log.Errorf("get disk list error: %v", err)
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
			SystemName:               manager.GetSubsystemName(),
			PoolId:                   poolId,
			PoolName:                 poolName,
			State:                    pool[PoolStatusKey].(string),
			CapacityWarningThreshold: threshold,
			StorageClass:             strings.Join(scnames, ","),
		}
		// metadata metrics
		poolMetaMetricDesc := f.poolDescriptors[PoolMetadata]
		log.Infof("subsystem: %s, pool id: %d, name: %s, state: %s, sc: %s, warning: %s",
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

		log.Infof("pool id: %d, physical_free_capacity: %v, reclaimable_capacity: %v, data_reduction: %v, "+
			"physical_capacity: %v, virtual_capacity: %v, real_capacity: %v, logical_capacity: %v, logical_free_capacity: %v",
			poolInfo.PoolId, pool[PhysicalFreeKey], pool[ReclaimableKey], pool[DataReductionKey],
			pool[PhysicalCapacityKey], pool[VirtualCapacityKey], pool[RealCapacityKey], pool[CapacityKey], pool[FreeCapacityKey])

		createPhysicalCapacityPoolMetrics(ch, f, pool, poolInfo, fsRestClient, mDisksList)
		createLogicalCapacityPoolMetrics(ch, f, pool, poolInfo, mDisksList)
		createTotalSavingPoolMetrics(ch, f, pool, poolInfo)

		// pool_efficiency_savings_thin
		// drpSavings = Math.max(0, [lsmdiskgrp]:used_capacity_before_reduction – [lsmdiskgrp]:used_capacity_after_reduction + [lsmdiskgrp]:reclaimable_capacity – [lsmdiskgrp]:deduplication_capacity_saving)
		// drpCompressionSavings (for DRP) = drpSavings
		// For non-DRP, drpCompressionSavings = 0
		// Thin Provisioning Savings = Math.max(0, [lsmdiskgrp]:virtual_capacity – realCap – drpCompressionSavings – Math.max([lsmdiskgrp]:compression_uncompressed_capacity - [lsmdiskgrp]:compression_compressed_capacity, 0))
		// var drpCompressionSavings float64
		// usedBefore, err := strconv.ParseFloat(pool[UsedBeforeDedupKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get used_capacity_before_reduction failed: %s", err)
		// }
		// usedAfter, err := strconv.ParseFloat(pool[UsedAfterDedupKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get used_capacity_before_reduction failed: %s", err)
		// }
		// dedupSaving, err := strconv.ParseFloat(pool[DedupSavingsKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get deduplication_capacity_saving failed: %s", err)
		// }
		// tempValue := usedBefore - usedAfter + reclaimable - dedupSaving
		// drpSavings := math.Max(0, tempValue)

		// if drpool {
		// 	drpCompressionSavings = drpSavings
		// } else {
		// 	drpCompressionSavings = 0
		// }

		// uncompressed, err := strconv.ParseFloat(pool[UncompressedKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get compression_uncompressed_capacity failed: %s", err)
		// }
		// compressed, err := strconv.ParseFloat(pool[CompressedKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get compression_compressed_capacity failed: %s", err)
		// }
		// comressDiff := uncompressed - compressed
		// tempValue = virtual - realCapacity - drpCompressionSavings - math.Max(0, comressDiff)
		// thinSavings := math.Max(0, tempValue)

		// log.Infof("pool: %d, thin saving: %f", poolInfo.PoolId, thinSavings)
		// newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsThin], thinSavings, &poolInfo)

		// pool_efficiency_savings_dedup
		// [lsmdiskgrp]:deduplication_capacity_saving
		// dedupSavings, err := strconv.ParseFloat(pool[DedupSavingsKey].(string), 64)
		// if err != nil {
		// 	log.Errorf("get deduplication_capacity_saving failed: %s", err)
		// }
		// log.Infof("pool: %d, dedup saving: %f", poolInfo.PoolId, dedupSavings)
		// newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsDedup], dedupSavings, &poolInfo)

		// pool_efficiency_savings_compression
		// If pool is not compressed, Compression Savings = 0
		// For new compression (drp pool),
		// Compression Savings = Math.max(0, drpTotalWrittenCapacity – (drpTotalWrittenCapacity – drpSavings))
		//					   = drpSavings
		// For old compression (non-drp pool),
		// Compression Savings = Math.max(0, ([lsmdiskgrp]:compression_uncompressed_capacity – [lsmdiskgrp]:compression_compressed_capacity)
		// compressSavings := 0.0
		// if "yes" == pool[CompressionActiveKey].(string) {
		// 	if drpool {
		// 		compressSavings = drpSavings
		// 	} else {
		// 		compressSavings = math.Max(0, comressDiff)
		// 	}
		// }
		// log.Infof("pool: %d, compression saving: %f", poolInfo.PoolId, compressSavings)
		// newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsCompression], compressSavings, &poolInfo)
	}

	// Not found pool metrics
	for poolName, poolId := range poolNames {
		if driver.INIT_POOL_ID == poolId {
			scnames := manager.GetSCNameByPoolName(poolName)
			poolInfo := PoolInfo{
				SystemName:               fsRestClient.DriverManager.GetSubsystemName(),
				PoolId:                   poolId,
				PoolName:                 poolName,
				State:                    "NotFound",
				CapacityWarningThreshold: "100",
				StorageClass:             strings.Join(scnames, ","),
			}

			log.Infof("subsystem: %s, pool id: %d, name: %s, state: %s, sc: %s, warning: %s",
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

func isParentPool(pool Pool) bool {
	return pool[MdiskIdKey] == pool[ParentMdiskIdKey]
}

func createLogicalCapacityPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, pool Pool, poolInfo PoolInfo, mDisksList rest.MDisksList) {
	totalLogicalCapacity, err := strconv.ParseFloat(pool[CapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get logical capacity failed: %s", err)
		return
	}
	logicalFreeCapacity, err := strconv.ParseFloat(pool[FreeCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get logical free capacity failed: %s", err)
		return
	}

	var reclaimable float64
	internalPool, err := isPoolInternalMDisks(pool[MdiskNameKey].(string), mDisksList)
	if err != nil {
		log.Errorf("Failed to determine internal or external pool: %s", err)
		return
	}
	if !internalPool {
		reclaimable = 0
	} else {
		reclaimable, err = strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
			return
		}
	}

	//childPoolCapacity, err := strconv.ParseFloat(pool[ChildPoolCapacityKey].(string), 64)
	//if err != nil {
	//	log.Errorf("get Child Pool Capacity failed: %s", err)
	//	return
	//}

	logicalUsableCapacity := logicalFreeCapacity + reclaimable
	logicalUsedCapacity := totalLogicalCapacity - logicalUsableCapacity
	//logicalUsedCapacity := totalLogicalCapacity - logicalUsableCapacity - childPoolCapacity

	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacityUsable], logicalUsableCapacity, &poolInfo)
	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacityUsed], logicalUsedCapacity, &poolInfo)
	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacity], totalLogicalCapacity, &poolInfo)
}

func createPhysicalCapacityPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, pool Pool, poolInfo PoolInfo, fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) {
	if isParentPool(pool) {
		var reclaimableCalculatedCapacity float64
		physicalFree, err := strconv.ParseFloat(pool[PhysicalFreeKey].(string), 64)
		if err != nil {
			log.Errorf("get physical free failed: %s", err)
			return
		}
		physical, err := strconv.ParseFloat(pool[PhysicalCapacityKey].(string), 64)
		if err != nil {
			log.Errorf("get physical capacity failed: %s", err)
			return
		}
		poolOrigReclaimable, err := strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
			return
		}
		if poolOrigReclaimable != 0 {
			reclaimableCalculatedCapacity, err = f.GetPoolReclaimablePhysicalCapacity(pool, fsRestClient, mDisksList)
			if err != nil {
				log.Errorf("get reduced reclaimable capacity failed: %s", err)
				return
			}
		} else {
			reclaimableCalculatedCapacity = 0
		}

		used := physical - (physicalFree + reclaimableCalculatedCapacity)
		usable := physicalFree + poolOrigReclaimable

		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], usable, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsed], used, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolPhysicalCapacity], physical, &poolInfo)
	} else {
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], float64(-1), &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsed], float64(-1), &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolPhysicalCapacity], float64(-1), &poolInfo)
	}
}

func (f *PerfCollector) GetPoolReclaimablePhysicalCapacity(pool Pool, fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) (float64, error) {
	var reclaimable float64
	compressionActive := pool[CompressionEnabledKey].(string)
	dataReduction := pool[DataReductionKey].(string)

	internalPool, err := isPoolInternalMDisks(pool[MdiskNameKey].(string), mDisksList)
	if err != nil {
		log.Errorf("Failed to determine internal or external pool: %s", err)
		return reclaimable, err
	}
	arrayMode, err := isPoolArrayMode(pool[MdiskNameKey].(string), mDisksList)
	if err != nil {
		log.Errorf("Failed to determine disk mode: %s", err)
		return reclaimable, err
	}

	if compressionActive == "yes" && dataReduction == "yes" && internalPool && arrayMode {
		reclaimable, err = f.CalcReducedReclaimableCapacityForPool(pool, fsRestClient, mDisksList)
		if err != nil {
			log.Errorf("get reduced reclaimable capacity for pool failed")
			return reclaimable, err
		}
	} else if !internalPool {
		reclaimable = 0
	} else {
		poolOrigReclaimable, err := strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
			return reclaimable, err
		}
		reclaimable = poolOrigReclaimable
	}
	return reclaimable, nil
}

func createTotalSavingPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, pool Pool, poolInfo PoolInfo) {
	// TODO:ticket #42 - expose total saving per system

	drpool := pool[DataReductionKey].(string) == "yes"

	physicalFree := float64(0)
	physical := float64(0)

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

	if drpool {
		realCapacity = physical - physicalFree
	} else {
		realCapacity = realCap
	}

	totalSaving = math.Max(0, virtual-realCapacity)
	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavings], totalSaving, &poolInfo)
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
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.SystemName,
		fmt.Sprintf("%d", info.PoolId),
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
	val := 1.0
	if "online" == info.State {
		val = 0.0
	} else if "offline" == info.State {
		val = 2.0
	}
	if "online" != info.State {
		log.Infof("pool: %d state: %s", info.PoolId, info.State)
	}
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		val,
		info.SystemName,
		info.PoolName,
	)
}
