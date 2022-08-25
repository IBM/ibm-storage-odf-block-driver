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
)

type PerfCollector struct {
	systems   map[string]*rest.FSRestClient
	namespace string

	sysInfoDescriptors     map[string]*prometheus.Desc
	sysPerfDescriptors     map[string]*prometheus.Desc
	sysCapacityDescriptors map[string]*prometheus.Desc
	poolDescriptors        map[string]*prometheus.Desc
	volumeDescriptors      map[string]*prometheus.Desc

	up prometheus.Gauge
	// totalScrapes   prometheus.Counter
	// failedScrapes  prometheus.Counter
	// scrapeDuration prometheus.Summary

	sequenceNumber uint64
}

func NewPerfCollector(systems map[string]*rest.FSRestClient, namespace string) (*PerfCollector, error) {

	f := &PerfCollector{
		systems:   systems,
		namespace: namespace,

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

	for _, v := range f.sysCapacityDescriptors {
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

func (f *PerfCollector) Collect(ch chan<- prometheus.Metric) {
	updatedSystems, err := clientmanagers.GetManagers(f.namespace, f.systems)
	if err != nil {
		panic(err)
	}
	f.systems = updatedSystems

	for systemName, fsRestClient := range f.systems {
		log.Info("Collect metrics for ", systemName)
		f.collectSystemMetrics(ch, fsRestClient)

		// TODO - collect pool metrics only if there is pools
		valid, _ := fsRestClient.CheckVersion()
		if valid {
			// Skip unsupported version when generate pool metrics
			f.collectPoolMetrics(ch, fsRestClient)
		}

	}
	ch <- f.up
	// ch <- f.scrapeDuration
	// ch <- f.totalScrapes
	// ch <- f.failedScrapes
}
