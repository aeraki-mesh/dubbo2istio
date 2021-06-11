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
	"regexp"
	"strings"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	istio "istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"
)

var labelRegexp *regexp.Regexp

func init() {
	labelRegexp = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
}

// ConvertServiceEntry converts dubbo provider znode to a service entry
func ConvertServiceEntry(registryName string,
	instances []DubboServiceInstance) map[string]*v1alpha3.ServiceEntry {
	serviceEntries := make(map[string]*v1alpha3.ServiceEntry)

	for _, instance := range instances {
		dubboAttributes := instance.Metadata

		// interface+group+version+nacosGroup+nacosNS is the primary key of a dubbo service
		host := constructServiceHost(instance.Interface, dubboAttributes["version"], dubboAttributes["group"],
			instance.Group, instance.Namespace)

		serviceAccount := dubboAttributes["aeraki_meta_app_service_account"]
		if serviceAccount == "" {
			serviceAccount = defaultServiceAccount
		}

		ns := dubboAttributes["aeraki_meta_app_namespace"]
		if ns == "" {
			log.Errorf("can't find aeraki_meta_app_namespace parameter, ignore provider %v,  ", dubboAttributes)
			continue
		}
		// All the providers of a dubbo service should be deployed in the same namespace
		if se, exist := serviceEntries[host]; exist {
			if ns != se.Namespace {
				log.Errorf("found provider in multiple namespaces: %s %s, ignore provider %v", se.Namespace, ns,
					dubboAttributes)
				continue
			}
		}

		// We assume that the port of all the provider instances should be the same. Is this a reasonable assumption?
		if se, exist := serviceEntries[host]; exist {
			if instance.Port != se.Spec.Ports[0].Number {
				log.Errorf("found multiple ports for service %s : %v  %v, ignore provider %v", host,
					se.Spec.Ports[0].Number, instance.Port, dubboAttributes)
				continue
			}
		}

		// What is this for? Authorization?
		selector := dubboAttributes["aeraki_meta_workload_selector"]
		if selector == "" {
			log.Errorf("can't find aeraki_meta_workload_selector parameter for provider %v,  ", dubboAttributes)
		}

		if se, exist := serviceEntries[host]; exist {
			workloadSelector := se.Annotations["workloadSelector"]
			if selector != workloadSelector {
				log.Errorf("found provider %s with different workload selectors: %v %s", dubboAttributes,
					workloadSelector,
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
			value = strings.ReplaceAll(value, ",", "-")
			labels[key] = value
			if isInvalidLabel(key, value) {
				delete(labels, key)
				log.Infof("drop invalid label: key %s, value: %s", key, value)
			}
		}
		// to distinguish endpoints from different zk clusters
		labels[registryNameLabel] = registryName
		serviceEntry, exist := serviceEntries[host]
		if !exist {
			serviceEntry = constructServiceEntry(ns, host, instance.Interface, instance.Port, selector)
			serviceEntries[host] = serviceEntry
		}
		serviceEntry.Spec.Endpoints = append(serviceEntry.Spec.Endpoints,
			constructWorkloadEntry(instance.IP, serviceAccount, instance.Port, locality, labels))
	}
	return serviceEntries
}

func constructServiceHost(dubboInterface, version, serviceGroup, registryGroup, registryNamespace string) string {
	host := dubboInterface
	if serviceGroup != "" {
		host = host + "-" + serviceGroup
	}
	version = strings.ReplaceAll(version, ".", "-")
	if version != "" {
		host = host + "-" + version
	}
	if registryGroup != "" && registryGroup != "DEFAULT_GROUP" {
		host = host + "." + registryGroup
	}
	if registryNamespace != "" && registryNamespace != "public" {
		host = host + "." + registryNamespace
	}
	return strings.ToLower(host)
}

func constructServiceEntry(namespace string, host string, dubboInterface string, servicePort uint32,
	workloadSelector string) *v1alpha3.ServiceEntry {
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
				"registry": dubbo2istio,
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

func constructWorkloadEntry(ip string, serviceAccount string, port uint32, locality string,
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

func convertPort(port uint32) *istio.Port {
	return &istio.Port{
		Number:     port,
		Protocol:   "tcp",
		Name:       dubboPortName,
		TargetPort: port,
	}
}

// ConstructServiceEntryName constructs the service entry name for a given dubbo service
func ConstructServiceEntryName(service string) string {
	validDNSName := strings.ReplaceAll(strings.ToLower(service), ".", "-")
	return aerakiFieldManager + "-" + validDNSName
}
