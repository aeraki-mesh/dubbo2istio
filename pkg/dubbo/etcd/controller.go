// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/common"
	"github.com/aeraki-framework/double2istio/pkg/dubbo/zk/model"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

var serviceMutex sync.Mutex

// Controller creates a etcd controller
type Controller struct {
	registryName string // registryName is the globally unique name of a dubbo registry
	addr         string
	isClient     *istioclient.Clientset
	etcdClient   *clientv3.Client
	watcher      *watcher
}

// NewController creates a etcd controller
func NewController(registryName string, addr string, client *istioclient.Clientset) *Controller {
	return &Controller{
		registryName: registryName,
		addr:         addr,
		isClient:     client,
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	hosts := strings.Split(c.addr, ",")
	c.etcdClient = c.connectEtcdUntilSuccess(hosts, stop)
}

func getServiceFromEvent(key string) string {
	if strings.Contains(key, "/providers/") {
		strArray := strings.Split(key, "/")
		if len(strArray) >= 5 && strArray[3] == "providers" {
			service := strArray[2]
			return service
		}
	}
	return ""
}
func (c *Controller) connectEtcdUntilSuccess(hosts []string, stop <-chan struct{}) *clientv3.Client {
	cli, err := c.createEtcdClient(hosts, stop)
	//Keep trying to connect to etcd until succeed
	for err != nil {
		log.Errorf("failed to connect to Etcd %s ,retrying : %v", hosts, err)
		time.Sleep(time.Second)
		cli, err = c.createEtcdClient(hosts, stop)
	}
	log.Infof("connected to etcd: %s", hosts)
	return cli
}

func (c *Controller) createEtcdClient(hosts []string, stop <-chan struct{}) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:            hosts,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    5 * time.Second,
		DialKeepAliveTimeout: 5 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock(),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{
				BaseDelay:  1.0 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   30 * time.Second,
			}}), grpc.
				WithStatsHandler(newConnHandler(c.syncAndWatch, stop))},
	})
}

func (c *Controller) syncAndWatch() {
	if c.watcher != nil {
		c.watcher.stop()
	}
	c.watcher = newWatcher(c.registryName, c.etcdClient, c.isClient)
	go c.watcher.watchService()

	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	c.syncAllService()
}

func (c *Controller) syncAllService() {
	log.Info("synchronize all dubbo services from etcd.")
	key := "/dubbo/"
	if response, err := c.etcdClient.Get(context.TODO(), key, clientv3.WithPrefix()); err != nil {
		log.Errorf("failed to get all services from Etcd")
	} else {
		services := parseGetResponse(response)
		for service, providers := range services {
			log.Infof("synchronize dubbo service: %s to Istio", service)
			syncServices2Istio(c.registryName, c.isClient, service, providers)
		}
	}
}

func syncServices2Istio(registryName string, isClient *istioclient.Clientset, service string, providers []string) {
	if len(providers) == 0 {
		log.Warnf("service %s has no providers, ignore synchronize job", service)
		return
	}
	serviceEntries, err := model.ConvertServiceEntry(registryName, providers)
	if err != nil {
		log.Errorf("failed to synchronize dubbo services to Istio: %v", err)
	}
	for _, new := range serviceEntries {
		common.SyncServices2IstioUntilMaxRetries(new, registryName, isClient)
	}
}

func parseGetResponse(response *clientv3.GetResponse) map[string][]string {
	services := make(map[string][]string)
	for _, kv := range response.Kvs {
		key := string(kv.Key)
		array := strings.Split(key, "/")
		if len(array) == 5 && array[1] == "dubbo" && array[3] == "providers" {
			service := array[2]
			providers := services[service]
			if providers == nil {
				providers = []string{}
			}
			providers = append(providers, array[4])
			services[service] = providers
		}
	}
	return services
}
