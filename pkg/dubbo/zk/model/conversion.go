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
func ConvertServiceEntry(service string, dubboProviders []string) (map[string]*v1alpha3.ServiceEntry, error) {
	serviceEntries := make(map[string]*v1alpha3.ServiceEntry)

	for _, provider := range dubboProviders {
		dubboAttributes := parseProvider(provider)
		dubboInterface := dubboAttributes["interface"]
		instanceIP := dubboAttributes["ip"]
		dubboPort := dubboAttributes["port"]
		if dubboInterface == "" || instanceIP == "" || dubboPort == "" {
			return nil, fmt.Errorf("failed to convert dubbo provider to workloadEntry, parameters service, "+
				"interface, ip or port is missing: %s", provider)
		}

		// interface+group+version is the primary key of a dubbo service
		host := constructIGV(dubboAttributes)

		serviceAccount := dubboAttributes["aeraki_meta_app_service_account"]
		if serviceAccount == "" {
			serviceAccount = defaultServiceAccount
		}

		ns := dubboAttributes["aeraki_meta_app_namespace"]
		if ns == "" {
			log.Errorf("can't find aeraki_meta_app_namespace parameter, ignore provider %s,  ", provider)
			continue
		}
		// All the providers of a dubbo service should be deployed in the same namespace
		if se, exist := serviceEntries[host]; exist {
			if ns != se.Namespace {
				log.Errorf("found provider in multiple namespaces: %s %s, ignore provider %s", se.Namespace, ns, provider)
				continue
			}
		}

		port, err := strconv.Atoi(dubboPort)
		if err != nil {
			log.Errorf("failed to convert dubbo port to int:  %s, ignore provider %s", dubboPort, provider)
			continue
		}

		// We assume that the port of all the provider instances should be the same. Is this a reasonable assumption?
		if se, exist := serviceEntries[host]; exist {
			if uint32(port) != se.Spec.Ports[0].Number {
				log.Errorf("found multiple ports for service %s : %v  %v, ignore provider %s", service, se.Spec.Ports[0].Number, port, provider)
				continue
			}
		}

		// What is this for? Authorization?
		selector := dubboAttributes["aeraki_meta_workload_selector"]
		if selector == "" {
			log.Errorf("can't find aeraki_meta_workload_selector parameter for provider %s,  ", provider)
		}

		if se, exist := serviceEntries[host]; exist {
			workloadSelector := se.Annotations["workloadSelector"]
			if selector != workloadSelector {
				log.Errorf("found provider %s with different workload selectors: %s %s", provider, workloadSelector,
					selector)
			}
		}

		locality := strings.ReplaceAll(dubboAttributes["aeraki_meta_locality"], "%2F", "/")

		labels := dubboAttributes
		delete(labels, "service")
		delete(labels, "ip")
		delete(labels, "port")
		delete(labels, "aeraki_meta_app_service_account")
		delete(labels, "aeraki_meta_app_namespace")
		delete(labels, "aeraki_meta_workload_selector")
		delete(labels, "aeraki_meta_locality")

		version := dubboAttributes["aeraki_meta_app_version"]
		if version != "" {
			labels["version"] = version
		}

		delete(labels, "aeraki_meta_app_version")
		for key, value := range labels {
			if isInvalidLabel(key, value) {
				delete(labels, key)
				log.Infof("drop invalid label: key %s, value: %s", key, value)
			}
		}
		serviceEntry, exist := serviceEntries[host]
		if !exist {
			serviceEntry = createServiceEntry(ns, host, dubboInterface, port, selector)
			serviceEntries[host] = serviceEntry
		}
		serviceEntry.Spec.Endpoints = append(serviceEntry.Spec.Endpoints,
			createWorkloadEntry(instanceIP, serviceAccount, uint32(port), locality, labels))
	}
	return serviceEntries, nil
}

func constructIGV(attributes map[string]string) string {
	igv := attributes["interface"]
	group := attributes["group"]
	version := strings.ReplaceAll(attributes["version"], ".", "-")

	if group != "" {
		igv = igv + "-" + group
	}
	if version != "" {
		igv = igv + "-" + version
	}

	return strings.ToLower(igv)
}

func createServiceEntry(namespace string, host string, dubboInterface string, servicePort int, workloadSelector string) *v1alpha3.ServiceEntry {
	spec := &istio.ServiceEntry{
		Hosts:      []string{host},
		Ports:      []*istio.Port{convertPort(servicePort)},
		Resolution: istio.ServiceEntry_STATIC,
		Location:   istio.ServiceEntry_MESH_INTERNAL,
		Endpoints:  make([]*istio.WorkloadEntry, 0),
	}

	serviceEntry := &v1alpha3.ServiceEntry{
		ObjectMeta: v1.ObjectMeta{
			Name:      ConstructServiceEntryName(host),
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

func createWorkloadEntry(ip string, serviceAccount string, port uint32, locality string,
	labels map[string]string) *istio.WorkloadEntry {
	return &istio.WorkloadEntry{
		Address:        ip,
		Ports:          map[string]uint32{dubboPortName: port},
		ServiceAccount: serviceAccount,
		Locality:       locality,
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

// ConstructServiceEntryName constructs the service entry name for a given dubbo service
func ConstructServiceEntryName(service string) string {
	validDNSName := strings.ReplaceAll(strings.ToLower(service), ".", "-")
	return aerakiFieldManager + "-" + validDNSName
}
