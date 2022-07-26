module github.com/IBM/ibm-storage-odf-block-driver

go 1.18

require (
	github.com/IBM/ibm-storage-odf-operator v1.0.2-0.20220718092359-83dc0c072fbc
	github.com/prometheus/client_golang v1.8.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.3
)

require github.com/docker/go-units v0.4.0 // indirect
