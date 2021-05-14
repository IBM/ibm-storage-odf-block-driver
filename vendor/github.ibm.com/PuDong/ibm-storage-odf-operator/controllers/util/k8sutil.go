package util

import (
	"fmt"
	"os"
)

// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
// which is the namespace where the watch activity happens.
// this value is empty if the operator is running with clusterScope.
const WatchNamespaceEnvVar = "WATCH_NAMESPACE"
const ExporterImageEnvVar = "EXPORTER_IMAGE"

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

// GetExporterImage returns the exporter image from operator env by OLM bundle
func GetExporterImage() (string, error) {
	image, found := os.LookupEnv(ExporterImageEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", ExporterImageEnvVar)
	}
	return image, nil
}

// GetLabels returns the labels with cluster name
func GetLabels(clusterName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": "ibm-storage-odf-operator",
		"app.kubernetes.io/name":      clusterName,
	}
}
