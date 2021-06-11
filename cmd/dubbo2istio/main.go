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
	"os"
	"os/signal"
	"syscall"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/nacos"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/zk"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultRegistryType = "zookeeper"
	defaultRegistryName = "default"
)

func main() {
	var registryType, registryName, registryAddr string
	flag.StringVar(&registryType, "type", defaultRegistryType, "Type of the registry: zookeeper or nacos")
	flag.StringVar(&registryName, "name", defaultRegistryName, "Name is the globally unique name for a dubbo registry")
	flag.StringVar(&registryAddr, "addr", "", "Registry address in the form of ip:port or dns:port")
	flag.Parse()

	stopChan := make(chan struct{}, 1)
	ic, err := getIstioClient()
	if err != nil {
		log.Errorf("failed to create istio client: %v", err)
		return
	}

	if registryType == "zookeeper" {
		log.Infof("dubbo2istio runs in Zookeeper mode: registry: %s, addr: %s", registryName, registryAddr)
		zk.NewController(registryName, registryAddr, ic).Run(stopChan)
	} else if registryType == "nacos" {
		log.Infof("dubbo2istio runs in Nacos mode: registry: %s, addr: %s", registryName, registryAddr)
		nacosController, err := nacos.NewController(registryName, registryAddr, ic)
		if err != nil {
			log.Errorf("failed to create nacos controller :%v", err)
		} else {
			nacosController.Run(stopChan)
		}
	} else {
		log.Errorf("unrecognized registry type: %s , dubbo2istio only supports zookeeper or nacos", registryType)
	}

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
