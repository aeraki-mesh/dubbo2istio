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
	"sync"
	"testing"
	"time"
)

func TestDebouncer_debounceAfter(t *testing.T) {
	lock := sync.Mutex{}
	bounceNumer := 0
	stop := make(chan struct{})
	callback := func() {
		lock.Lock()
		defer lock.Unlock()
		if bounceNumer != 10 {
			t.Errorf("test failed, expect: %v get: %v", 10, bounceNumer)
		} else {
			fmt.Printf("test succeed, expect: %v get: %v", 10, bounceNumer)
		}
	}
	d := NewDebouncer(500*time.Millisecond, 2*time.Second, callback, stop)
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		d.Bounce()
		lock.Lock()
		bounceNumer++
		lock.Unlock()
	}
	time.Sleep(600 * time.Millisecond)
	d.Bounce()
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		d.Bounce()
		lock.Lock()
		bounceNumer++
		lock.Unlock()
	}
	stop <- struct{}{}
}

func TestDebouncer_debounceMax(t *testing.T) {
	lock := sync.Mutex{}
	bounceNumer := 0
	stop := make(chan struct{})
	callback := func() {
		lock.Lock()
		defer lock.Unlock()
		if bounceNumer != 20 {
			t.Errorf("test failed, expect: %v get: %v", 20, bounceNumer)
		} else {
			fmt.Printf("test succeed, expect: %v get: %v", 20, bounceNumer)
		}
	}
	d := NewDebouncer(200*time.Millisecond, 2*time.Second, callback, stop)
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		d.Bounce()
		lock.Lock()
		bounceNumer++
		lock.Unlock()

	}
	time.Sleep(100 * time.Millisecond)
	d.Bounce()
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		d.Bounce()
		lock.Lock()
		bounceNumer++
		lock.Unlock()
	}
	stop <- struct{}{}
}
