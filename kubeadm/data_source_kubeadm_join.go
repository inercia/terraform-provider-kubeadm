package kubeadm

import (
	"github.com/hashicorp/terraform/helper/schema"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

// dataSourceKubeadmReadToJoinConfig copies some settings to a Join configuration
func dataSourceKubeadmReadToJoinConfig(d *schema.ResourceData, token string) ([]byte, error) {
	joinConfig := &kubeadmapiv1beta1.JoinConfiguration{
		NodeRegistration: kubeadmapiv1beta1.NodeRegistrationOptions{},
		Discovery: kubeadmapiv1beta1.Discovery{
			BootstrapToken: &kubeadmapiv1beta1.BootstrapTokenDiscovery{
				APIServerEndpoint:        "", // this should be filled in the provisioner
				Token:                    token,
				UnsafeSkipCAVerification: true,
			},
		},
	}

	if _, ok := d.GetOk("extra_args.0"); ok {
		if args, ok := d.GetOk("extra_args.0.kubelet"); ok {
			joinConfig.NodeRegistration.KubeletExtraArgs = args.(map[string]string)
		}
	}

	kubeadmscheme.Scheme.Default(joinConfig)
	nodebytes, err := kubeadmutil.MarshalToYamlForCodecs(joinConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return []byte{}, err
	}

	return nodebytes, nil
}
