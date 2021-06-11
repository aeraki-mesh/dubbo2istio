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
	"testing"
)

func Test_parseProvider(t *testing.T) {
	dubboProvider := "dubbo%3A%2F%2F172.18.0.9%3A20880%2Forg.apache.dubbo.samples.basic.api.DemoService%3F" +
		"anyhost%3Dtrue%26application%3Ddemo-provider%26default%3Dtrue%26deprecated%3Dfalse%26dubbo%3D2.0.2" +
		"%26dynamic%3Dtrue%26generic%3Dfalse%26interface%3Dorg.apache.dubbo.samples.basic.api.DemoService%26" +
		"methods%3DtestVoid%2CsayHello%26pid%3D6%26release%3D1.0-SNAPSHOT%26revision%3D1.0-SNAPSHOT%26side%3D" +
		"provider%26timestamp%3D1618918522655"
	attributes := parseProvider(dubboProvider)
	if attributes["ip"] != "172.18.0.9" {
		t.Errorf("parseProvider ip => %v, want %v", attributes["ip"], "172.18.0.9")
	}
	if attributes["port"] != "20880" {
		t.Errorf("parseProvider port => %v, want %v", attributes["port"], "20880")
	}
	if attributes["interface"] != "org.apache.dubbo.samples.basic.api.DemoService" {
		t.Errorf("parseProvider port => %v, want %v", attributes["interface"], "org.apache.dubbo.samples.basic.api.DemoService")
	}
}
