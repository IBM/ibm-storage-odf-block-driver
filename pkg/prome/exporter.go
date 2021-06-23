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

package prome

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "k8s.io/klog"

	collector "github.com/IBM/ibm-storage-odf-block-driver/pkg/collectors"
	"github.com/IBM/ibm-storage-odf-block-driver/pkg/rest"
)

func RunExporter(restClient *rest.FSRestClient, subsystemName string, namespace string) {

	c, err := collector.NewPerfCollector(restClient, subsystemName, namespace)
	if err != nil {
		log.Warningf("NewFSPerfCollector fails, err:%s", err)
	}

	// Use custom registry to remove default go metrics
	r := prometheus.NewRegistry()
	r.MustRegister(c)
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)

	// prometheus.MustRegister(c)
	// http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var _, _ = w.Write([]byte(`<html>
            <head><title>Promethues Exporter</title></head>
            <body>
            <h1>FlashSystem Overall Perf Promethues Exporter </h1>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	log.Info("Beginning to serve on port :9100")
	log.Fatal(http.ListenAndServe(":9100", nil))
}
