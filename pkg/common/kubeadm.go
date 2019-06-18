package common

import (
	"bytes"
	"log"

	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

//
// Init
//

// YAMLToInitConfig converts a YAML to InitConfiguration
func YAMLToInitConfig(configBytes []byte) (*kubeadmapiv1beta1.InitConfiguration, error) {
	var initConfig *kubeadmapiv1beta1.InitConfiguration
	var clusterConfig *kubeadmapiv1beta1.ClusterConfiguration

	objects, err := kubeadmutil.SplitYAMLDocuments(configBytes)
	if err != nil {
		return nil, err
	}
	for k, v := range objects {
		if kubeadmutil.GroupVersionKindsHasInitConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.InitConfiguration)
			if !ok || cfg2 == nil {
				return nil, err
			}

			initConfig = cfg2
		} else if kubeadmutil.GroupVersionKindsHasClusterConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.ClusterConfiguration)
			if !ok || cfg2 == nil {
				return nil, err
			}

			clusterConfig = cfg2
		}
	}

	if initConfig != nil && clusterConfig != nil {
		initConfig.ClusterConfiguration = *clusterConfig
	}

	return initConfig, nil
}

// InitConfigToYAML converts a InitConfiguration to YAML
func InitConfigToYAML(initConfig *kubeadmapiv1beta1.InitConfiguration) ([]byte, error) {
	log.Printf("[DEBUG] [KUBEADM] creating initialization configuration...")

	kubeadmscheme.Scheme.Default(initConfig)

	initbytes, err := kubeadmutil.MarshalToYamlForCodecs(initConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return []byte{}, err
	}
	allFiles := [][]byte{initbytes}

	clusterbytes, err := kubeadmutil.MarshalToYamlForCodecs(&initConfig.ClusterConfiguration, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return []byte{}, err
	}
	allFiles = append(allFiles, clusterbytes)

	return bytes.Join(allFiles, []byte(kubeadmconstants.YAMLDocumentSeparator)), nil
}

//
// Join
//

// YAMLToJoinConfig converts a YAML to JoinConfiguration
func YAMLToJoinConfig(configBytes []byte) (*kubeadmapiv1beta1.JoinConfiguration, error) {
	var joinConfig *kubeadmapiv1beta1.JoinConfiguration

	objects, err := kubeadmutil.SplitYAMLDocuments(configBytes)
	if err != nil {
		return nil, err
	}
	for k, v := range objects {
		if kubeadmutil.GroupVersionKindsHasJoinConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.JoinConfiguration)
			if !ok || cfg2 == nil {
				return nil, err
			}

			joinConfig = cfg2
		}
	}

	return joinConfig, nil
}

// JoinConfigToYAML converts a JoinConfiguration to YAML
func JoinConfigToYAML(joinConfig *kubeadmapiv1beta1.JoinConfiguration) ([]byte, error) {

	kubeadmscheme.Scheme.Default(joinConfig)
	nodebytes, err := kubeadmutil.MarshalToYamlForCodecs(joinConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return []byte{}, err
	}

	return nodebytes, nil
}
