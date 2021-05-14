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
	"strconv"
	"math"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/rest"
	drivermanager "github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/driver"
)

const (
	// Metric name defines
	PoolMetadata						= "pool_metadata"
	PoolCapacityUsable					= "pool_capacity_usable"
	PoolCapacityUsed					= "pool_capacity_used"
	PoolEfficiencySavings				= "pool_efficiency_savings"
	PoolEfficiencySavingsThin			= "pool_efficiency_savings_thin"
	PoolEfficiencySavingsDedup			= "pool_efficiency_savings_dedup"
	PoolEfficiencySavingsCompression	= "pool_efficiency_savings_compression"

	// Pool state
	StateOnline		= "online"
	StateDegraded	= "degraded"
	StateOffline	= "offline"
)

// Interested keys
const (
	DataReductionKey		= "data_reduction"
	MdiskIdKey				= "id"
	//MdiskIdKey				= "mdisk_grp_id"
	//MdiskNameKey			= "mdisk_grp_name"
	PoolStatusKey			= "status"
	PhysicalFreeKey			= "physical_free_capacity"
	ReclaimableKey			= "reclaimable_capacity"
	PhysicalCapacityKey		= "physical_capacity"
	FreeCapacityKey			= "free_capacity"
	ChildPoolCapacityKey	= "child_mdisk_grp_capacity"
	RealCapacityKey			= "real_capacity"
	UsedBeforeDedupKey		= "used_capacity_before_reduction"
	UsedAfterDedupKey		= "used_capacity_after_reduction"
	DedupSavingsKey			= "deduplication_capacity_saving"
	VirtualCapacityKey		= "virtual_capacity"
	UncompressedKey			= "compression_uncompressed_capacity"
	CompressedKey			= "compression_compressed_capacity"
	CompressionActiveKey	= "compression_active"
)

var (
	// Pool Metadata label
	poolMetadataLabel = []string{
		"subsystem_name",
		"pool_id",
		"pool_name",
		"storageclass",
		"state",
	}

	// Other metrics label
	poolLabelCommon = []string{
		"subsystem_name",
		"pool_id",
	}

	// Metric define mapping
	poolMetricsMap = map[string]MetricLabel{
		PoolMetadata: 						{ "Pool metadata", poolMetadataLabel },
		PoolCapacityUsable:					{ "Pool usable capacity (Byte)", poolLabelCommon },
		PoolCapacityUsed: 					{ "Pool used capacity (byte)", poolLabelCommon },
		PoolEfficiencySavings:				{ "dedupe, thin provisioning, and compression savings", poolLabelCommon },
		PoolEfficiencySavingsThin:			{ "thin provisioning savings", poolLabelCommon },
		PoolEfficiencySavingsDedup:			{ "dedeup savings", poolLabelCommon },
		PoolEfficiencySavingsCompression:	{ "compression savings", poolLabelCommon } ,
	}

)

