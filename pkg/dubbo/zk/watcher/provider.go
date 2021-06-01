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
	"context"
	"encoding/json"
	"fmt"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"strings"
	"time"

	"github.com/aeraki-framework/double2istio/pkg/dubbo/zk/model"

	"github.com/go-zookeeper/zk"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// aerakiFieldManager is the FileldManager for Aeraki CRDs
	aerakiFieldManager = "aeraki"

	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 500 * time.Millisecond

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second

	// the maximum retries if failed to sync dubbo services to Istio
	maxRetries = 10
)

// ProviderWatcher watches changes on dubbo service providers and synchronize the changed dubbo providers to the Istio
// control plane via service entries
type ProviderWatcher struct {
	service        string
	path           string
	conn           *zk.Conn
	ic             *istioclient.Clientset
	serviceEntryNS map[string]string //key service entry name, value namespace
	zkName         string
}

// NewWatcher creates a ProviderWatcher
func NewProviderWatcher(ic *istioclient.Clientset, conn *zk.Conn, service string, zkName string) *ProviderWatcher {
	path := "/dubbo/" + service + "/providers"
	return &ProviderWatcher{
		service:        service,
		path:           path,
		conn:           conn,
		ic:             ic,
		serviceEntryNS: make(map[string]string),
		zkName:         zkName,
	}
}

// Run starts the ProviderWatcher until it receives a message over the stop chanel
// This method blocks the caller
func (w *ProviderWatcher) Run(stop <-chan struct{}) {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time
	debouncedEvents := 0
	syncCounter := 0

	providers, eventChan := watchUntilSuccess(w.path, w.conn)
	w.syncServices2IstioUntilMaxRetries(w.service, providers)

	for {
		select {
		case <-eventChan:
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				log.Debugf("This is the first debounced event")
				startDebounce = lastResourceUpdateTime
			}
			debouncedEvents++
			timeChan = time.After(debounceAfter)
			providers, eventChan = watchUntilSuccess(w.path, w.conn)
		case <-timeChan:
			log.Debugf("Receive event from time chanel")
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= debounceMax || quietTime >= debounceAfter {
				if debouncedEvents > 0 {
					syncCounter++
					log.Infof("Sync %s debounce stable[%d] %d: %v since last change, %v since last push",
						w.service, syncCounter, debouncedEvents, quietTime, eventDelay)
					w.syncServices2IstioUntilMaxRetries(w.service, providers)
					debouncedEvents = 0
				}
			} else {
				timeChan = time.After(debounceAfter - quietTime)
			}
		case <-stop:
			return
		}
	}
}

func (w *ProviderWatcher) syncServices2IstioUntilMaxRetries(service string, providers []string) {
	serviceEntries, err := model.ConvertServiceEntry(w.zkName, service, providers)
	if err != nil {
		log.Errorf("Failed to synchronize dubbo services to Istio: %v", err)
	}

	serviceEntryMap, err := w.getServiceEntryByDubboService(service)
	if err != nil {
		log.Errorf("Failed to list exist serviceentriyes : %v", err)
	}

	for _, new := range serviceEntries {
		w.syncService2IstioWithRetry(new, serviceEntryMap[new.Name])
		delete(serviceEntryMap, new.Name)
	}

	for _, serviceEntry := range serviceEntryMap {
		w.syncService2IstioWithRetry(nil, serviceEntry)
	}
}

func (w *ProviderWatcher) syncService2IstioWithRetry(new *v1alpha3.ServiceEntry, old *v1alpha3.ServiceEntry) {
	err := w.syncService2Istio(new, old)
	retries := 0
	for err != nil {
		if isRetryableError(err) && retries < maxRetries {
			log.Errorf("Failed to synchronize dubbo services to Istio, error: %v,  retrying %v ...", err, retries)
			err = w.syncService2Istio(new, old)
			retries++
		} else {
			log.Errorf("Failed to synchronize dubbo services to Istio: %v", err)
			err = nil
		}
	}
}

func (w *ProviderWatcher) syncService2Istio(new *v1alpha3.ServiceEntry, old *v1alpha3.ServiceEntry) error {
	if new == nil && old != nil {
		filterWorkloadEntryByZkName(w.zkName, old)
		if len(old.Spec.Endpoints) == 0 {
			return w.deleteServiceEntry(old)
		} else {
			return w.updateServiceEntry(old, old)
		}
	}
	// delete old service entry if multiple service entries found in different namespaces.
	// Aeraki doesn't support deploying providers of the same dubbo interface in multiple namespaces because interface
	// is used as the global dns name for dubbo service across the whole mesh
	if oldNS, exist := w.serviceEntryNS[new.Name]; exist {
		if oldNS != new.Namespace {
			log.Errorf("found service entry %s in two namespaces : %s %s ,delete the older one %s/%s", new.Name, oldNS,
				new.Namespace, oldNS, new.Name)
			if err := w.ic.NetworkingV1alpha3().ServiceEntries(oldNS).Delete(context.TODO(), new.Name,
				metav1.DeleteOptions{}); err != nil {
				if isRealError(err) {
					log.Errorf("failed to delete service entry: %s/%s", oldNS, new.Name)
				}
			}
		}
	}

	if new != nil && old == nil {
		return w.createServiceEntry(new)
	}
	if new != nil && old != nil {
		mergeServiceEntryEndpoints(w.zkName, new, old)
		return w.updateServiceEntry(new, old)
	}
	return nil
}

