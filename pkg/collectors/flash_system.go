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
	clientmanagers "github.com/IBM/ibm-storage-odf-block-driver/pkg/managers"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	"github.com/prometheus/client_golang/prometheus"
	log "k8s.io/klog"
	"strconv"
)

type PerfCollector struct {
	systems   map[string]*rest.FSRestClient
	namespace string

	sysInfoDescriptors     map[string]*prometheus.Desc
	sysPerfDescriptors     map[string]*prometheus.Desc
	sysCapacityDescriptors map[string]*prometheus.Desc
	poolDescriptors        map[string]*prometheus.Desc
	volumeDescriptors      map[string]*prometheus.Desc

	// totalScrapes   prometheus.Counter
	// failedScrapes  prometheus.Counter
	// scrapeDuration prometheus.Summary

}

func NewPerfCollector(systems map[string]*rest.FSRestClient, namespace string) (*PerfCollector, error) {

	f := &PerfCollector{
		systems:   systems,
		namespace: namespace,

		// totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
		// 	Name: "exporter_total_scrapes",
		// 	Help: "Number of total scrapes",
		// }),

		// failedScrapes: prometheus.NewCounter(prometheus.CounterOpts{
		// 	Name: "exporter_failed_scrapes",
		// 	Help: "Number of failed scrapes",
		// }),

		// scrapeDuration: prometheus.NewSummary(prometheus.SummaryOpts{
		// 	Name:       "exporter_scrape_duration_seconds",
		// 	Help:       "Histogram of scrape time",
		// 	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		// }),
	}

	f.initSubsystemDescs()
	f.initPoolDescs()

	return f, nil
}

func (f *PerfCollector) Describe(ch chan<- *prometheus.Desc) {

	for _, v := range f.sysInfoDescriptors {
		ch <- v
	}

	for _, v := range f.sysPerfDescriptors {
		ch <- v
	}

	for _, v := range f.sysCapacityDescriptors {
		ch <- v
	}

	for _, v := range f.poolDescriptors {
		ch <- v
	}

	for _, v := range f.volumeDescriptors {
		ch <- v
	}

	// ch <- f.totalScrapes.Desc()
	// ch <- f.failedScrapes.Desc()
	// ch <- f.scrapeDuration.Desc()

}

func (f *PerfCollector) Collect(ch chan<- prometheus.Metric) {
	updatedSystems, err := clientmanagers.GetManagers(f.namespace, f.systems)
	if err != nil {
		return
	}
	f.systems = updatedSystems

	for systemName, fsRestClient := range f.systems {
		var poolsInfoList []PoolInfo
		pools, mDisksList, err := getSystemPoolsAndMDisks(fsRestClient)
		if err != nil {
			log.Errorf("get pools or mdisks failed: %v", err)
			return
		}
		for _, pool := range pools {
			poolInfo := PoolInfo{}
			poolInfo.PoolName = pool[MdiskNameKey].(string)
			poolInfo.PoolMDisksList, err = getPoolMDisks(fsRestClient, poolInfo.PoolName, mDisksList)
			if err != nil {
				log.Errorf("get mdisks for pool failed: %v", err)
				return
			}
			poolInfo.IsInternalStorage = IsPoolFromInternalStorage(poolInfo)
			poolInfo.IsCompressionEnabled = IsCompressionEnabled(poolInfo)
			poolInfo.IsArrayMode = IsPoolArrayMode(poolInfo)
			poolInfo.PoolId, _ = strconv.Atoi(pool[MdiskIdKey].(string))
			poolInfo.PoolMDiskGrpInfo = pool
			poolsInfoList = append(poolsInfoList, poolInfo)
		}

		log.Info("Collect metrics for ", systemName)
		f.collectSystemMetrics(ch, fsRestClient, poolsInfoList)

		valid, _ := fsRestClient.CheckVersion()
		if valid && len(fsRestClient.DriverManager.GetPoolNames()) > 0 {
			// Skip unsupported version when generate pool metrics
			f.collectPoolMetrics(ch, fsRestClient, poolsInfoList)
		}

	}
	// ch <- f.scrapeDuration
	// ch <- f.totalScrapes
	// ch <- f.failedScrapes
}

func getSystemPoolsAndMDisks(fsRestClient *rest.FSRestClient) (rest.PoolList, rest.MDisksList, error) {
	var pools rest.PoolList
	var mDisksList rest.MDisksList
	pools, err := fsRestClient.Lsmdiskgrp()
	if err != nil {
		log.Errorf("get pool list error: %v", err)
		return pools, mDisksList, err
	}

	mDisksList, err = fsRestClient.LsAllMDisk()
	if err != nil {
		log.Errorf("get disk list error: %v", err)
		return pools, mDisksList, err
	}
	return pools, mDisksList, nil
}

func getPoolMDisks(fsRestClient *rest.FSRestClient, poolName string, mDisksList rest.MDisksList) ([]rest.SingleMDiskInfo, error) {
	var mDisksInPool []rest.SingleMDiskInfo
	for _, mDisk := range mDisksList {
		if poolName == mDisk[MdiskGroupNameKey].(string) {
			mDiskId, _ := strconv.Atoi(mDisk[MdiskIdKey].(string))
			mDiskInfo, err := fsRestClient.LsSingleMDisk(mDiskId)
			if err != nil {
				log.Errorf("get single mdisk info error: %v", err)
				return mDisksInPool, err
			}
			mDisksInPool = append(mDisksInPool, mDiskInfo)
		}
	}
	return mDisksInPool, nil
}
