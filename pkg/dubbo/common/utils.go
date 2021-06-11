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

package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// CopyMap deep copy a map
func CopyMap(m map[string]string) map[string]string {
	cp := make(map[string]string)
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// NewNacosClisent creates a Nacos client
func NewNacosClisent(addr, namespace string) (naming_client.INamingClient, error) {
	host, port, err := parseNacosURL(addr)
	if err != nil {
		return nil, err
	}

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port),
	}

	cc := *constant.NewClientConfig(
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithRotateTime("1h"),
		constant.WithMaxAge(3),
		constant.WithLogLevel("info"),
		constant.WithNamespaceId(namespace),
	)

	return clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
}

func parseNacosURL(ncAddr string) (string, uint64, error) {
	strArray := strings.Split(strings.TrimSpace(ncAddr), ":")
	if len(strArray) < 2 {
		return "", 0, fmt.Errorf("invalid nacos address: %s, "+
			"please specify a valid nacos address like \"nacos:8848\"", ncAddr)
	}
	port, err := strconv.Atoi(strArray[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid nacos address: %s, "+
			"please specify a valid nacos address like \"nacos:8848\"", ncAddr)
	}
	host := strArray[0]
	return host, uint64(port), nil
}
