package provisioner

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// unmarshallInitConfig unmarshalls the initConfiguration passed from
// the kubeadm `data` resource
func unmarshallInitConfig(d *schema.ResourceData) (*kubeadmapiv1beta1.InitConfiguration, []byte, error) {

	cfg, ok := d.GetOk("config.init")
	if !ok {
		return nil, nil, errNoInitConfigFound
	}

	// deserialize the configuration saved in the `config`
	configBytes, err := common.FromTerraformSafeString(cfg.(string))
	if err != nil {
		return nil, nil, err
	}

	// load the initConfiguration from the `config` field
	initConfig, err := common.YAMLToInitConfig(configBytes)
	if err != nil {
		return nil, nil, err
	}

	// ... update some things, like the seeder, the nodename, etc
	if nodenameOpt, ok := d.GetOk("nodename"); ok {
		initConfig.NodeRegistration.Name = nodenameOpt.(string)
	}

	configBytes, err = common.InitConfigToYAML(initConfig)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] [KUBEADM] init config:\n%s\n", configBytes)
	return initConfig, configBytes, nil
}

// unmarshallJoinConfig unmarshalls the joinConfiguration passed from
// the kubeadm `data` resource
func unmarshallJoinConfig(d *schema.ResourceData) (*kubeadmapiv1beta1.JoinConfiguration, []byte, error) {

	var err error
	var joinConfig *kubeadmapiv1beta1.JoinConfiguration

	seeder := d.Get("join").(string)
	cfg, ok := d.GetOk("config.join")
	if !ok {
		return nil, nil, errNoJoinConfigFound
	}

	// deserialize the configuration saved in the `config`
	configBytes, err := common.FromTerraformSafeString(cfg.(string))
	if err != nil {
		return nil, nil, err
	}

	joinConfig, err = common.YAMLToJoinConfig(configBytes)
	if err != nil {
		return nil, nil, err
	}

	// ... update some things, like the seeder, the nodename, etc
	joinConfig.Discovery.BootstrapToken.APIServerEndpoint = common.AddressWithPort(seeder, common.DefAPIServerPort)
	if nodenameOpt, ok := d.GetOk("nodename"); ok {
		joinConfig.NodeRegistration.Name = nodenameOpt.(string)
	}

	/// ... and serialize again
	configBytes, err = common.JoinConfigToYAML(joinConfig)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] [KUBEADM] join config:\n%s\n", configBytes)
	return joinConfig, configBytes, nil
}
