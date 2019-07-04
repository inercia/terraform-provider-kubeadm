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
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
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
	join := strings.TrimSpace(d.Get("join").(string))
	role := strings.TrimSpace(strings.ToLower(d.Get("role").(string)))
	log.Printf("[DEBUG] [KUBEADM] will join %q, with role %q", join, role)

	var action ssh.Applyer
	if len(join) == 0 {
		switch role {
		case "worker":
			action = ssh.DoAbort(fmt.Sprintf("role is %q while no \"join\" argument has been provided", role))
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

// doKubeadmInit runs the `kubeadm init`
func doKubeadmInit(d *schema.ResourceData) ssh.Applyer {
	_, initConfigBytes, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for init'ing: %s", err))
	}
	extraArgs := []string{"--skip-token-print"}

	actions := []ssh.Applyer{
		ssh.DoMessageInfo("Initializing the cluster with 'kubadm init'"),
		doDeleteLocalKubeconfig(d),
		doUploadCerts(d),
		ssh.DoIfElse(
			ssh.CheckFileExists(ssh.DefAdminKubeconfig),
			ssh.DoMessage("admin.conf already exists: skipping `kubeadm init`"),
			doKubeadm(d, "init", initConfigBytes, extraArgs...),
		),
		doDownloadKubeconfig(d),
		doPrintEtcdMembers(d),
		doLoadCNI(d),
		doLoadDashboard(d),
		doLoadHelm(d),
		doLoadManifests(d),
	}

	return ssh.DoComposed(actions...)
}

// doKubeadmJoinWorker runs the `kubeadm join`
func doKubeadmJoinWorker(d *schema.ResourceData) ssh.Applyer {
	_, joinConfigBytes, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// check if we are joining the Control Plane: we must upload the certificates and
	// use the '--control-plane' flag
	actions := []ssh.Applyer{
		ssh.DoMessageInfo("Joining the cluster with 'kubadm join'"),
		doKubeadm(d, "join", joinConfigBytes),
	}
	return ssh.DoComposed(actions...)
}

// doKubeadmJoinWorker runs the `kubeadm join`
func doKubeadmJoinControlPlane(d *schema.ResourceData) ssh.Applyer {
	joinConfig, _, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// check that we have a stable control plane endpoint
	initConfig, _, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}
	if len(initConfig.ClusterConfiguration.ControlPlaneEndpoint) == 0 {
		return ssh.ApplyError("Cannot create additional masters when the 'kubeadm.<name>.api.external' is empty")
	}

	// add a local Control-Plane section to the JoinConfiguration
	endpoint := kubeadmapi.APIEndpoint{}
	if hp, ok := d.GetOk("listen"); ok {
		h, p, err := common.SplitHostPort(hp.(string), common.DefAPIServerPort)
		if err != nil {
			return ssh.ApplyError(fmt.Sprintf("could not parse listen address %q: %s", hp.(string), err))
		}
		endpoint = kubeadmapi.APIEndpoint{AdvertiseAddress: h, BindPort: int32(p)}
	} else {
		endpoint = kubeadmapi.APIEndpoint{AdvertiseAddress: "", BindPort: common.DefAPIServerPort}
	}
	joinConfig.ControlPlane = &kubeadmapi.JoinControlPlane{LocalAPIEndpoint: endpoint}

	joinConfigBytes, err := common.JoinConfigToYAML(joinConfig)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	extraArgs := []string{}
	actions := []ssh.Applyer{
		ssh.DoMessageInfo("Joining the cluster with 'kubadm join'"),
		doUploadCerts(d),
		doKubeadm(d, "join", joinConfigBytes, extraArgs...),
	}
	return ssh.DoComposed(actions...)
}

// doDeleteLocalKubeconfig deletes the current, local kubeconfig, doing a backup before
func doDeleteLocalKubeconfig(d *schema.ResourceData) ssh.Applyer {
	kubeconfig := getKubeconfig(d)
	kubeconfigBak := kubeconfig + ".bak"

	return ssh.DoIf(
		ssh.CheckLocalFileExists(kubeconfig),
		ssh.DoComposed(
			ssh.DoMessage("Removing local kubeconfig (with backup)"),
			ssh.DoMoveLocalFile(kubeconfig, kubeconfigBak)),
	)
}

// doDownloadKubeconfig downloads a kubeconfig from the remote master
func doDownloadKubeconfig(d *schema.ResourceData) ssh.Applyer {
	kubeconfig := getKubeconfig(d)
	return ssh.DoDownloadFile(ssh.DefAdminKubeconfig, kubeconfig)
}