type PoolInfo struct {
	SystemName		string
	PoolId			int
	PoolName		string
	StorageClass	string
	State			string
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
	poolInfoList := []PoolInfo{}

	// Get pool names
	manager, err := drivermanager.GetManager()
	if err != nil {
		log.Errorf("get driver manager error: %s", err)
		return false
	}
	poolNames := manager.GetPoolNames()
	log.Infof("pool ids count: %d, pools: %s", len(poolNames), poolNames)

	// Get pool list and filter pool names
	poolIds, err := getPoolIdsByPoolNames(f.client, poolNames)
	if err != nil {
		log.Errorf("get pool id error: %s", err)
		return false
	}
	log.Infof("pool ids count: %d", len(poolIds))

	// Get pool info for each pool id
	mdiskInfoList, err := getMdiskInfoByPoolIds(f.client, poolIds)
	if err != nil {
		log.Errorf("get pool info error: %s", err)
		return false
	}
	log.Infof("pool info count: %d", len(mdiskInfoList))

	// Pool metrics
	for i, mdiskInfo := range mdiskInfoList {
		poolInfo := PoolInfo{}

		value := mdiskInfo[PoolStatusKey]
		status := fmt.Sprintf("%v", value)

		scname := manager.GetSCNameByPoolName(poolNames[i])

		poolInfo.SystemName = f.systemName
		poolInfo.PoolId = poolIds[i]
		poolInfo.PoolName = poolNames[i]
		poolInfo.State = status
		poolInfo.StorageClass = scname

		log.Infof("subsystem: %s, pool id: %d, pool name: %s, state: %s, sc: %s",
			poolInfo.SystemName,
			poolInfo.PoolId,
			poolInfo.PoolName,
			poolInfo.State,
			poolInfo.StorageClass,
		)
		// metadata metrics
		poolMetaMetricDesc := f.poolDescriptors[PoolMetadata]
		newPoolMetadataMetrics(ch, poolMetaMetricDesc, 0, &poolInfo)
		poolInfoList = append(poolInfoList, poolInfo)

		// Get pool 'data_reduction'
		drpool := getDataReductionFromPoolInfo(mdiskInfo)

		// pool_capacity_usable
		// [lsmdiskgrp]:physical_free_capacity + [lsmdiskgrp]:reclaimable_capacity
		physicalFree, err := getValue(mdiskInfo[PhysicalFreeKey])
		if err != nil {
			log.Errorf("get physical free failed: %s", err)
		}
		reclaimable, err := getValue(mdiskInfo[ReclaimableKey])
		if err != nil {
			log.Errorf("get reclaimable failed: %s", err)
		}
		usable := physicalFree + reclaimable
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolCapacityUsable], usable, &poolInfo)
		log.Infof("pool: %d capacity usable: %f", poolInfo.PoolId, usable)

		// pool_capacity_used
		// [lsmdiskgrp]:physical_capacity - pool_capacity_usable
		physical, err := getValue(mdiskInfo[PhysicalCapacityKey])
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
		virtual, err := getValue(mdiskInfo[VirtualCapacityKey])
		if err != nil {
			log.Errorf("get virtual capacity failed: %s", err)
		}
		realCap, err := getValue(mdiskInfo[RealCapacityKey])
		if err != nil {
			log.Errorf("get real capacity failed: %s", err)
		}
		phyFree, err := getValue(mdiskInfo[PhysicalFreeKey])
		if err != nil {
			log.Errorf("get physical free capacity failed: %s", err)
		}
		if drpool {
			realCapacity = physical - phyFree
		} else {
			realCapacity = realCap
		}
		totalSaving = math.Max(0, virtual - realCapacity)
		log.Infof("pool: %d, total saving: %f", poolInfo.PoolId, totalSaving)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavings], totalSaving, &poolInfo)

		// pool_efficiency_savings_thin
		// drpSavings = Math.max(0, [lsmdiskgrp]:used_capacity_before_reduction – [lsmdiskgrp]:used_capacity_after_reduction + [lsmdiskgrp]:reclaimable_capacity – [lsmdiskgrp]:deduplication_capacity_saving) 
		// drpCompressionSavings (for DRP) = drpSavings
		// For non-DRP, drpCompressionSavings = 0
		// Thin Provisioning Savings = Math.max(0, [lsmdiskgrp]:virtual_capacity – realCap – drpCompressionSavings – Math.max([lsmdiskgrp]:compression_uncompressed_capacity - [lsmdiskgrp]:compression_compressed_capacity, 0))
		var drpCompressionSavings float64
		usedBefore, err := getValue(mdiskInfo[UsedBeforeDedupKey])
		if err != nil {
			log.Errorf("get used_capacity_before_reduction failed: %s", err)
		}
		usedAfter, err := getValue(mdiskInfo[UsedAfterDedupKey])
		if err != nil {
			log.Errorf("get used_capacity_before_reduction failed: %s", err)
		}
		dedupSaving, err := getValue(mdiskInfo[DedupSavingsKey])
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

		uncompressed, err := getValue(mdiskInfo[UncompressedKey])
		if err != nil {
			log.Errorf("get compression_uncompressed_capacity failed: %s", err)
		}
		compressed, err := getValue(mdiskInfo[CompressedKey])
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
		dedupSavings, err := getValue(mdiskInfo[DedupSavingsKey])
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
		var compressSavings float64
		compression := getCompression(mdiskInfo)
		if compression {
			if drpool {
				compressSavings = drpSavings
			} else {
				compressSavings = math.Max(0, comressDiff)
			}
		} else {
			compressSavings = 0
		}
		log.Infof("pool: %d, compression saving: %f", poolInfo.PoolId, compressSavings)
		newPoolCapacityMetrics(ch, f.poolDescriptors[PoolEfficiencySavingsCompression], compressSavings, &poolInfo)
	}

	return true
}

