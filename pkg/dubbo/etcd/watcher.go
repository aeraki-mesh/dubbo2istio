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

package etcd

import (
	"context"
	"time"

	"github.com/zhaohuabing/debounce"

	clientv3 "go.etcd.io/etcd/client/v3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/pkg/log"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 5 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 20 * time.Second
)

type watcher struct {
	registryName string // registryName is the globally unique name of a dubbo registry
	isClient     *istioclient.Clientset
	etcdClient   *clientv3.Client
	stopChan     chan struct{}
}

func newWatcher(registryName string, etcdClient *clientv3.Client,
	client *istioclient.Clientset) *watcher {
	return &watcher{
		registryName: registryName,
		etcdClient:   etcdClient,
		isClient:     client,
		stopChan:     make(chan struct{}),
	}
}

func (w *watcher) stop() {
	w.stopChan <- struct{}{}
}

func (w *watcher) watchService() {
	log.Info("start to watch etcd")
	ctx, cancel := context.WithCancel(context.Background())
	watchChannel := w.etcdClient.Watch(clientv3.WithRequireLeader(ctx), "/dubbo", clientv3.WithPrefix())
	changedServices := make(map[string]string)

	callback := func() {
		serviceMutex.Lock()
		defer serviceMutex.Unlock()
		log.Infof("changedEvent :%v", changedServices)
		for service := range changedServices {
			w.processChangedService(service)
		}
	}
	debouncer := debounce.New(debounceAfter, debounceMax, callback, w.stopChan)
	for {
		select {
		case response, more := <-watchChannel:
			if err := response.Err(); err != nil || !more {
				log.Errorf("failed to watch etcd: %v", err)
				ctx, cancel = context.WithCancel(context.Background())
				watchChannel = w.etcdClient.Watch(clientv3.WithRequireLeader(ctx), "/dubbo", clientv3.WithPrefix())
			}
			serviceMutex.Lock()
			for _, event := range response.Events {
				key := string(event.Kv.Key)
				if service := getServiceFromEvent(key); service != "" {
					changedServices[service] = service
					log.Infof("%s service %s", event.Type.String(), service)
				}
			}
			serviceMutex.Unlock()
			debouncer.Bounce()
		case <-w.stopChan:
			cancel()
			log.Infof("this etcd watcher has been closed.")
			return
		}
	}
}

func (w *watcher) processChangedService(service string) {
	key := "/dubbo/" + service + "/providers/"
	if response, err := w.etcdClient.Get(context.TODO(), key, clientv3.WithPrefix()); err != nil {
		log.Errorf("failed to get service providers: %s from Etcd", service)
	} else {
		providers := make([]string, 0)
		for _, kv := range response.Kvs {
			providers = append(providers, string(kv.Key))

		}
		log.Infof("synchronize dubbo service: %s to Istio", service)
		syncServices2Istio(w.registryName, w.isClient, service, providers)
	}
}
