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

	log "github.com/sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/rest"
	drivermanager "github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/driver"
)

const (
	// Metric name defines
	VolumeMetadata			= "volume_metadata"
	VolumeCapacityUsed		= "volume_capacity_used"

	// Feature on/off
	FeatureOn	= "on"
	FeatureOff	= "off"

	// Provision type
	TypeThin	= "thin"
	TypeThick	= "thick"
)

// Interested keys
const (
	PoolNameKey		= "mdisk_grp_name"
	PoolIdKey		= "mdisk_grp_id"
	VdiskIdKey		= "id"
	VolumeNameKey	= "volume_name"
	PrimaryKey		= "primary"
	ThinCopy		= "se_copy"
	CompressionKey	= "compressed_copy"
	EncryptionKey	= "encrypt"
	DedupKey		= "deduplicated_copy"
	CapacityKey		= "capacity"
	Uncompressed	= "uncompressed_used_capacity"
	BeforeDedup		= "used_capacity_before_reduction"
)

var (
	// Volume Metadata label
	volMetadataLabel = []string{
		"subsystem_name",
		"pool_name",
		"vol_name",
		//"pv_name",
		"compression",
		"deduplication",
		"type",
		"encryption",
	}

	// Volume capacity label
	volCapacityLabel = []string{
		"subsystem_name",
		"pool_name",
		"vol_name",
	}

	// Metric define mapping
	volumeMetricsMap = map[string]MetricLabel{
		VolumeMetadata:			{ "Volume metadata", volMetadataLabel },
		VolumeCapacityUsed:		{ "Volume capacity used (byte)", volCapacityLabel },
	}
)

type VolumeInfo struct {
	SystemName		string
	VolumeName		string
	PoolName		string
	PoolId			int
	Compression		string
	Deduplication	string
	Type			string
	Encryption		string
	CapacityUsed	float64
}

func (f *PerfCollector) initVolumeDescs() {
	f.volumeDescriptors = make(map[string]*prometheus.Desc)

	for metricName, metricLabel := range volumeMetricsMap {
		f.volumeDescriptors[metricName] = prometheus.NewDesc(
			metricName,
			metricLabel.Name, metricLabel.Labels, nil,
		)
	}
}

func (f *PerfCollector) collectVolumeMetrics(ch chan<- prometheus.Metric) bool {
	var volumes []VolumeInfo

	// Get pool names
	manager, err := drivermanager.GetManager()
	if err != nil {
		log.Errorf("get driver manager error: %s", err)
		return false
	}
	poolNames := manager.GetPoolNames()
	// Get vdisk list and filter the vdisks by pool names
	vdiskIds, err := getVdisksByPoolNames(f.client, poolNames)
	if err != nil {
		log.Errorf("get vdisks error: %s", err)
	}

	// Get pool list and filter pool names
	poolIds, err := getPoolIdsByPoolNames(f.client, poolNames)
	if err != nil {
		log.Errorf("get pool id error: %s", err)
		return false
	}

	// Get pool info for each pool id
	mdiskInfoList, err := getMdiskInfoByPoolIds(f.client, poolIds)
	if err != nil {
		log.Errorf("get pool info error: %s", err)
		return false
	}

	// Perform lsvdisk <id> for each vdisk, create VolumeInfo
	for _, vdiskId := range vdiskIds {
		volume, err := getVdiskInfoById(f.client, mdiskInfoList, vdiskId, f.systemName)
		if err != nil {
			log.Errorf("get vdisk info error: %s", err)
			continue
		}
		volumes = append(volumes, volume)
	}

	// For each volume, create metrics
	for _, vol := range volumes {
		// Get metric desc
		volumeMetaMetricDesc := f.volumeDescriptors[VolumeMetadata]
		volumeCapMetricDesc := f.volumeDescriptors[VolumeCapacityUsed]

		// Create metrics
		//log.Info(VolumeMetadata, volumeMetaMetricDesc, vol)
		newVolumeMetrics(ch, volumeMetaMetricDesc, 0, &vol)
		newVolumeCapacityMetrics(ch, volumeCapMetricDesc, vol.CapacityUsed, &vol)
	}

	return true
}

