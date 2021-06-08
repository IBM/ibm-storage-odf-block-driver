module github.com/IBM/ibm-storage-odf-block-driver

go 1.15

require (
	github.com/IBM/ibm-storage-odf-operator v0.0.2
	github.com/google/uuid v1.1.2 // indirect
	github.com/prometheus/client_golang v1.8.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.7.2
)
