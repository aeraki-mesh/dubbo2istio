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
	"strings"
	"time"

	watcher "github.com/aeraki-mesh/double2istio/pkg/dubbo/zk/watcher"
	"github.com/go-zookeeper/zk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

// Controller creates a ZK controller
type Controller struct {
	registryName string // registryName is the globally unique name of a dubbo registry
	zkAddr       string
	isClient     *istioclient.Clientset
}

// NewController creates a ZK controller
func NewController(registryName string, zkAddr string, client *istioclient.Clientset) *Controller {
	return &Controller{
		registryName: registryName,
		zkAddr:       zkAddr,
		isClient:     client,
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	hosts := strings.Split(c.zkAddr, ",")
	conn := connectZKUntilSuccess(hosts)
	serviceWatcher := watcher.NewServiceWatcher(conn, c.isClient, c.registryName)
	go serviceWatcher.Run(stop)
}

func connectZKUntilSuccess(hosts []string) *zk.Conn {
	option := zk.WithEventCallback(callback)
	conn, _, err := zk.Connect(hosts, time.Second*10, option)
	//Keep trying to connect to ZK until succeed
	for err != nil {
		log.Errorf("failed to connect to ZooKeeper %s ,retrying : %v", hosts, err)
		time.Sleep(time.Second)
		conn, _, err = zk.Connect(hosts, time.Second*10, option)
	}
	return conn
}

func callback(event zk.Event) {
	log.Debug("--------ZK Event--------")
	log.Debug("path:", event.Path)
	log.Debug("type:", event.Type.String())
	log.Debug("state:", event.State.String())
	log.Debug("-------------------------")
}
