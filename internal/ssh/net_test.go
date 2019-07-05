// Copyright © 2019 Alvaro Saurin
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

package ssh

import (
	"testing"
)

func TestAllMatchesIPv4(t *testing.T) {
	str1 := `Proxy Port Last Check Proxy Speed Proxy Country Anonymity 118.99.81.204
    118.99.81.204 8080 34 sec Indonesia - Tangerang Transparent 2.184.31.2 8080 58 sec 
    Iran Transparent 93.126.11.189 8080 1 min Iran - Esfahan Transparent 202.118.236.130
    7777 1 min China - Harbin Transparent 62.201.207.9 8080 1 min Iraq Transparent`

	str1Addresseses := []string{
		"118.99.81.204",
		"118.99.81.204",
		"2.184.31.2",
		"93.126.11.189",
		"202.118.236.130",
		"62.201.207.9",
	}

	inList := func(lst []string, target string) bool {
		for _, v := range lst {
			if v == target {
				return true
			}
		}
		return false
	}

	r := []string{}
	_ = AllMatchesIPv4(str1, &r)
	for _, expected := range str1Addresseses {
		if !inList(r, expected) {
			t.Fatalf("%q is not in the result: %q", expected, r)
		}
	}
}
