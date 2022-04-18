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

package watcher

import (
	"time"

	"github.com/aeraki-mesh/dubbo2istio/pkg/dubbo/common"
	"github.com/aeraki-mesh/dubbo2istio/pkg/dubbo/nacos/model"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosmodel "github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"istio.io/pkg/log"
)

// NamespaceWatcher contains the runtime configuration for a Nacos Namespace Watcher
type NamespaceWatcher struct {
	namespace          string
	nacosClient        naming_client.INamingClient
	subscribedServices map[string]bool
	notifyChan         chan<- []common.DubboServiceInstance
}

// NewNamespaceWatcher creates a Namespace Watcher
func NewNamespaceWatcher(nacosAddr, namespace string, notifyChan chan<- []common.DubboServiceInstance) (
	*NamespaceWatcher, error) {
	nacosClient, error := common.NewNacosClisent(nacosAddr, namespace)
	if error != nil {
		return nil, error
	}
	return &NamespaceWatcher{
		namespace:          namespace,
		nacosClient:        nacosClient,
		subscribedServices: make(map[string]bool),
		notifyChan:         notifyChan,
	}, nil
}

// Run until a signal is received, this function blocks
func (w *NamespaceWatcher) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Infof("get catalog services of namespace %s ", w.namespace)
			catalogServiceList, err := w.nacosClient.GetCatalogServices(w.namespace)
			if err != nil {
				log.Errorf("failed to get catalog services of namespace %s : %v", w.namespace, err)
				continue
			}
			for _, catalogService := range catalogServiceList.ServiceList {
				if !w.subscribedServices[catalogService.Name+catalogService.GroupName] {
					err := w.nacosClient.Subscribe(&vo.SubscribeParam{
						ServiceName: catalogService.Name,
						GroupName:   catalogService.GroupName,
						Clusters:    []string{},
						SubscribeCallback: func(services []nacosmodel.SubscribeService, err error) {
							if err != nil {
								log.Errorf("failed to get notification: %v", err)
							}
							log.Infof("service %s changed: %v", catalogService.Name, len(services))
							erviceEntries, err := model.ConvertDubboServices(w.namespace,
								catalogService.GroupName,
								services)
							if err != nil {
								log.Errorf("failed to convert dubbo service: %v", err)
							}
							w.notifyChan <- erviceEntries
						},
					})
					if err != nil {
						log.Errorf("failed to subscribe to service: %s group: %s: %v", catalogService.Name,
							catalogService.GroupName,
							err)
					} else {
						log.Infof("subscribe to service: %s group: %s", catalogService.Name, catalogService.GroupName)
						w.subscribedServices[catalogService.Name+catalogService.GroupName] = true
					}
				}
			}
		case <-stop:
			return
		}
	}
}
