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

func TestGetNodesAndIPs(t *testing.T) {
	count := 0

	nodesIPs := []string{
		"172.20.0.16",
		"172.20.0.32",
	}
	nodesOutput := `
kubeadm-master-0 172.20.0.16
kubeadm-worker-0 172.20.0.32
`
	// overwrite the startFunction
	comm := DummyCommunicator{}
	comm.startFunction = func(cmd *remote.Cmd) error {
		t.Logf("it is running %q", cmd.Command)
		switch count {
		case 0:
			cmd.Init()
			cmd.Stdout.Write([]byte("CONDITION_SUCCEEDED"))
			cmd.SetExitStatus(0, nil)
		case 1:
			cmd.Init()
			cmd.Stdout.Write([]byte(nodesOutput))
			cmd.SetExitStatus(0, nil)
		}

		count += 1
		return nil
	}

	nodes := KubeNodesSet{}
	cfg := Config{UserOutput: DummyOutput{}, Comm: comm, UseSudo: false}
	res := DoGetKubeNodesSet("kubectl", "some-kubeconfig", &nodes).Apply(cfg)
	if IsError(res) {
		t.Fatalf("Error: %s", res.Error())
	}
	if len(nodes) != 2 {
		t.Fatalf("Error: wrong number of nodes: %d nodes found in %q", len(nodes), nodes)
	}
	for _, ip := range nodesIPs {
		if _, ok := nodes[ip]; !ok {
			t.Fatalf("Error: %q not found in nodes: %q", ip, nodes)
		}
	}
}
