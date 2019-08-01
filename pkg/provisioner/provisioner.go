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

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
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

	//ssh.Debug("kubeadm provisioner: configuration:\n%s\n", spew.Sdump(d))
	ssh.Debug("connection:\n%s\n", spew.Sdump(connData))
	ssh.Debug("instance state:\n%s\n", spew.Sdump(s))

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

	// add some extra things to the context
	newCtx := ssh.WithValues(ctx, o, o, comm, useSudo)

	//
	// resource destruction
	//

	drain := d.Get("drain").(bool)
	if drain {
		ssh.Debug("node will be drained")
		action := doRemoveNode(d)
		return action.Apply(newCtx)
	}

	//
	// resource creation
	//

	actions := ssh.ActionList{}

	if s.Tainted {
		actions = append(actions, ssh.DoMessageInfo("This node will be recreated"))
		// TODO: maybe we should exit here
	}
	if d.IsNewResource() {
		actions = append(actions, ssh.DoMessageInfo("New resource: provisioning"))
	}

	// add the actions for installing kubeadm
	actions = append(actions, doKubeadmSetup(d))

	// determine what to do (init, join or join --control-plane) depending on the argument provided
	join := getJoinFromResourceData(d)
	role := getRoleFromResourceData(d)

	// some common actions to do BEFORE doing initting/joining
	actions = append(actions,
		ssh.DoMessageInfo("Checking we have the required binaries..."),
		doCheckCommonBinaries(d),
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadBytesToFile([]byte(assets.KubeletSysconfigCode), getSysconfigPathFromResourceData(d)),
		ssh.DoUploadBytesToFile([]byte(assets.KubeletServiceCode), getServicePathFromResourceData(d)),
		ssh.DoUploadBytesToFile([]byte(assets.KubeadmDropinCode), getDropinPathFromResourceData(d)),
	)

	if len(join) == 0 {
		switch role {
		case "worker":
			actions = append(actions, ssh.ActionError(fmt.Sprintf("role is %q while no \"join\" argument has been provided", role)))
		default:
			actions = append(actions, doKubeadmInit(d))
		}
	} else {
		switch role {
		case "master":
			actions = append(actions, doKubeadmJoinControlPlane(d))
		case "worker":
			actions = append(actions, doKubeadmJoinWorker(d))
		case "":
			actions = append(actions, doKubeadmJoinWorker(d))
		default:
			actions = append(actions, ssh.ActionError(fmt.Sprintf("unknown provisioning profile: join is %q and role is %q", join, role)))
		}
	}

	// ... and some common actions to do AFTER initting/joining
	actions = append(actions,
		ssh.DoMessageInfo("Gathering some info about this node..."),
		doCheckLocalKubeconfigIsAlive(d),
		doPrintEtcdStatus(d),
	)

	return ssh.ActionList{
		ssh.DoWithCleanup(
			actions,
			ssh.DoCleanupLeftovers()),
	}.Apply(newCtx)
}
