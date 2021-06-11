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

package nacos

import (
	"time"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/common"
	"github.com/aeraki-framework/double2istio/pkg/dubbo/nacos/watcher"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 3 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second
)

// Controller contains the runtime configuration for a Nacos controller
type Controller struct {
	registryName     string // registryName is the globally unique name of a dubbo registry
	ncAddr           string
	ic               *istioclient.Clientset
	nacosClient      naming_client.INamingClient
	watchedNamespace map[string]bool
	serviceEntryNS   map[string]string // key service entry name, value namespace
	eventChan        chan []common.DubboServiceInstance
}

// NewController creates a Nacos Controller
func NewController(ncName string, ncAddr string, client *istioclient.Clientset) (*Controller, error) {
	nacosClient, error := common.NewNacosClisent(ncAddr, "")
	if error != nil {
		return nil, error
	}
	return &Controller{
		registryName:     ncName,
		ncAddr:           ncAddr,
		ic:               client,
		nacosClient:      nacosClient,
		watchedNamespace: make(map[string]bool),
		serviceEntryNS:   make(map[string]string),
		eventChan:        make(chan []common.DubboServiceInstance),
	}, nil
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go c.watchNamespace(stop)

	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time
	debouncedEvents := 0
	syncCounter := 0
	changedServices := make([]common.DubboServiceInstance, 0)

	for {
		select {
		case services := <-c.eventChan:
			changedServices = append(changedServices, services...)
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				log.Debugf("This is the first debounced event")
				startDebounce = lastResourceUpdateTime
			}
			debouncedEvents++
			timeChan = time.After(debounceAfter)
		case <-timeChan:
			log.Debugf("Receive event from time chanel")
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= debounceMax || quietTime >= debounceAfter {
				if debouncedEvents > 0 {
					syncCounter++
					log.Infof("Sync %v services, debounce stable[%d] %d: %v since last change, %v since last push",
						len(changedServices), syncCounter, debouncedEvents, quietTime, eventDelay)
					serviceEntries := common.ConvertServiceEntry(c.registryName, changedServices)
					for _, new := range serviceEntries {
						common.SyncServices2IstioUntilMaxRetries(new, c.registryName, c.ic)
					}
					debouncedEvents = 0
				}
			} else {
				timeChan = time.After(debounceAfter - quietTime)
			}
		case <-stop:
			return
		}
	}
}

func (c *Controller) watchNamespace(stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Infof("get all namespaces")
			nameSpaces, err := c.nacosClient.GetAllNamespaces()
			if err != nil {
				log.Errorf("failed to get all namespaces: %v", err)
			}
			for _, ns := range nameSpaces {
				if !c.watchedNamespace[ns.Namespace] {
					watcher, err := watcher.NewNamespaceWatcher(c.ncAddr, ns.Namespace, c.eventChan)
					if err != nil {
						log.Errorf("failed to watch namespace %s", ns.Namespace, err)
					} else {
						go watcher.Run(stop)
						c.watchedNamespace[ns.Namespace] = true
						log.Infof("start watching namespace %s", ns.Namespace)
					}
				}
			}
		case <-stop:
			return
		}
	}
}
