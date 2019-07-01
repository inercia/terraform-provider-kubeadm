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

func TestParseMembersListOutput(t *testing.T) {
	s := "e942f75ad6f00855, started, kubeadm-master-0, https://172.30.0.2:2380, https://172.30.0.2:2379"

	member, err := getMemberFromString(s)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if member.Name != "kubeadm-master-0" {
		t.Fatalf("Name does not match: %s", member.Name)
	}

	if member.ID != "e942f75ad6f00855" {
		t.Fatalf("ID does not match: %s", member.ID)
	}
}

func TestParseEndpointsListOutput(t *testing.T) {
	s := "https://127.0.0.1:2379, e942f75ad6f00855, 3.3.10, 1.8 MB, true, 2, 24139"

	endpoint, err := getEndpointFromString(s)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if !endpoint.IsLeader {
		t.Fatalf("isLeader is not set")
	}

	if endpoint.ID != "e942f75ad6f00855" {
		t.Fatalf("ID does not match: %s", endpoint.ID)
	}
}
