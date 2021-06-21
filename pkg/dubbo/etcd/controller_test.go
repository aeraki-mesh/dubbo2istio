package etcd

import (
	"reflect"
	"testing"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func Test_getServiceFromEvent(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{{
		name: "test",
		key:  "/dubbo/org.apache.dubbo.samples.basic.api.ComplexService/providers/dubbo%3A%2F%2F172.20.0.69%3A20880%2Forg.apache.dubbo.samples.basic.api.ComplexService%3Faeraki_meta_app_namespace%3Ddubbo%26aeraki_meta_app_service_account%3Ddefault%26aeraki_meta_app_version%3Dv1%26aeraki_meta_locality%3Dbj%2F800002%26aeraki_meta_workload_selector%3Ddubbo-sample-provider%26anyhost%3Dtrue%26application%3Ddubbo-sample-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2%26dynamic%3Dtrue%26generic%3Dfalse%26group%3Dproduct%26interface%3Dorg.apache.dubbo.samples.basic.api.ComplexService%26methods%3DtestVoid%2CsayHello%26pid%3D7%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26service.name%3DServiceBean%3Aproduct%2Forg.apache.dubbo.samples.basic.api.ComplexService%3A2.0.0%26service_group%3Duser%26side%3Dprovider%26timestamp%3D1623819379414%26version%3D2.0.0",
		want: "org.apache.dubbo.samples.basic.api.ComplexService",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getServiceFromEvent(tt.key); got != tt.want {
				t.Errorf("getServiceFromEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseGetResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *clientv3.GetResponse
		want     map[string][]string
	}{
		{
			name: "testParseGetResponse",
			response: &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{
						Key: []byte("/dubbo/org.apache.dubbo.samples.basic.api.DemoService/providers/dubbo%3A%2F%2F172.20.0.29%3A20880%2Forg.apache.dubbo.samples.basic.api.DemoService%3Faeraki_meta_app_namespace%3Ddubbo%26aeraki_meta_app_service_account%3Ddefault%26aeraki_meta_app_version%3Dv2%26aeraki_meta_locality%3Dbj%2F800005%26aeraki_meta_workload_selector%3Ddubbo-sample-provider%26anyhost%3Dtrue%26application%3Ddubbo-sample-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2%26dynamic%3Dtrue%26generic%3Dfalse%26interface%3Dorg.apache.dubbo.samples.basic.api.DemoService%26methods%3DtestVoid%2CsayHello%26pid%3D6%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26service.name%3DServiceBean%3A%2Forg.apache.dubbo.samples.basic.api.DemoService%26service_group%3Dbatchjob%26side%3Dprovider%26timestamp%3D1624184681126"),
					},
					{
						Key: []byte("/dubbo/org.apache.dubbo.samples.basic.api.DemoService/providers/dubbo%3A%2F%2F172.20.0.28%3A20880%2Forg.apache.dubbo.samples.basic.api.DemoService%3Faeraki_meta_app_namespace%3Ddubbo%26aeraki_meta_app_service_account%3Ddefault%26aeraki_meta_app_version%3Dv2%26aeraki_meta_locality%3Dbj%2F800005%26aeraki_meta_workload_selector%3Ddubbo-sample-provider%26anyhost%3Dtrue%26application%3Ddubbo-sample-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2%26dynamic%3Dtrue%26generic%3Dfalse%26interface%3Dorg.apache.dubbo.samples.basic.api.DemoService%26methods%3DtestVoid%2CsayHello%26pid%3D6%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26service.name%3DServiceBean%3A%2Forg.apache.dubbo.samples.basic.api.DemoService%26service_group%3Dbatchjob%26side%3Dprovider%26timestamp%3D1624184681892"),
					},
				},
			},
			want: map[string][]string{"org.apache.dubbo.samples.basic.api.DemoService": {
				"dubbo%3A%2F%2F172.20.0.29%3A20880%2Forg.apache.dubbo.samples.basic.api.DemoService%3Faeraki_meta_app_namespace%3Ddubbo%26aeraki_meta_app_service_account%3Ddefault%26aeraki_meta_app_version%3Dv2%26aeraki_meta_locality%3Dbj%2F800005%26aeraki_meta_workload_selector%3Ddubbo-sample-provider%26anyhost%3Dtrue%26application%3Ddubbo-sample-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2%26dynamic%3Dtrue%26generic%3Dfalse%26interface%3Dorg.apache.dubbo.samples.basic.api.DemoService%26methods%3DtestVoid%2CsayHello%26pid%3D6%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26service.name%3DServiceBean%3A%2Forg.apache.dubbo.samples.basic.api.DemoService%26service_group%3Dbatchjob%26side%3Dprovider%26timestamp%3D1624184681126",
				"dubbo%3A%2F%2F172.20.0.28%3A20880%2Forg.apache.dubbo.samples.basic.api.DemoService%3Faeraki_meta_app_namespace%3Ddubbo%26aeraki_meta_app_service_account%3Ddefault%26aeraki_meta_app_version%3Dv2%26aeraki_meta_locality%3Dbj%2F800005%26aeraki_meta_workload_selector%3Ddubbo-sample-provider%26anyhost%3Dtrue%26application%3Ddubbo-sample-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2%26dynamic%3Dtrue%26generic%3Dfalse%26interface%3Dorg.apache.dubbo.samples.basic.api.DemoService%26methods%3DtestVoid%2CsayHello%26pid%3D6%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26service.name%3DServiceBean%3A%2Forg.apache.dubbo.samples.basic.api.DemoService%26service_group%3Dbatchjob%26side%3Dprovider%26timestamp%3D1624184681892",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGetResponse(tt.response); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGetResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