func (w *ProviderWatcher) createServiceEntry(serviceEntry *v1alpha3.ServiceEntry) error {
	_, err := w.ic.NetworkingV1alpha3().ServiceEntries(serviceEntry.Namespace).Create(context.TODO(), serviceEntry,
		metav1.CreateOptions{FieldManager: aerakiFieldManager})
	if err == nil {
		w.serviceEntryNS[serviceEntry.Name] = serviceEntry.Namespace
		log.Infof("service entry %s has been created: %s", serviceEntry.Name, struct2JSON(serviceEntry))
	}
	return err
}

func (w *ProviderWatcher) updateServiceEntry(new *v1alpha3.ServiceEntry,
	old *v1alpha3.ServiceEntry) error {
	new.Spec.Ports = old.Spec.Ports
	new.ResourceVersion = old.ResourceVersion
	_, err := w.ic.NetworkingV1alpha3().ServiceEntries(new.Namespace).Update(context.TODO(), new,
		metav1.UpdateOptions{FieldManager: aerakiFieldManager})
	if err == nil {
		log.Infof("service entry %s has been updated: %s", new.Name, struct2JSON(new))
	}
	return err
}

func (w *ProviderWatcher) deleteServiceEntry(serviceEntry *v1alpha3.ServiceEntry) error {
	err := w.ic.NetworkingV1alpha3().ServiceEntries(serviceEntry.Namespace).Delete(context.TODO(),
		serviceEntry.Name, metav1.DeleteOptions{})
	if err == nil {
		delete(w.serviceEntryNS, serviceEntry.Name)
		log.Infof("service entry %s/%s has been deleted", serviceEntry.Namespace, serviceEntry.Name)
	} else if isNotFound(err) {
		log.Infof("service entry %s/%s doesn't exist", serviceEntry.Namespace, serviceEntry.Name)
		return nil
	}
	return err
}

func (w *ProviderWatcher) getServiceEntryByDubboService(service string) (map[string]*v1alpha3.ServiceEntry, error) {
	//labelSelector := &metav1.LabelSelector{
	//	MatchLabels:map[string]string{
	//		"dubboservice": service,
	//	},
	//}
	//labelMap, _ := metav1.LabelSelectorAsMap(labelSelector)
	options := metav1.ListOptions{
		//FieldSelector: labels.SelectorFromSet(labelMap).String(),
	}
	serviceEntryList, err := w.ic.NetworkingV1alpha3().ServiceEntries("").List(context.TODO(), options)
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("failed to list service entry: %v", err)
	}

	name := model.ConstructServiceEntryName(service)
	serviceEntryMap := make(map[string]*v1alpha3.ServiceEntry)
	for _, serviceEntry := range serviceEntryList.Items {
		if strings.HasPrefix(serviceEntry.Name, name) {
			serviceEntryMap[serviceEntry.Name] = serviceEntry.DeepCopy()
		}
	}
	return serviceEntryMap, nil
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

func watchUntilSuccess(path string, conn *zk.Conn) ([]string, <-chan zk.Event) {
	providers, _, eventChan, err := conn.ChildrenW(path)
	//Retry until succeed
	for err != nil {
		log.Errorf("failed to watch zookeeper path %s, %v", path, err)
		time.Sleep(1 * time.Second)
		providers, _, eventChan, err = conn.ChildrenW(path)
	}
	return providers, eventChan
}

func struct2JSON(ojb interface{}) interface{} {
	b, err := json.Marshal(ojb)
	if err != nil {
		return ojb
	}
	return string(b)
}

func mergeServiceEntryEndpoints(zkName string, new *v1alpha3.ServiceEntry, old *v1alpha3.ServiceEntry) error {
	if old == nil || old.Spec.WorkloadSelector != nil {
		return nil
	}
	endpoints := new.Spec.Endpoints
	for _, ep := range old.Spec.Endpoints {
		// keep endpoints synchronized by other zk and delete old endpoints synchronized by zk configured in zkAddr
		if ep.Labels["zkName"] != zkName {
			endpoints = append(endpoints, ep)
		}
	}
	new.Spec.Endpoints = endpoints
	return nil
}

func filterWorkloadEntryByZkName(zkName string, serviceEntry *v1alpha3.ServiceEntry) (*v1alpha3.ServiceEntry, error) {
	var workloadEntries []*networkingv1alpha3.WorkloadEntry
	for _, workloadEntry := range serviceEntry.Spec.Endpoints {
		if workloadEntry.Labels["zkName"] != zkName {
			workloadEntries = append(workloadEntries, workloadEntry)
		}
	}
	serviceEntry.Spec.Endpoints = workloadEntries
	return serviceEntry, nil
}
