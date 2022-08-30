module github.com/IBM/ibm-storage-odf-block-driver

go 1.15

require (
	github.com/IBM/ibm-storage-odf-operator v1.3.0-rc1.0.20220830102604-52e74686da0b
	// TODO change to updated 1.3.0 operator version
	github.com/prometheus/client_golang v1.8.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.3
)
