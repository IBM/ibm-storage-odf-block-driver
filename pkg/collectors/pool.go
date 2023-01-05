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

	InvalidVal = float64(-1)
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
	OverProvisionedKey         = "over_provisioned"
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
		"is_internal_storage",
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
	InternalStorage          bool
	ArrayMode                bool
	PoolMDiskgrpInfo         Pool
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

func IsPoolFromInternalStorage(poolName string, disksList rest.MDisksList) bool {
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			if disk[ControllerNameKey].(string) != "" {
				return false
			}
		}
	}
	return true
}

func isCompressionEnabled(poolName string, disksList rest.MDisksList, fsRestClient *rest.FSRestClient) (bool, error) {
	disksInPool := listMDisksInPool(poolName, disksList)

	for _, diskID := range disksInPool {
		MDiskInfo, err := fsRestClient.LsSingleMDisk(diskID)
		if err != nil {
			log.Errorf("get single mdisk info error: %v", err)
			return false, err
		}
		if MDiskInfo[MdiskEffectiveUsedCapacity].(string) == "" {
			return false, nil
		}
	}
	return true, nil
}

func isPoolArrayMode(poolName string, disksList rest.MDisksList) bool {
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			if disk[DiskModekey].(string) != "array" {
				return false
			}
		}
	}
	return true
}

func calcReducedReclaimableCapacityForPool(pool Pool, fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) (float64, error) {
	var totalDisksCapacities float64
	var midSum float64
	disksInPool := listMDisksInPool(pool[MdiskNameKey].(string), mDisksList)
	reclaimable, err := strconv.ParseFloat(pool[ReclaimableKey].(string), 64)
	if err != nil {
		log.Errorf("get pool reclaimable capacity failed: %s", err)
		return InvalidVal, err
	}

	for _, diskID := range disksInPool {
		PC, EU, physicalFree, err := calcCapacityForSingleDisk(fsRestClient, diskID)
		if err != nil {
			log.Errorf("get single disk capacity failed: %s", err)
			return InvalidVal, err
		}
		PU := math.Max(0, PC-physicalFree)
		diskRatio := PC * PU / EU

		totalDisksCapacities += PC
		midSum += diskRatio

		log.Infof("Calculating reduced reclaimable capacity for Disk ID: %d related to pool %v, "+
			"PhysicalCapacity PC: %f, MdiskEffectiveUsedCapacity EU: %f, PU: %f, diskRatio: %f, totalDisksCapacities: %f, midSum: %f",
			diskID, pool[MdiskNameKey].(string), PC, EU, PU, diskRatio, totalDisksCapacities, midSum)
	}

	if totalDisksCapacities == 0 || midSum == 0 {
		return 0, nil
	} else {
		return (reclaimable / totalDisksCapacities) * midSum, nil
	}
}

func listMDisksInPool(poolName string, disksList rest.MDisksList) []int {
	var disksInPool []int
	for _, disk := range disksList {
		if poolName == disk[MdiskGroupNameKey].(string) {
			diskId, _ := strconv.Atoi(disk[MdiskIdKey].(string))
			disksInPool = append(disksInPool, diskId)
		}
	}
	return disksInPool
}

