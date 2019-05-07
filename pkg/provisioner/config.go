package provisioner

import (
	"bytes"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// unmarshallInitConfig unmarshalls the initConfiguration passed from
// the kubeadm `data` resource
func unmarshallInitConfig(d *schema.ResourceData) (*kubeadmapiv1beta1.InitConfiguration, []byte, error) {
	var initConfig *kubeadmapiv1beta1.InitConfiguration
	var clusterConfig *kubeadmapiv1beta1.ClusterConfiguration

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
	objects, err := kubeadmutil.SplitYAMLDocuments(configBytes)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range objects {
		if kubeadmutil.GroupVersionKindsHasInitConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.InitConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			initConfig = cfg2
		} else if kubeadmutil.GroupVersionKindsHasClusterConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.ClusterConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			clusterConfig = cfg2
		}
	}
	initConfig.ClusterConfiguration = *clusterConfig

	initbytes, err := kubeadmutil.MarshalToYamlForCodecs(initConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}
	allFiles := [][]byte{initbytes}

	clusterbytes, err := kubeadmutil.MarshalToYamlForCodecs(&initConfig.ClusterConfiguration, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}
	allFiles = append(allFiles, clusterbytes)

	configBytes = bytes.Join(allFiles, []byte(kubeadmconstants.YAMLDocumentSeparator))

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

	// load the initConfiguration from the `config` field
	objects, err := kubeadmutil.SplitYAMLDocuments(configBytes)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range objects {
		if kubeadmutil.GroupVersionKindsHasJoinConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.JoinConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			joinConfig = cfg2
		}
	}

	// update some things, like the seeder
	joinConfig.Discovery.BootstrapToken.APIServerEndpoint = common.AddressWithPort(seeder, common.DefAPIServerPort)

	kubeadmscheme.Scheme.Default(joinConfig)
	configBytes, err = kubeadmutil.MarshalToYamlForCodecs(joinConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}

	// NOTE: we are currently not really using the unmarshalled joinConfiguration
	log.Printf("[DEBUG] [KUBEADM] join config:\n%s\n", configBytes)

	return joinConfig, configBytes, nil
}
