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

package model

import (
	"github.com/aeraki-framework/dubbo2istio/pkg/dubbo/common"

	nacosmodel "github.com/nacos-group/nacos-sdk-go/model"
	"istio.io/pkg/log"
)

// ConvertDubboServices converts SubscribeServices to DubboServices
func ConvertDubboServices(registryNS, registryGroup string,
	subscribeServices []nacosmodel.SubscribeService) ([]common.DubboServiceInstance, error) {
	instances := make([]common.DubboServiceInstance, 0)

	for _, subscribeService := range subscribeServices {
		dubboInterface := subscribeService.Metadata["interface"]
		if dubboInterface == "" {
			log.Errorf("invalid dubbo service instance: interface is missing: %v", subscribeService.Metadata)
			continue
		}
		instances = append(instances, common.DubboServiceInstance{
			IP:        subscribeService.Ip,
			Port:      uint32(subscribeService.Port),
			Interface: dubboInterface,
			Namespace: registryNS,
			Group:     registryGroup,
			Metadata:  common.CopyMap(subscribeService.Metadata),
		})
	}
	return instances, nil
}
