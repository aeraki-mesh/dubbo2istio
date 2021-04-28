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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	watcher "github.com/aeraki-framework/double2istio/pkg/dubbo/zk/watcher"
	"github.com/go-zookeeper/zk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultZKAddr = ""
)

func main() {
	var zkAddr string
	flag.StringVar(&zkAddr, "zkaddr", defaultZKAddr, "ZooKeeper address")
	flag.Parse()

	hosts := strings.Split(zkAddr, ",")
	if len(hosts) == 0 || hosts[0] == "" {
		log.Errorf("please specify zookeeper address")
		return
	}

	conn := connectZKUntilSuccess(hosts)
	defer conn.Close()

	ic, err := getIstioClient()
	if err != nil {
		log.Errorf("failed to create istio client: %v", err)
		return
	}

	stopChan := make(chan struct{}, 1)
	serviceWatcher := watcher.NewServiceWatcher(conn, ic)
	go serviceWatcher.Run(stopChan)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	stopChan <- struct{}{}
}

func getIstioClient() (*istioclient.Clientset, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	ic, err := istioclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return ic, nil
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
	fmt.Println("--------ZK Event--------")
	fmt.Println("path:", event.Path)
	fmt.Println("type:", event.Type.String())
	fmt.Println("state:", event.State.String())
	fmt.Println("-------------------------")
}
