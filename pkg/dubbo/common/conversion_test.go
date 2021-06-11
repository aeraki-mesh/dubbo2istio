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
	"testing"
)

func Test_isValidLabel(t *testing.T) {
	tests := []struct {
		key   string
		value string
		want  bool
	}{
		{
			key:   "method",
			value: "testVoid%2CsayHello",
			want:  true,
		},
		{
			key:   "interface",
			value: "org.apache.dubbo.samples.basic.api.DemoService",
			want:  false,
		},
		{
			key:   "interface",
			value: "org.apache_dubbo-samples.basic.api.DemoService",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := isInvalidLabel(tt.key, tt.value); got != tt.want {
				t.Errorf("isValidLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}
