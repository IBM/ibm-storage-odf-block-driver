module github.ibm.com/PuDong/ibm-storage-odf-block-driver

go 1.14

require (
	github.com/golang/protobuf v1.4.3
	github.com/prometheus/client_golang v1.8.0
	github.com/sirupsen/logrus v1.7.0
	github.ibm.com/PuDong/ibm-storage-odf-operator v0.0.0-20210420100406-f2b7415db947
	google.golang.org/grpc v1.33.1
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.7.2
)