func getVdiskInfoById(client *rest.FSRestClient, mdiskInfoList []rest.PoolInfo, vdiskId string, systemName string) (VolumeInfo, error) {
	volume := VolumeInfo{}
	var err error
	var vdiskInfo rest.VdiskInfo

	// Get vdisk lists
	for i := 0; i < 2; i++ {
		// Use retry as workaround for rest commands
		vdiskInfo, err = client.LsvdiskInfo(vdiskId)
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
		return volume, err
	}

	// Subsystem name
	volume.SystemName = systemName

	baseInfo := vdiskInfo[0]
	primaryInfo := vdiskInfo[1]
	// Primary copy or not
	value, ok := primaryInfo[PrimaryKey]
	isPrimary := fmt.Sprintf("%v", value)
	if !ok || isPrimary != "yes" {
		return volume, fmt.Errorf("volume has no primary copy")
	}
	// Pool name
	value = primaryInfo[PoolNameKey]
	poolName := fmt.Sprintf("%v", value)
	volume.PoolName = poolName
	// Pool id
	value = primaryInfo[PoolIdKey]
	strId := fmt.Sprintf("%v", value)
	poolId, err := strconv.Atoi(strId)
	if err != nil {
		return volume, err
	}
	volume.PoolId = poolId
	// Volume name
	value = baseInfo[VolumeNameKey]
	volName := fmt.Sprintf("%v", value)
	volume.VolumeName = volName
	// Compression
	value = primaryInfo[CompressionKey]
	compressed := fmt.Sprintf("%v", value)
	if compressed == "yes" {
		volume.Compression = FeatureOn
	} else {
		volume.Compression = FeatureOff
	}
	// Dedup
	value = primaryInfo[DedupKey]
	dedup := fmt.Sprintf("%v", value)
	if dedup == "yes" {
		volume.Deduplication = FeatureOn
	} else {
		volume.Deduplication = FeatureOff
	}
	// Encryption
	value = primaryInfo[EncryptionKey]
	encrypt := fmt.Sprintf("%v", value)
	if encrypt == "yes" {
		volume.Encryption = FeatureOn
	} else {
		volume.Encryption = FeatureOff
	}
	// Type, thin/thick
	value = primaryInfo[ThinCopy]
	thincopy := fmt.Sprintf("%v", value)
	if thincopy == "yes" {
		volume.Type = TypeThin
	} else {
		volume.Type = TypeThick
	}
	log.Infof(
		"volume: %s, pool: %s, poolid: %d, compress: %s, dedeup: %s, type: %s, encryption: %s",
		volume.VolumeName,
		volume.PoolName,
		volume.PoolId,
		volume.Compression,
		volume.Deduplication,
		volume.Type,
		volume.Encryption,
	)

	// Capacity used
	// For thick volume, same with "capacity"
	// For thin volume, 
	// If pool is DRPool, then it's "used_capacity_before_reduction"
	// else, it's "uncompressed_used_capacity"
	var capacity string
	if volume.Type == TypeThin {
		// Get pool 'data_reduction'
		drpool, err := getDataReductionFromPoolList(mdiskInfoList, volume.PoolId)
		if err != nil {
			log.Errorf("get data reduction error: %s", err)
			return volume, err
		}
		if drpool {
			value = baseInfo[BeforeDedup]
			capacity = fmt.Sprintf("%v", value)
		} else {
			value = baseInfo[Uncompressed]
			capacity = fmt.Sprintf("%v", value)
		}
	} else {
		value = baseInfo[CapacityKey]
		capacity = fmt.Sprintf("%v", value)
	}
	// Convert capacity
	converted, err := convertCapacity(capacity)
	if err != nil {
		log.Errorf("convert capacity error: %s", err)
		return volume, err
	}
	log.Infof("vdisk: %s capacity used: %f", volume.VolumeName, converted)
	volume.CapacityUsed = converted

	return volume, nil
}

func getVdisksByPoolNames(client *rest.FSRestClient, poolNames []string) ([]string, error) {
	var err error
	var vdiskList rest.VdiskList
	vdiskIds :=  []string{}
	// Get vdisk lists
	for i := 0; i < 2; i++ {
		// Use retry as workaround for rest commands
		vdiskList, err = client.Lsvdisk()
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
		return vdiskIds, err
	}

	for _, vdisk := range vdiskList {
		//log.Infof("vdisk result: %s", vdisk)
		value, ok := vdisk[PoolNameKey]
		poolName := fmt.Sprintf("%v", value)
		if !ok {
			log.Warnf("no pool name in metric response: %s", PoolNameKey)
			continue
		}
		value, _ = vdisk[VdiskIdKey]
		vdiskid := fmt.Sprintf("%v", value)
		//log.Infof("vdisk id: %s, pool: %s", vdiskid, poolName)

		for _, pool := range poolNames {
			if poolName == pool {
				vdiskIds = append(vdiskIds, vdiskid)
				break
			}
		}
	}

	return vdiskIds, nil
}

func newVolumeMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *VolumeInfo) {
	ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			value,
			info.SystemName,
			info.PoolName,
			info.VolumeName,
			info.Compression,
			info.Deduplication,
			info.Type,
			info.Encryption,
		)
}

func newVolumeCapacityMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, info *VolumeInfo) {
	ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			value,
			info.SystemName,
			info.PoolName,
			info.VolumeName,
		)
}
