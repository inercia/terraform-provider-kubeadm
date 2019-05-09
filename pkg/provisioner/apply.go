package provisioner

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
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

	log.Printf("[DEBUG] [KUBEADM] kubeadm provisioner: configuration:\n%s\n", spew.Sdump(d))
	log.Printf("[DEBUG] [KUBEADM] connection:\n%s\n", spew.Sdump(connData))
	log.Printf("[DEBUG] [KUBEADM] instance state:\n%s\n", spew.Sdump(s))

	// ensure that this is a linux machine
	if s.Ephemeral.ConnInfo["type"] != "ssh" {
		return fmt.Errorf("Unsupported connection type: %s. This provisioner currently only supports linux", s.Ephemeral.ConnInfo["type"])
	}

	join := d.Get("join").(string)
	preventSudo := d.Get("prevent_sudo").(bool)
	useSudo := !preventSudo && s.Ephemeral.ConnInfo["user"] != "root"

	// build a communicator for the provisioner to use
	comm, err := getCommunicator(ctx, o, s)
	if err != nil {
		o.Output("Error when creating communicator")
		return err
	}

	if _, ok := d.GetOk("install"); ok {
		auto := d.Get("install.0.auto").(bool)
		if auto {
			script := d.Get("install.0.script").(string)
			if err := doKubeadmSetup(o, comm, script, useSudo); err != nil {
				return err
			}
		}
	}

	// TODO: d has a `diff.Destroy` field... maybe we could use that for knowing
	// if we want to destroy the object

	if !d.IsNewResource() {
		o.Output(fmt.Sprintf("WARNING: %q seems to be an old resource: running kubeadm anyway...", d.Id()))

		// TODO: I guess we could detect if we need a `kubeadm reset`
		//  with `d.HasChange("config")`...
	}

	// run kubeadm init/join
	if len(join) == 0 {
		_, kubeadmConfig, err := unmarshallInitConfig(d)
		if err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Initializing the cluster with 'kubadm init'"))
		return ssh.ApplyList([]ssh.Applyer{
			doCommonProvisioning(),
			doKubeadmInit(d, kubeadmConfig),
			doDownloadKubeconfig(d),
			doLoadCNI(d),
			doLoadDashboard(d),
			doLoadHelm(d),
			doLoadManifests(d),
		}, o, comm, useSudo)
	} else {
		_, kubeadmConfig, err := unmarshallJoinConfig(d)
		if err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Joining the cluster with 'kubadm join'"))
		return ssh.ApplyList([]ssh.Applyer{
			doCommonProvisioning(),
			doKubeadmJoin(d, kubeadmConfig),
		}, o, comm, useSudo)
	}
}

// doCommonProvisioning are the common provisioning things, for the `init` as well
// as for the `join`.
func doCommonProvisioning() ssh.ApplyFunc {
	return ssh.ApplyComposed(
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadFile(strings.NewReader(assets.KubeletSysconfigCode), "/etc/sysconfig/kubelet"),
		ssh.DoUploadFile(strings.NewReader(assets.KubeletServiceCode), "/usr/lib/systemd/system/kubelet.service"),
		ssh.DoUploadFile(strings.NewReader(assets.KubeadmDropinCode), common.DefKubeadmDropinPath),
	)
}

// doPrepareCRI preparse the CRI in the target node
func doPrepareCRI() ssh.ApplyFunc {
	return ssh.ApplyComposed(
		ssh.DoUploadFile(strings.NewReader(assets.CNIDefConfCode), common.DefCniLookbackConfPath),
		// we must reload the containers runtime engine after changing the CNI configuration
		ssh.ApplyIf(
			ssh.CheckServiceExists("crio.service"),
			ssh.DoRestartService("crio.service")),
		ssh.ApplyIf(
			ssh.CheckServiceExists("docker.service"),
			ssh.DoRestartService("docker.service")),
	)
}
