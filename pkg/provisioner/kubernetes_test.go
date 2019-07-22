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

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

func TestDoGetNodename(t *testing.T) {
	machineID := "  bf38f8ac633e4f64a4924b0ed7b25946\r"
	output := `
bf38f8ac633e4f64a4924b0ed7b25946        kubeadm-master-0
0b44fe52491e401181c4ef5607b70e96        kubeadm-worker-0
`

	// responses from the fake remote machine
	responses := []string{
		machineID,
		"CONDITION_SUCCEEDED",
		output,
	}

	d := schema.ResourceData{}
	_ = d.Set("install.0.kubectl_path", "")

	node := ssh.KubeNode{}
	ctx := ssh.NewTestingContextWithResponses(responses)
	actions := ssh.ActionList{
		DoGetNodename(&d, &node),
	}
	res := actions.Apply(ctx)
	if ssh.IsError(res) {
		t.Fatalf("Error: %s", res.Error())
	}
	if node.Nodename != "kubeadm-master-0" {
		t.Fatalf("Error: wrong nodename %q", node.Nodename)
	}
}
