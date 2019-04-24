package kubeadm

//go:generate ../utils/generate.sh --out-var kubeletSysconfigCode --out-package kubeadm --out-file generated_kubelet_sysconfig.go ./assets/kubelet.sysconfig
//go:generate ../utils/generate.sh --out-var kubeadmDropinCode --out-package kubeadm --out-file generated_kubeadm_dropin.go ./assets/kubeadm-dropin.conf
//go:generate ../utils/generate.sh --out-var kubeletServiceCode --out-package kubeadm --out-file generated_kubelet_service.go ./assets/service.conf
//go:generate ../utils/generate.sh --out-var CniDefConfCode --out-package kubeadm --out-file generated_cni_conf.go ./assets/cni-default.conflist

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"

	"github.com/inercia/terraform-kubeadm/internal/ssh"
)

var (
	errNoConfig = errors.New("no config provided")
)

func init() {
	spew.Config.Indent = "\t"
}

type ResourceProvisioner struct {
	initConfig *kubeadmapiv1beta1.InitConfiguration
	joinConfig *kubeadmapiv1beta1.JoinConfiguration

	comm    communicator.Communicator
	useSudo bool
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
	log.Printf("[DEBUG] [KUBEADM] connection:\n%s\n%", spew.Sdump(connData))
	log.Printf("[DEBUG] [KUBEADM] instance state:\n%s\n%", spew.Sdump(s))

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

	// TODO: I guess we could detect if we need a `kubeadm reset`
	//  with `d.HasChange("config")`...

	// run kubeadm init/join
	if len(join) == 0 {
		_, configFile, err := unmarshallInitConfig(d)
		if err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Initializing the cluster with 'kubadm init'"))
		return ssh.RunActions([]ssh.Action{
			doCommonProvisioning(),
			doKubeadmInit(d, configFile),
			doDownloadKubeconfig(d, configFile),
			doLoadManifests(d, configFile),
		}, o, comm, useSudo)
	} else {
		_, configFile, err := unmarshallJoinConfig(d)
		if err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Joining the cluster with 'kubadm join'"))
		return ssh.RunActions([]ssh.Action{
			doCommonProvisioning(),
			doKubeadmJoin(d, configFile),
		}, o, comm, useSudo)
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// getCommunicator gets a new communicator for the remote machine
func getCommunicator(ctx context.Context, o terraform.UIOutput, s *terraform.InstanceState) (communicator.Communicator, error) {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return nil, err
	}

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	return comm, err
}
