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

	"github.com/go-zookeeper/zk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

const (
	dubboRegistryPath = "/dubbo"
)

// ServiceWatcher watches for newly created dubbo services and creates a providerWatcher for each service
type ServiceWatcher struct {
	path             string
	conn             *zk.Conn
	ic               *istioclient.Clientset
	providerWatchers map[string]*ProviderWatcher
	zkName           string
}

// NewWatcher creates a ServiceWatcher
func NewServiceWatcher(conn *zk.Conn, clientset *istioclient.Clientset, zkName string) *ServiceWatcher {
	return &ServiceWatcher{
		ic:               clientset,
		path:             dubboRegistryPath,
		conn:             conn,
		providerWatchers: make(map[string]*ProviderWatcher),
		zkName:           zkName,
	}
}

// Run starts the ServiceWatcher until it receives a message over the stop chanel
// This method blocks the caller
func (w *ServiceWatcher) Run(stop <-chan struct{}) {
	w.waitFroDubboRootPath()
	eventChan := w.watchProviders(stop)
	for {
		select {
		case event := <-eventChan:
			log.Infof("received event :  %s, %v", event.Type.String(), event)
			eventChan = w.watchProviders(stop)
		case <-stop:
			return
		}
	}
}

func (w *ServiceWatcher) waitFroDubboRootPath() {
	exists := false
	for !exists {
		var err error
		exists, _, err = w.conn.Exists(w.path)
		if err != nil {
			log.Errorf("failed to check path existence : %v", err)
		}
		if !exists {
			log.Warnf("zookeeper path " + dubboRegistryPath + " doesn't exist, wait until it's created")
		}
		time.Sleep(time.Second * 2)
	}
}

func (w *ServiceWatcher) watchProviders(stop <-chan struct{}) <-chan zk.Event {
	children, newChan := watchUntilSuccess(w.path, w.conn)
	for _, node := range children {
		//skip conig node
		if node == "config" {
			continue
		}
		if _, exists := w.providerWatchers[node]; !exists {
			providerWatcher := NewProviderWatcher(w.ic, w.conn, node, w.zkName)
			w.providerWatchers[node] = providerWatcher
			log.Infof("start to watch service %s on zookeeper", node)
			go providerWatcher.Run(stop)
		}
	}
	return newChan
}
