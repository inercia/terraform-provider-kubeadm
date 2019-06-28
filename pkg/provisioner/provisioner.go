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
			action = doKubeadmJoin(d, true)
		case "worker":
			action = doKubeadmJoin(d, false)
		case "":
			action = doKubeadmJoin(d, false)
		default:
			o.Output(fmt.Sprintf("Unknown provisioning profile: join is %q and role is %q", join, role))
			return ErrUnknownProvisioningProfile
		}
	}

	return action.Apply(o, comm, useSudo)
}

// doKubeadmInit runs the `kubeadm init`
func doKubeadmInit(d *schema.ResourceData) ssh.ApplyFunc {
	_, initConfigBytes, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.DoAbort(fmt.Sprintf("could not get a valid 'config' for init'ing: %s", err))
	}
	extraArgs := []string{}

	actions := []ssh.Applyer{
		ssh.DoMessage("Initializing the cluster with 'kubadm init'"),
		doDeleteLocalKubeconfig(d),
		doUploadCerts(d),
		ssh.DoIfElse(
			ssh.CheckFileExists(common.DefAdminKubeconfig),
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

// doKubeadmJoin runs the `kubeadm join`
func doKubeadmJoin(d *schema.ResourceData, controlPlane bool) ssh.ApplyFunc {
	_, joinConfigBytes, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.DoAbort(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// check if we are joining the Control Plane: we must upload the certificates and
	// use the '--control-plane' flag
	extraArgs := []string{}
	controlPlaneAction := ssh.DoNothing()
	if controlPlane {
		controlPlaneAction = doUploadCerts(d)
		extraArgs = append(extraArgs, "--experimental-control-plane")
	}

	actions := []ssh.Applyer{
		ssh.DoMessage("Joining the cluster with 'kubadm join'"),
		controlPlaneAction,
		doKubeadm(d, "join", joinConfigBytes, extraArgs...),
	}
	return ssh.DoComposed(actions...)
}

// doDeleteLocalKubeconfig deletes the current, local kubeconfig, doing a backup before
func doDeleteLocalKubeconfig(d *schema.ResourceData) ssh.ApplyFunc {
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
func doDownloadKubeconfig(d *schema.ResourceData) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	return ssh.DoDownloadFile(common.DefAdminKubeconfig, kubeconfig)
}
