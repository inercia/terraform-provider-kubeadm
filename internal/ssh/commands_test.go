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

func TestCheckBinaryExists(t *testing.T) {
	responses := []string{
		"  /usr/bin/kubeadm\r  ",
		"CONDITION_SUCCEEDED",
	}

	ctx := NewTestingContextWithResponses(responses)
	exists, err := CheckBinaryExists("kubeadm").Check(ctx)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	if !exists {
		t.Fatalf("Error: unexpected result for exists: %t", exists)
	}
}
