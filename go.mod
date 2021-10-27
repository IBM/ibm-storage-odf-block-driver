module github.com/IBM/ibm-storage-odf-block-driver

go 1.15

require (
	github.com/IBM/ibm-storage-odf-operator v1.0.0
	github.com/google/uuid v1.1.2 // indirect
	github.com/prometheus/client_golang v1.8.0
	github.com/ugorji/go v1.1.4 // indirect
	k8s.io/api v0.21.0-rc.0
	k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.3
)