func calcCapacityForSingleDisk(fsRestClient *rest.FSRestClient, diskID int) (float64, float64, float64, error) {
	MDisksInfo, err := fsRestClient.LsSingleMDisk(diskID)
	if err != nil {
		log.Errorf("get disk list error: %v", err)
		return InvalidVal, InvalidVal, InvalidVal, err
	}

	PC, err := strconv.ParseFloat(MDisksInfo[PhysicalCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get disk physical capacity failed: %s", err)
		return InvalidVal, InvalidVal, InvalidVal, err
	}
	physicalFree, err := strconv.ParseFloat(MDisksInfo[PhysicalFreeKey].(string), 64)
	if err != nil {
		log.Errorf("get disk physical free capacity failed: %s", err)
		return InvalidVal, InvalidVal, InvalidVal, err
	}

	EU, err := strconv.ParseFloat(MDisksInfo[MdiskEffectiveUsedCapacity].(string), 64)
	if err != nil {
		if MDisksInfo[MdiskEffectiveUsedCapacity].(string) == "" { // can happen only on drives without compression
			EU = PC - physicalFree
		} else {
			log.Errorf("get disk physical effective used capacity failed: %s", err)
			return InvalidVal, InvalidVal, InvalidVal, err
		}
	}

	return PC, EU, physicalFree, nil
}

func (f *PerfCollector) collectPoolMetrics(ch chan<- prometheus.Metric, fsRestClient *rest.FSRestClient,
	mDisksList rest.MDisksList, PoolsInfoList []PoolInfo) bool {
	// Get pool names
	manager := fsRestClient.DriverManager
	poolNames := manager.GetPoolNames()
	// log.Infof("pool count: %d, pools: %v", len(poolNames), poolNames)

	// Pool metrics
	for _, pool := range PoolsInfoList {
		pool.PoolId, _ = strconv.Atoi(pool.PoolMDiskgrpInfo[MdiskIdKey].(string))
		pool.PoolName = pool.PoolMDiskgrpInfo[MdiskNameKey].(string)
		if _, bHas := poolNames[pool.PoolName]; bHas {
			poolNames[pool.PoolName] = pool.PoolId
		}

		scnames := manager.GetSCNameByPoolName(pool.PoolName)
		sort.Strings(scnames) // For testing to get unique value
		threshold := pool.PoolMDiskgrpInfo[CapacityWarningThreshold].(string)
		// The 0 means turn off the warning
		if threshold == "0" {
			threshold = "100"
		}
		pool.CapacityWarningThreshold = threshold
		pool.SystemName = manager.GetSubsystemName()
		pool.State = pool.PoolMDiskgrpInfo[PoolStatusKey].(string)
		pool.StorageClass = strings.Join(scnames, ",")

		// metadata metrics
		poolMetaMetricDesc := f.poolDescriptors[PoolMetadata]
		log.Infof("subsystem: %s, pool id: %d, name: %s, state: %s, sc: %s, warning: %s, interalStorage: %t",
			pool.SystemName,
			pool.PoolId,
			pool.PoolName,
			pool.State,
			pool.StorageClass,
			pool.CapacityWarningThreshold,
			pool.InternalStorage,
		)
		newPoolMetadataMetrics(ch, poolMetaMetricDesc, 0, &pool)
		f.newPoolWarningThreshold(ch, &pool)
		f.newPoolHealthMetrics(ch, &pool)

		log.Infof("pool id: %d, physical_free_capacity: %v, reclaimable_capacity: %v, data_reduction: %v, "+
			"physical_capacity: %v, virtual_capacity: %v, real_capacity: %v, logical_capacity: %v, logical_free_capacity: %v",
			pool.PoolId, pool.PoolMDiskgrpInfo[PhysicalFreeKey], pool.PoolMDiskgrpInfo[ReclaimableKey],
			pool.PoolMDiskgrpInfo[DataReductionKey], pool.PoolMDiskgrpInfo[PhysicalCapacityKey],
			pool.PoolMDiskgrpInfo[VirtualCapacityKey], pool.PoolMDiskgrpInfo[RealCapacityKey],
			pool.PoolMDiskgrpInfo[CapacityKey], pool.PoolMDiskgrpInfo[FreeCapacityKey])

		createPhysicalCapacityPoolMetrics(ch, f, pool, fsRestClient, mDisksList)
		createLogicalCapacityPoolMetrics(ch, f, pool)
		createTotalSavingPoolMetrics(ch, f, pool)

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
				InternalStorage:          true,
			}

			log.Infof("subsystem: %s, pool id: %d, name: %s, state: %s, sc: %s, warning: %s, internalStorage: %t",
				poolInfo.SystemName,
				poolInfo.PoolId,
				poolInfo.PoolName,
				poolInfo.State,
				poolInfo.StorageClass,
				poolInfo.CapacityWarningThreshold,
				poolInfo.InternalStorage,
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

func createLogicalCapacityPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, poolInfo PoolInfo) {
	totalLogicalCapacity, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[CapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get logical capacity failed: %s", err)
		return
	}
	logicalFreeCapacity, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[FreeCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get logical free capacity failed: %s", err)
		return
	}

	reclaimable, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[ReclaimableKey].(string), 64)
	if err != nil {
		log.Errorf("get reclaimable failed: %s", err)
		return
	}

	logicalUsableCapacity := logicalFreeCapacity + reclaimable
	logicalUsedCapacity := totalLogicalCapacity - logicalUsableCapacity

	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacityUsable], logicalUsableCapacity, &poolInfo)
	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacityUsed], logicalUsedCapacity, &poolInfo)
	newPoolCapacityMetrics(ch, f.poolDescriptors[PoolLogicalCapacity], totalLogicalCapacity, &poolInfo)
}

func createPhysicalCapacityPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, poolInfo PoolInfo,
	fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) {
	if isParentPool(poolInfo.PoolMDiskgrpInfo) {
		var reclaimableCalculatedCapacity float64
		physicalFree, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[PhysicalFreeKey].(string), 64)
		if err != nil {
			log.Errorf("get physical free failed: %s", err)
			return
		}
		physical, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[PhysicalCapacityKey].(string), 64)
		if err != nil {
			log.Errorf("get physical capacity failed: %s", err)
			return
		}
		poolOrigReclaimable, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
			return
		}
		if poolOrigReclaimable != 0 {
			reclaimableCalculatedCapacity, err = GetPoolReclaimablePhysicalCapacity(poolInfo, fsRestClient, mDisksList)
			if err != nil {
				log.Errorf("get reduced reclaimable capacity failed: %s", err)
				return
			}
		} else {
			reclaimableCalculatedCapacity = 0
		}

		used := physical - physicalFree - reclaimableCalculatedCapacity
		usable := physicalFree + poolOrigReclaimable

		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], usable, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsed], used, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolPhysicalCapacity], physical, &poolInfo)
	} else {
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], InvalidVal, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsed], InvalidVal, &poolInfo)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolPhysicalCapacity], InvalidVal, &poolInfo)
	}
}

func GetPoolReclaimablePhysicalCapacity(pool PoolInfo, fsRestClient *rest.FSRestClient, mDisksList rest.MDisksList) (float64, error) {
	var reclaimable float64
	dataReduction := pool.PoolMDiskgrpInfo[DataReductionKey].(string) == "yes"
	compressionEnabled, err := isCompressionEnabled(pool.PoolMDiskgrpInfo[MdiskNameKey].(string), mDisksList, fsRestClient)
	if err != nil {
		log.Errorf("Failed to determine internal or external pool: %s", err)
		return InvalidVal, err
	}

	if compressionEnabled && dataReduction && pool.InternalStorage && pool.ArrayMode {
		reclaimable, err = calcReducedReclaimableCapacityForPool(pool.PoolMDiskgrpInfo, fsRestClient, mDisksList)
		if err != nil {
			log.Errorf("get reduced reclaimable capacity for pool failed")
			return InvalidVal, err
		}
	} else {
		poolOrigReclaimable, err := strconv.ParseFloat(pool.PoolMDiskgrpInfo[ReclaimableKey].(string), 64)
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
			return InvalidVal, err
		}
		reclaimable = poolOrigReclaimable
	}
	return reclaimable, nil
}

func createTotalSavingPoolMetrics(ch chan<- prometheus.Metric, f *PerfCollector, poolInfo PoolInfo) {
	// TODO:ticket #42 - expose total saving per system

	drpool := poolInfo.PoolMDiskgrpInfo[DataReductionKey].(string) == "yes"

	physicalFree := float64(0)
	physical := float64(0)

	// pool_efficiency_savings
	// virtualCap = [lsmdiskgrp]:virtual_capacity
	// realCap = [lsmdiskgrp]:physical_capacity – [lsmdiskgrp]:physical_free_capacity … for DRP
	//         = [lsmdiskgrp]:real_capacity … for non-DRP
	// Total Savings = Math.max(0, virtualCap – realCap)

	var totalSaving float64
	var realCapacity float64

	virtual, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[VirtualCapacityKey].(string), 64)
	if err != nil {
		log.Errorf("get virtual capacity failed: %s", err)
	}

	realCap, err := strconv.ParseFloat(poolInfo.PoolMDiskgrpInfo[RealCapacityKey].(string), 64)
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
	var internalStorage int
	if info.InternalStorage {
		internalStorage = 1
	} else {
		internalStorage = 0
	}

	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		info.SystemName,
		fmt.Sprintf("%d", info.PoolId),
		info.PoolName,
		info.StorageClass,
		fmt.Sprintf("%d", internalStorage),
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
