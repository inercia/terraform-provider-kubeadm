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
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

var (
	ErrUnknownProvisioningProfile = errors.New("unknown provisioning profile")
)

func init() {
	spew.Config.Indent = "\t"
}

// runActions runs the provisioner on a specific resource and returns the new
// resource state along with an error. Instead of a diff, the ResourceConfig
// is provided since provisioners only run after a resource has been
// newly created.
func applyFn(ctx context.Context) error {
	connData := ctx.Value(schema.ProvConnDataKey).(*schema.ResourceData)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	//log.Printf("[DEBUG] [KUBEADM] kubeadm provisioner: configuration:\n%s\n", spew.Sdump(d))
	log.Printf("[DEBUG] [KUBEADM] connection:\n%s\n", spew.Sdump(connData))
	log.Printf("[DEBUG] [KUBEADM] instance state:\n%s\n", spew.Sdump(s))

	// ensure that this is a linux machine
	if s.Ephemeral.ConnInfo["type"] != "ssh" {
		return fmt.Errorf("Unsupported connection type: %s. This provisioner currently only supports linux", s.Ephemeral.ConnInfo["type"])
	}

	preventSudo := d.Get("prevent_sudo").(bool)
	useSudo := !preventSudo && s.Ephemeral.ConnInfo["user"] != "root"

	// build a communicator for the provisioner to use
	comm, err := getCommunicator(ctx, o, s)
	if err != nil {
		o.Output("Error when creating communicator")
		return err
	}

	if err := doKubeadmSetup(d, o, comm, useSudo); err != nil {
		return err
	}

	// determine what to do (init, join or join --control-plane) depending on the argument provided
	join := getJoinFromResourceData(d)
	role := getRoleFromResourceData(d)
	log.Printf("[DEBUG] [KUBEADM] will join %q, with role %q", join, role)

	var action ssh.Action
	if len(join) == 0 {
		switch role {
		case "worker":
			action = ssh.DoAbort("role is %q while no \"join\" argument has been provided", role)
		default:
			action = doKubeadmInit(d)
		}
	} else {
		switch role {
		case "master":
			action = doKubeadmJoinControlPlane(d)
		case "worker":
			action = doKubeadmJoinWorker(d)
		case "":
			action = doKubeadmJoinWorker(d)
		default:
			o.Output(fmt.Sprintf("Unknown provisioning profile: join is %q and role is %q", join, role))
			return ErrUnknownProvisioningProfile
		}
	}

	return action.Apply(o, comm, useSudo)
}
