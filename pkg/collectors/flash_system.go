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
	"github.com/prometheus/client_golang/prometheus"
	log "k8s.io/klog"

	drivermanager "github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
	operutil "github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

type PerfCollector struct {
	systemName string
	namespace  string
	client     *rest.FSRestClient

	sysInfoDescriptors map[string]*prometheus.Desc
	sysPerfDescriptors map[string]*prometheus.Desc
	poolDescriptors    map[string]*prometheus.Desc
	volumeDescriptors  map[string]*prometheus.Desc

	up prometheus.Gauge
	// totalScrapes   prometheus.Counter
	// failedScrapes  prometheus.Counter
	// scrapeDuration prometheus.Summary

	sequenceNumber uint64
}

func NewPerfCollector(restClient *rest.FSRestClient, name string, namespace string) (*PerfCollector, error) {

	f := &PerfCollector{
		systemName: name,
		namespace:  namespace,
		client:     restClient,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "up",
			Help: "Was the last scrape successful.",
		}),

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

	for _, v := range f.poolDescriptors {
		ch <- v
	}

	for _, v := range f.volumeDescriptors {
		ch <- v
	}

	ch <- f.up.Desc()
	// ch <- f.totalScrapes.Desc()
	// ch <- f.failedScrapes.Desc()
	// ch <- f.scrapeDuration.Desc()

}

// Remove dependency for unit test
var getPoolMap = func() (operutil.ScPoolMap, error) {
	return operutil.GetPoolConfigmapContent()
}

func (f *PerfCollector) Collect(ch chan<- prometheus.Metric) {

	// Refresh pool from manager
	scPoolMap, mistake := getPoolMap()
	if mistake != nil {
		log.Fatalf("Read ConfigMap failed, error: %s", mistake)
	} else {
		log.Info("Pool read OK", scPoolMap)
	}

	mgr, err := drivermanager.GetManager()
	if err != nil {
		log.Fatalf("Get mamager failed, error: %s", err)
	}

	mgr.UpdatePoolMap(scPoolMap.ScPool)

	f.collectSystemMetrics(ch)

	valid, _ := f.client.CheckVersion()
	if valid {
		// Skip unsupported version when generate pool metrics
		f.collectPoolMetrics(ch)
	}

	ch <- f.up
	// ch <- f.scrapeDuration
	// ch <- f.totalScrapes
	// ch <- f.failedScrapes
}
