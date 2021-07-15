module github.com/kubernetes-sigs/ibm-vpc-block-csi-driver

go 1.16

require (
	github.com/IBM/ibm-csi-common v1.0.0-beta6
	github.com/IBM/ibmcloud-volume-interface v1.0.0-beta7
	github.com/IBM/ibmcloud-volume-vpc v1.0.0-beta9
	github.com/container-storage-interface/spec v1.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.1.2
	github.com/kubernetes-csi/csi-test/v4 v4.2.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20201209123823-ac852fbbde11
	golang.org/x/sys v0.0.0-20201207223542-d4d67f95c62d
	golang.org/x/tools v0.0.0-20201023174141-c8cfbd0f21e6 // indirect
	google.golang.org/grpc v1.34.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190516230258-a675ac48af67
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190516232619-2bf8e45c8454
)
