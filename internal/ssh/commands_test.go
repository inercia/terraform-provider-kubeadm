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

	"github.com/hashicorp/terraform/communicator/remote"
)

func TestCheckBinaryExists(t *testing.T) {
	o := DummyOutput{}
	comm := DummyCommunicator{}
	count := 0

	// overwrite the startFunction
	comm.startFunction = func(cmd *remote.Cmd) error {
		t.Logf("it is running %q", cmd.Command)
		switch count {
		case 0:
			t.Log("returning full path to kubeadm")
			cmd.Init()
			cmd.Stdout.Write([]byte("  /usr/bin/kubeadm\r  "))
			cmd.SetExitStatus(0, nil)
		case 1:
			t.Log("returning CONDITION_SUCCEEDED")
			cmd.Init()
			cmd.Stdout.Write([]byte("CONDITION_SUCCEEDED"))
			cmd.SetExitStatus(0, nil)
		}

		count += 1
		return nil
	}

	exists, err := CheckBinaryExists("kubeadm").Check(o, comm, false)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	if !exists {
		t.Fatalf("Error: unexpected result for exists: %t", exists)
	}
}
