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

package provisioner

import "testing"

func TestParseEndpointsListOutput(t *testing.T) {
	s := "https://127.0.0.1:2379, e942f75ad6f00855, 3.3.10, 1.8 MB, true, 2, 24139"

	endpoint := EtcdEndpoint{}
	if err := endpoint.FromString(s); err != nil {
		t.Fatalf("Error: %v", err)
	}

	if !endpoint.IsLeader {
		t.Fatalf("isLeader is not set")
	}

	if endpoint.ID != "e942f75ad6f00855" {
		t.Fatalf("ID does not match: %s", endpoint.ID)
	}

	s = `
https://127.0.0.1:2379, e942f75ad6f00855, 3.3.10, 1.8 MB, true, 2, 24139\r
https://127.0.5.1:2379, 2f75f75431008954, 3.3.10, 1.8 MB, true, 3, 24139\r
https://127.0.8.1:2379, f0085f42f7f00855, 3.3.10, 1.8 MB, false, 2, 24139\r
`

	endpoints := EtcdEndpointsSet{}
	if err := endpoints.FromString(s); err != nil {
		t.Fatalf("Error: %v", err)
	}

	expected := map[string]struct{}{
		"e942f75ad6f00855": {},
		"2f75f75431008954": {},
		"f0085f42f7f00855": {},
	}
	for _, e := range endpoints {
		if _, ok := expected[e.ID]; !ok {
			t.Fatalf("%s not found in expected map", e.ID)
		}
	}

}
