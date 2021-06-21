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
	"os"
	"testing"

	"github.com/aeraki-framework/double2istio/test/e2e"

	"github.com/aeraki-framework/double2istio/test/e2e/util"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	util.LabelNamespace("dubbo", "istio-injection=enabled", "")
	util.KubeApply("dubbo", "../../../demo/k8s/etcd/etcd.yaml", "")
	util.KubeApply("dubbo", "../../../demo/k8s/etcd/dubbo-example.yaml", "")
}

func shutdown() {
	util.KubeDelete("dubbo", "../../../demo/k8s/etcd/etcd.yaml", "")
	util.KubeDelete("dubbo", "../../../demo/k8s/etcd/dubbo-example.yaml", "")
}

func TestCreateServiceEntry(t *testing.T) {
	e2e.TestCreateServiceEntry(t)
}

func TestDeleteServiceEntry(t *testing.T) {
	e2e.TestDeleteServiceEntry(t)
}
