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
	"sync"

	istioclient "istio.io/client-go/pkg/clientset/versioned"

	"istio.io/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"
	"encoding/json"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	// the maximum retries if failed to sync dubbo services to Istio
	maxRetries = 10
)

var serviceEntryNS = make(map[string]string) //key service entry name, value namespace
var serviceEntryMutex sync.Mutex

// SyncServices2IstioUntilMaxRetries will try to synchronize service entries to the Istio until max retry number
func SyncServices2IstioUntilMaxRetries(new *v1alpha3.ServiceEntry, registryName string, ic *istioclient.Clientset) {
	serviceEntryMutex.Lock()
	defer serviceEntryMutex.Unlock()

	err := syncService2Istio(new, registryName, ic)
	retries := 0
	for err != nil {
		if isRetryableError(err) && retries < maxRetries {
			log.Errorf("Failed to synchronize dubbo services to Istio, error: %v,  retrying %v ...", err, retries)
			err = syncService2Istio(new, registryName, ic)
			retries++
		} else {
			log.Errorf("Failed to synchronize dubbo services to Istio: %v", err)
			err = nil
		}
	}
}

func syncService2Istio(new *v1alpha3.ServiceEntry, registryName string, ic *istioclient.Clientset) error {
	// delete old service entry if multiple service entries found in different namespaces.
	// Aeraki doesn't support deploying providers of the same dubbo interface in multiple namespaces because interface
	// is used as the global dns name for dubbo service across the whole mesh
	if oldNS, exist := serviceEntryNS[new.Name]; exist {
		if oldNS != new.Namespace {
			log.Errorf("found service entry %s in two namespaces : %s %s ,delete the older one %s/%s", new.Name, oldNS,
				new.Namespace, oldNS, new.Name)
			if err := ic.NetworkingV1alpha3().ServiceEntries(oldNS).Delete(context.TODO(), new.Name,
				metav1.DeleteOptions{}); err != nil {
				if isRealError(err) {
					log.Errorf("failed to delete service entry: %s/%s", oldNS, new.Name)
				}
			}
		}
	}

	existingServiceEntry, err := ic.NetworkingV1alpha3().ServiceEntries(new.Namespace).Get(context.TODO(), new.Name,
		metav1.GetOptions{},
	)

	if isRealError(err) {
		return err
	} else if isNotFound(err) {
		return createServiceEntry(new, ic)
	} else {
		mergeServiceEntryEndpoints(registryName, new, existingServiceEntry)
		return updateServiceEntry(new, existingServiceEntry, ic)
	}
}

func createServiceEntry(serviceEntry *v1alpha3.ServiceEntry, ic *istioclient.Clientset) error {
	_, err := ic.NetworkingV1alpha3().ServiceEntries(serviceEntry.Namespace).Create(context.TODO(), serviceEntry,
		metav1.CreateOptions{FieldManager: aerakiFieldManager})
	if err == nil {
		serviceEntryNS[serviceEntry.Name] = serviceEntry.Namespace
		log.Infof("service entry %s has been created: %s", serviceEntry.Name, struct2JSON(serviceEntry))
	}
	return err
}

func updateServiceEntry(new *v1alpha3.ServiceEntry,
	old *v1alpha3.ServiceEntry, ic *istioclient.Clientset) error {
	new.Spec.Ports = old.Spec.Ports
	new.ResourceVersion = old.ResourceVersion
	_, err := ic.NetworkingV1alpha3().ServiceEntries(new.Namespace).Update(context.TODO(), new,
		metav1.UpdateOptions{FieldManager: aerakiFieldManager})
	if err == nil {
		log.Infof("service entry %s has been updated: %s", new.Name, struct2JSON(new))
	}
	return err
}

func isRealError(err error) bool {
	return err != nil && !errors.IsNotFound(err)
}

func isRetryableError(err error) bool {
	return errors.IsInternalError(err) || errors.IsResourceExpired(err) || errors.IsServerTimeout(err) ||
		errors.IsServiceUnavailable(err) || errors.IsTimeout(err) || errors.IsTooManyRequests(err) ||
		errors.ReasonForError(err) == metav1.StatusReasonUnknown
}

func isNotFound(err error) bool {
	return err != nil && errors.IsNotFound(err)
}

func struct2JSON(ojb interface{}) interface{} {
	b, err := json.Marshal(ojb)
	if err != nil {
		return ojb
	}
	return string(b)
}

func mergeServiceEntryEndpoints(registryName string, new *v1alpha3.ServiceEntry, old *v1alpha3.ServiceEntry) {
	if old == nil || old.Spec.WorkloadSelector != nil {
		return
	}
	endpoints := new.Spec.Endpoints
	for _, ep := range old.Spec.Endpoints {
		// keep endpoints synchronized by other zk and delete old endpoints synchronized by zk configured in zkAddr
		if ep.Labels[registryNameLabel] != registryName {
			endpoints = append(endpoints, ep)
		}
	}
	new.Spec.Endpoints = endpoints
}
