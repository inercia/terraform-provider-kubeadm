package kubeadm

import (
	"bytes"
	"fmt"
	"log"
	"path"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doKubeadmJoin performs a `kubeadm join` in the remote host
func doKubeadmJoin(d *schema.ResourceData, configFile []byte) ssh.ApplyFunc {
	configPath := path.Join(defKubeadmJoinConfPath)
	extraArgs := ""
	extraArgs += " " + getKubeadmIgnoredChecksArg(d)
	extraArgs += " " + getKubeadmNodenameArg(d)

	return ssh.Composite(
		ssh.ActionIf(
			ssh.CheckFileExists(configPath),
			ssh.DoExec("kubeadm reset --force")),
		ssh.DoUploadFile(bytes.NewReader(configFile), configPath),
		ssh.DoExec(fmt.Sprintf("kubeadm join --config=%s %s", configPath, extraArgs)),
	)
}

// unmarshallJoinConfig unmarshalls the joinConfiguration passed from
// the kubeadm `data` resource
func unmarshallJoinConfig(d *schema.ResourceData) (*kubeadmapiv1beta1.JoinConfiguration, []byte, error) {

	var err error
	var joinConfig *kubeadmapiv1beta1.JoinConfiguration

	config := d.Get("config").(string)
	seeder := d.Get("join").(string)

	// deserialize the configuration saved in the `config`
	configBytes, err := FromTerraformSafeString(config)
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
	joinConfig.Discovery.BootstrapToken.APIServerEndpoint = addressWithPort(seeder, defAPIServerPort)

	kubeadmscheme.Scheme.Default(joinConfig)
	configBytes, err = kubeadmutil.MarshalToYamlForCodecs(joinConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}

	// NOTE: we are currently not really using the unmarshalled joinConfiguration
	log.Printf("[DEBUG] [KUBEADM] join config:\n%s\n%", configBytes)

	return joinConfig, configBytes, nil
}
