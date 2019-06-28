package provisioner

import (
	"fmt"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

func getMasterNodes(kubeconfig string) (*v1.NodeList, error) {
	clientSet, err := common.GetClientSet(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get admin client set")
	}

	return clientSet.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=", kubeadmconstants.LabelNodeRoleMaster),
	})
}

func isMaster(node *v1.Node) bool {
	_, isMaster := node.ObjectMeta.Labels[kubeadmconstants.LabelNodeRoleMaster]
	return isMaster
}

// doLocalKubectl runs a local kubectl with the kubeconfig specified in the schema
func doLocalKubectl(d *schema.ResourceData, args ...string) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	return ssh.DoLocalKubectl(kubeconfig, args...)
}

// DoLocalKubectlApply applies some manifests with a local kubectl with the kubeconfig specified in the schema
func doLocalKubectlApply(d *schema.ResourceData, manifests []string) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, manifests)
}

// doRefreshToken uses the kubeconfig for connecting to the API server and refreshing the token
func doRefreshToken(d *schema.ResourceData) ssh.ApplyFunc {
	token, ok := d.GetOk("config.token")
	if !ok {
		panic("there should be a token")
	}

	return ssh.DoIfElse(
		checkKubeconfigAlive(d),
		ssh.ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
			// load the existing kubeconfig and use it for refreshing the token
			client, err := common.GetClientSet(getKubeconfig(d))
			if err != nil {
				return err
			}

			err = common.CreateOrRefreshToken(client, token.(string))
			if err != nil {
				return err
			}
			return nil
		}),
		ssh.DoAbort("no valid kubeconfig exists or the cluster is not alive/reachable: the token not refreshed, so the node cannot join the cluster"),
	)
}

// doMaybePublishCertificates publishes the certificates (if needed)
func doMaybePublishCertificates(d *schema.ResourceData, initConfig *kubeadmapi.InitConfiguration) ssh.ApplyFunc {
	return ssh.DoIfElse(
		checkKubeconfigAlive(d),
		ssh.ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
			if err := RetrieveAndUploadCerts(d, initConfig); err != nil {
				return err
			}
			return nil
		}),
		ssh.DoAbort("no valid kubeconfig exists or the cluster is not alive/reachable: certificates cannot be uploaded to the API server"),
	)
}

// checkKubeconfigExists checks if the kubeconfig exists
func checkKubeconfigExists(d *schema.ResourceData) ssh.CheckerFunc {
	return ssh.CheckLocalFileExists(getKubeconfig(d))
}

// checkKubeconfigAlive checks if the kubeconfig exists and is alive
func checkKubeconfigAlive(d *schema.ResourceData) ssh.CheckerFunc {
	kubeconfig := getKubeconfig(d)
	return ssh.CheckAnd(
		checkKubeconfigExists(d),
		ssh.CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
			return common.IsClusterAlive(kubeconfig), nil
		}))
}
