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

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/aeraki-framework/double2istio/test/e2e/util"
)

// TestCreateServiceEntry tests creating service entry
func TestCreateServiceEntry(t *testing.T) {
	util.Shell("kubectl get pod -n dubbo -oyaml")
	err := util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	if err != nil {
		t.Errorf("failed to wait for deployment: %v", err)
	}
	util.Shell("kubectl get deploy dubbo2istio -n dubbo -oyaml")
	util.Shell("kubectl get pod -n dubbo -oyaml")
	//wait 60 seconds for service entries to be created
	time.Sleep(60 * time.Second)

	serviceEntries, err := util.KubeGetYaml("dubbo", "serviceentry", "", "")
	if err != nil {
		t.Errorf("failed to get serviceentry %v", err)
	}

	expectedServices := []string{
		"org.apache.dubbo.samples.basic.api.complexservice-product-2-0-0",
		"org.apache.dubbo.samples.basic.api.complexservice-test-1-0-0",
		"org.apache.dubbo.samples.basic.api.demoservice",
		"org.apache.dubbo.samples.basic.api.testservice"}
	for _, service := range expectedServices {
		if !strings.Contains(serviceEntries, service) {
			t.Errorf("can't find expected serviceentry: %s", service)
		}
	}

	serviceEntry, err := util.Shell("kubectl get serviceentry aeraki-org-apache-dubbo-samples-basic-api-demoservice" +
		" -n dubbo -o=jsonpath='{range .items[*]}{.spec.endpoints}'")
	if err != nil {
		t.Errorf("failed to get serviceentry %v", err)
	}
	count := strings.Count(serviceEntry, "address")
	if count != 2 {
		t.Errorf("endpoint number is not correct, expect: %v, get %v", 2, count)
	}
}

// TestDeleteServiceEntry tests deleting service entry
func TestDeleteServiceEntry(t *testing.T) {
	_, err := util.Shell("kubectl delete deploy dubbo-sample-provider-v1 -n dubbo")
	if err != nil {
		t.Errorf("failed to delete deploy %v", err)
	}

	//wait 60 seconds for the endpoint to be deleted
	time.Sleep(120 * time.Second)

	serviceEntry, err := util.Shell("kubectl get serviceentry aeraki-org-apache-dubbo-samples-basic-api-demoservice -n dubbo -o=jsonpath='{range .items[*]}{.spec.endpoints}'")
	if err != nil {
		t.Errorf("failed to get serviceentry %v", err)
	}
	count := strings.Count(serviceEntry, "address")
	if count != 1 {
		t.Errorf("endpoint number is not correct, expect: %v, get %v", 1, count)
	}
}
