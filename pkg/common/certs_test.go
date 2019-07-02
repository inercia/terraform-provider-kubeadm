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

package common

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestCertsSerialization(t *testing.T) {
	etcdCrtContents := "1234567890"

	certsMap := map[string]interface{}{
		"ca_crt":    "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"ca_key":    "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"sa_crt":    "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"sa_key":    "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"etcd_crt":  etcdCrtContents,
		"etcd_key":  "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"proxy_crt": "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"proxy_key": "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
	}

	certsConfig := CertsConfig{}
	if certsConfig.HasAllCertificates() {
		t.Fatalf("Error: certsConfig seems to be filled")
	}

	err := certsConfig.FromMap(certsMap)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("certs config object:\n%s", spew.Sdump(certsConfig))

	if !certsConfig.HasAllCertificates() {
		t.Fatalf("Error: certsConfig seems to be empty")
	}

	certsMap2, err := certsConfig.ToMap()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("certs config map:\n%s", spew.Sdump(certsMap2))
	certContents, ok := certsMap2["etcd_crt"]
	if !ok {
		t.Fatalf("Error: etcd_crt not in map")
	}
	if certContents != etcdCrtContents {
		t.Fatalf("Error: etcd_crt does not match")
	}
}
