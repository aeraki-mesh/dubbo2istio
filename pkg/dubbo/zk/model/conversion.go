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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	istio "istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"
)

const (
	dubboPortName = "tcp-dubbo"

	dubboRegistry = "dubbo-zookeeper"

	defaultServiceAccount = "default"

	// aerakiFieldManager is the FileldManager for Aeraki CRDs
	aerakiFieldManager = "aeraki"
)

var providerRegexp *regexp.Regexp
var labelRegexp *regexp.Regexp

func init() {
	providerRegexp = regexp.MustCompile("dubbo%3A%2F%2F(.*)%3A([0-9]+)%2F(.*)%3F(.*)")
	labelRegexp = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
}

// ConvertServiceEntry converts dubbo provider znode to a service entry
func ConvertServiceEntry(service string, dubboProviders []string) (*v1alpha3.ServiceEntry, error) {
	endpoints := make([]*istio.WorkloadEntry, 0)
	namespace := ""
	dubboInterface := ""
	workloadSelector := ""
	servicePort := 0
	for _, provider := range dubboProviders {
		dubboAttributes := parseProvider(provider)
		dubboService := strings.ToLower(dubboAttributes["service"])
		dubboInterface = dubboAttributes["interface"]
		instanceIP := dubboAttributes["ip"]
		dubboPort := dubboAttributes["port"]

		if dubboService == "" || dubboInterface == "" || instanceIP == "" || dubboPort == "" {
			return nil, fmt.Errorf("failed to convert dubbo provider to workloadEntry, parameters service, "+
				"interface, ip or port is missing: %s", provider)
		}

		serviceAccount := dubboAttributes["aeraki_meta_app_service_account"]
		if serviceAccount == "" {
			serviceAccount = defaultServiceAccount
		}

		// All the providers of a dubbo service should be deployed in the same namespace
		ns := dubboAttributes["aeraki_meta_app_namespace"]
		if ns == "" {
			log.Errorf("can't find aeraki_meta_app_namespace parameter, ignore provider %s,  ", provider)
			continue
		}
		if namespace != "" && ns != namespace {
			log.Errorf("found provider in multiple namespaces: %s %s, ignore provider %s", namespace, ns, provider)
			continue
		}
		if namespace == "" {
			namespace = ns
		}

		port, err := strconv.Atoi(dubboPort)
		if err != nil {
			log.Errorf("failed to convert dubbo port to int:  %s, ignore provider %s", dubboPort, provider)
			continue
		}

		if servicePort != 0 && port != servicePort {
			log.Errorf("found multiple ports for service %s : %v  %v, ignore provider %s", service, servicePort, port, provider)
			continue
		}

		if servicePort == 0 {
			servicePort = port
		}

		selector := dubboAttributes["aeraki_meta_workload_selector"]
		if selector == "" {
			log.Errorf("can't find aeraki_meta_workload_selector parameter for provider %s,  ", provider)
		}
		if workloadSelector != "" && selector != workloadSelector {
			log.Errorf("found provider with different workload selectors: %s %s", workloadSelector, selector,
				provider)
		}
		if workloadSelector == "" && selector != "" {
			workloadSelector = selector
		}

		labels := dubboAttributes
		delete(labels, "service")
		delete(labels, "ip")
		delete(labels, "port")
		delete(labels, "aeraki_meta_app_service_account")
		delete(labels, "aeraki_meta_app_namespace")
		delete(labels, "aeraki_meta_workload_selector")
		labels["version"] = dubboAttributes["aeraki_meta_app_version"]
		delete(labels, "aeraki_meta_app_version")
		for key, value := range labels {
			if isInvalidLabel(key, value) {
				delete(labels, key)
				log.Infof("drop invalid label: key %s, value: %s", key, value)
			}
		}

		endpoints = append(endpoints, createWorkloadEntry(instanceIP, serviceAccount, uint32(servicePort), labels))
	}
	serviceEntry := createServiceEntry(namespace, service, dubboInterface, servicePort, endpoints, workloadSelector)
	return serviceEntry, nil
}

func createServiceEntry(namespace string, service string, dubboInterface string, servicePort int,
	endpoints []*istio.WorkloadEntry, workloadSelector string) *v1alpha3.ServiceEntry {
	spec := &istio.ServiceEntry{
		Hosts:      []string{strings.ToLower(service)},
		Ports:      []*istio.Port{convertPort(servicePort)},
		Resolution: istio.ServiceEntry_STATIC,
		Location:   istio.ServiceEntry_MESH_INTERNAL,
		Endpoints:  endpoints,
	}

	serviceEntry := &v1alpha3.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      ConstructServiceEntryName(service),
			Namespace: namespace,
			Labels: map[string]string{
				"manager":  aerakiFieldManager,
				"registry": dubboRegistry,
			},
			Annotations: map[string]string{
				"interface":        dubboInterface,
				"workloadSelector": workloadSelector,
			},
		},
		Spec: *spec,
	}
	return serviceEntry
}

func createWorkloadEntry(ip string, serviceAccount string, port uint32,
	labels map[string]string) *istio.WorkloadEntry {
	return &istio.WorkloadEntry{
		Address:        ip,
		Ports:          map[string]uint32{dubboPortName: port},
		ServiceAccount: serviceAccount,
		Labels:         labels,
	}
}

func isInvalidLabel(key string, value string) bool {
	return !labelRegexp.MatchString(key) || !labelRegexp.MatchString(value) || len(key) > 63 || len(value) > 63
}

func convertPort(port int) *istio.Port {
	return &istio.Port{
		Number:     uint32(port),
		Protocol:   "tcp",
		Name:       dubboPortName,
		TargetPort: uint32(port),
	}
}

func parseProvider(provider string) map[string]string {
	dubboAttributes := make(map[string]string)
	result := providerRegexp.FindStringSubmatch(provider)
	if result != nil && len(result) == 5 {
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

func ConstructServiceEntryName(service string) string {
	validDNSName := strings.ReplaceAll(strings.ToLower(service), ".", "-")
	return aerakiFieldManager + "-" + validDNSName
}
