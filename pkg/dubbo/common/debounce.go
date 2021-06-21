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
	"time"

	"istio.io/pkg/log"
)

// Debouncer ensures the callback function is not executed too frequently. Debouncer only call the callback function
// after a given time without new bounce event, or the time after the previous execution reaches the max duration.
type Debouncer struct {
	mutex         sync.Mutex
	debounceAfter time.Duration
	debounceMax   time.Duration
	callback      func()
	eventChan     chan struct{}
	stopChan      <-chan struct{}
	canceled      bool
}

// NewDebouncer create a Debouncer.
// The callback function will be called after Bounce function stops being called for debounceAfter,
// or the total duration reaches debounceMax.
// A event sent to stop channel will stop the Debouncer.
// Callback will be called in the go routine of the Debouncer, so make sure any resources used in the Callback to be
// thread-safe.
func NewDebouncer(debounceAfter, debounceMax time.Duration, callback func(), stop <-chan struct{}) *Debouncer {
	debouncer := &Debouncer{
		debounceAfter: debounceAfter,
		debounceMax:   debounceMax,
		callback:      callback,
		stopChan:      stop,
		eventChan:     make(chan struct{}),
	}
	go debouncer.run()
	return debouncer
}

// Bounce results a new bounce event
func (d *Debouncer) Bounce() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.canceled = false
	d.eventChan <- struct{}{}
}

// Cancel the next callback if there is no bounce event after being canceled
func (d *Debouncer) Cancel() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.canceled = true
}

func (d *Debouncer) run() {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time
	debouncedEvents := 0
	fireCounter := 0

	for {
		select {
		case <-d.eventChan:
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				startDebounce = lastResourceUpdateTime
			}
			debouncedEvents++
			timeChan = time.After(d.debounceAfter)
		case <-timeChan:
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= d.debounceMax || quietTime >= d.debounceAfter {
				if debouncedEvents > 0 {
					fireCounter++
					log.Infof("debounce stable[%d] %d: %v since last event, %v since last debounce",
						fireCounter, debouncedEvents, quietTime, eventDelay)
					d.mutex.Lock()
					if d.canceled {
						log.Infof("this callback has been canceled")
					} else {
						d.callback()
					}
					d.mutex.Unlock()
					debouncedEvents = 0
				}
			} else {
				timeChan = time.After(d.debounceAfter - quietTime)
			}
		case <-d.stopChan:
			return
		}
	}
}
