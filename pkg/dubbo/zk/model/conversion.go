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
	"regexp"
	"strconv"
	"strings"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/common"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/pkg/log"
)

var providerRegexp *regexp.Regexp

func init() {
	providerRegexp = regexp.MustCompile("dubbo%3A%2F%2F(.*)%3A([0-9]+)%2F(.*)%3F(.*)")
}

// ConvertServiceEntry converts dubbo provider znode to a service entry
func ConvertServiceEntry(registryName string, dubboProviders []string) (map[string]*v1alpha3.ServiceEntry, error) {
	instances := make([]common.DubboServiceInstance, 0)

	for _, provider := range dubboProviders {
		dubboAttributes := parseProvider(provider)
		dubboInterface := dubboAttributes["interface"]
		instanceIP := dubboAttributes["ip"]
		dubboPort := dubboAttributes["port"]
		if dubboInterface == "" || instanceIP == "" || dubboPort == "" {
			log.Errorf("failed to convert dubbo provider to workloadEntry, parameters service, "+
				"interface, ip or port is missing: %s", provider)
			continue
		}

		port, err := strconv.Atoi(dubboPort)
		if err != nil {
			log.Errorf("failed to convert dubbo port to int:  %s, ignore provider %s", dubboPort, provider)
			continue
		}

		instances = append(instances, common.DubboServiceInstance{
			IP:        instanceIP,
			Port:      uint32(port),
			Interface: dubboInterface,
			Namespace: "",
			Group:     "",
			Metadata:  dubboAttributes,
		})
	}
	return common.ConvertServiceEntry(registryName, instances), nil
}

func parseProvider(provider string) map[string]string {
	dubboAttributes := make(map[string]string)
	result := providerRegexp.FindStringSubmatch(provider)
	if len(result) == 5 {
		dubboAttributes["ip"] = result[1]
		dubboAttributes["port"] = result[2]
		dubboAttributes["service"] = result[3]
		parameters := strings.Split(result[4], "%26")
		for _, parameter := range parameters {
			keyValuePair := strings.Split(parameter, "%3D")
			if len(keyValuePair) == 2 {
				dubboAttributes[keyValuePair[0]] = strings.ReplaceAll(keyValuePair[1], "%2C", "-")
			}
		}
	}
	return dubboAttributes
}
