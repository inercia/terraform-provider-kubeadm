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
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

func TestDoLoadHelm(t *testing.T) {
	// responses from the fake remote machine
	responses := []string{
		"CONDITION_SUCCEEDED",
	}

	t.Skip("FIXME: it seems config.helm_enabled is not properly set...")
	d := schema.ResourceData{}
	_ = d.Set("install.0.kubectl_path", "")
	_ = d.Set("config.helm_enabled", "true")

	ctx, uploads := ssh.NewTestingContextForUploads(responses)
	actions := ssh.ActionList{
		doLoadHelm(&d),
	}
	res := actions.Apply(ctx)
	if ssh.IsError(res) {
		t.Fatalf("Error: %s", res.Error())
	}
	t.Logf("Uploads: %+v", *uploads)
	if len(*uploads) == 0 {
		t.Fatalf("Error: no uploads performed")
	}
	for _, manifest := range *uploads {
		if !strings.Contains(manifest, "tiller") {
			t.Fatalf("Error: no 'tiller' found in manifest:\n%s", manifest)
		}
		break
	}
}
