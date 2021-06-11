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

package zk

import (
	"time"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/common"
	"github.com/aeraki-framework/double2istio/pkg/dubbo/zk/model"

	"github.com/go-zookeeper/zk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 1 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second
)

// ProviderWatcher watches changes on dubbo service providers and synchronize the changed dubbo providers to the Istio
// control plane via service entries
type ProviderWatcher struct {
	service        string
	path           string
	conn           *zk.Conn
	ic             *istioclient.Clientset
	serviceEntryNS map[string]string // key service entry name, value namespace
	registryName   string            // registryName is the globally unique name of a dubbo registry
}

// NewProviderWatcher creates a ProviderWatcher
func NewProviderWatcher(ic *istioclient.Clientset, conn *zk.Conn, service string, registryName string) *ProviderWatcher {
	path := "/dubbo/" + service + "/providers"
	return &ProviderWatcher{
		service:        service,
		path:           path,
		conn:           conn,
		ic:             ic,
		serviceEntryNS: make(map[string]string),
		registryName:   registryName,
	}
}

// Run starts the ProviderWatcher until it receives a message over the stop chanel
// This method blocks the caller
func (w *ProviderWatcher) Run(stop <-chan struct{}) {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time
	debouncedEvents := 0
	syncCounter := 0

	providers, eventChan := watchUntilSuccess(w.path, w.conn)
	w.syncServices2Istio(w.service, providers)

	for {
		select {
		case <-eventChan:
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				log.Debugf("This is the first debounced event")
				startDebounce = lastResourceUpdateTime
			}
			debouncedEvents++
			timeChan = time.After(debounceAfter)
			providers, eventChan = watchUntilSuccess(w.path, w.conn)
		case <-timeChan:
			log.Debugf("Receive event from time chanel")
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= debounceMax || quietTime >= debounceAfter {
				if debouncedEvents > 0 {
					syncCounter++
					log.Infof("Sync %s debounce stable[%d] %d: %v since last change, %v since last push",
						w.service, syncCounter, debouncedEvents, quietTime, eventDelay)
					w.syncServices2Istio(w.service, providers)
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

func (w *ProviderWatcher) syncServices2Istio(service string, providers []string) {
	if len(providers) == 0 {
		log.Warnf("Service %s has no providers, ignore synchronize job", service)
		return
	}
	serviceEntries, err := model.ConvertServiceEntry(w.registryName, providers)
	if err != nil {
		log.Errorf("Failed to synchronize dubbo services to Istio: %v", err)
	}

	for _, new := range serviceEntries {
		common.SyncServices2IstioUntilMaxRetries(new, w.registryName, w.ic)
	}
}

func watchUntilSuccess(path string, conn *zk.Conn) ([]string, <-chan zk.Event) {
	providers, _, eventChan, err := conn.ChildrenW(path)
	//Retry until succeed
	for err != nil {
		log.Errorf("failed to watch zookeeper path %s, %v", path, err)
		time.Sleep(1 * time.Second)
		providers, _, eventChan, err = conn.ChildrenW(path)
	}
	return providers, eventChan
}