func getMdiskInfoByPoolIds(client *rest.FSRestClient, poolIds []int) ([]rest.PoolInfo, error) {
	var err error
	poolInfoList := []rest.PoolInfo{}

	for _, poolId := range poolIds {
		poolInfo := rest.PoolInfo{}

		for i := 0; i < 2; i++ {
			poolInfo, err = client.LsmdiskInfo(poolId)

			if err != nil {
				client.Reconnect()
				log.Errorf("fails to do LsmdiskInfo, err:%s", err)
				continue
			}

			if err == nil {
				poolInfoList = append(poolInfoList, poolInfo)
				break
			}
		}

		if err != nil {
			return poolInfoList, err
		}
	}

	return poolInfoList, nil
}

func getPoolIdsByPoolNames(client *rest.FSRestClient, poolNames []string) ([]int, error) {
	var err error
	poolIds := []int{}
	poolList := rest.PoolList{}

	for i := 0; i < 2; i++ {
		poolList, err = client.Lsmdisk()

		if err != nil {
			client.Reconnect()
			log.Errorf("fails to do Lsvdisk, err:%s", err)
			continue
		}

		if err == nil {
			break
		}
	}

	if err != nil {
		return poolIds, err
	}

	for _, pool := range poolList {
		value := pool[PoolIdKey]
		strId := fmt.Sprintf("%v", value)
		poolId, err := strconv.Atoi(strId)
		if err != nil {
			return poolIds, err
		}
		value = pool[PoolNameKey]
		strName := fmt.Sprintf("%v", value)
		for _, poolName := range poolNames {
			if poolName == strName {
				poolIds = append(poolIds, poolId)
				break;
			}
		}
	}

	return poolIds, nil
}

func convertCapacity(raw string) (float64, error) {
	//expect format, such as "3.47TB"
	var numStr, unitStr []rune

	numFlag := true
	for _, r := range raw {
		if numFlag {
			if (r >= '0' && r <= '9') || r == '.' {
				numStr = append(numStr, r)
			} else {
				numFlag = false
				unitStr = append(unitStr, r)
			}
		} else {
			unitStr = append(unitStr, r)
		}
	}

	num, err := strconv.ParseFloat(string(numStr), 64)
	if err != nil {
		return 0, fmt.Errorf("fail to convert %s to float", raw)
	}

	unit := 1.0 // Byte
	switch string(unitStr) {
	case "TB":
		unit = 1024 * 1024 * 1024 * 1024.0
	case "GB":
		unit = 1024 * 1024 * 1024.0
	case "MB":
		unit = 1024 * 1024.0
	default:
		return 0, fmt.Errorf("unsupported capacity unit %s from string:%s", string(unitStr), raw)
	}

	return num * unit, nil
}

// Get "compression_active" to determine pool is compressed or not
func getCompression(poolInfo rest.PoolInfo) bool {
	var compressed bool

	value := poolInfo[CompressionActiveKey]
	compression := fmt.Sprintf("%s", value)
	if compression == "yes" {
		compressed = true
	} else {
		compressed = false
	}

	return compressed
}

// Get "data_reduction" from poolinfo
func getDataReductionFromPoolInfo(poolInfo rest.PoolInfo) bool {
	var drpool bool

	value := poolInfo[DataReductionKey]
	dataReduction := fmt.Sprintf("%s", value)
	if dataReduction == "yes" {
		drpool = true
	} else {
		drpool = false
	}

	return drpool
}

// Get "data_reduction" from poolinfo list
func getDataReductionFromPoolList(poolInfoList []rest.PoolInfo, poolId int) (bool, error) {
	var drpool bool

	for _, mdiskInfo := range poolInfoList {
		value := mdiskInfo[MdiskIdKey]
		strId := fmt.Sprintf("%s", value)
		id, err := strconv.Atoi(strId)
		if err != nil {
			return drpool, err
		}

		if id == poolId {
			value := mdiskInfo[DataReductionKey]
			dataReduction := fmt.Sprintf("%s", value)
			if dataReduction == "yes" {
				drpool = true
			} else {
				drpool = false
			}
		}
	}

	return drpool, nil
}

func newPoolCapacityMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *PoolInfo) {
	strPoolId := fmt.Sprintf("%d", info.PoolId)

	ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			value,
			info.SystemName,
			strPoolId,
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
			info.State,
		)
}

func getValue(value interface{}) (float64, error) {
	var result float64
	var err error

	result, err = convertCapacity(fmt.Sprintf("%v", value))
	if err != nil {
		log.Errorf("convert capacity error: %s, value: %s", err, value)
		return result, err
	}

	return result, nil
}

