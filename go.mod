module github.com/aeraki-mesh/double2istio

go 1.14

replace github.com/nacos-group/nacos-sdk-go => github.com/aeraki-mesh/nacos-sdk-go v0.0.0-20210611053344-ca0530b4c111

require (
	github.com/go-zookeeper/zk v1.0.2
	github.com/google/go-github v17.0.0+incompatible
	github.com/hashicorp/go-multierror v1.1.1
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/pkg/errors v0.9.1
	github.com/zhaohuabing/debounce v1.0.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/grpc v1.38.0
	istio.io/api v0.0.0-20210601145914-9a4239731e79
	istio.io/client-go v0.0.0-20210601151459-89ee09f12704
	istio.io/istio v0.0.0-20210603041206-aa439f6e4772
	istio.io/pkg v0.0.0-20210528151021-2059ed14a0e6
	k8s.io/apimachinery v0.21.1
	sigs.k8s.io/controller-runtime v0.9.0-beta.5
)
