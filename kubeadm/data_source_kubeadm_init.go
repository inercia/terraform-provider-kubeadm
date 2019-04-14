package kubeadm

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

const (
	defaultAPIServerPort = 6443
)

// dataSourceKubeadmReadToInitConfig copies some settings from the
// Terraform `data` definition to a kubeadm Init configuration
func dataSourceKubeadmReadToInitConfig(d *schema.ResourceData, token string) ([]byte, error) {
	log.Printf("[DEBUG] [KUBEADM] creating initialization configuration...")

	initConfig := &kubeadmapiv1beta1.InitConfiguration{
		ClusterConfiguration: kubeadmapiv1beta1.ClusterConfiguration{
			// FeatureGates:         featureGates,
			APIServer: kubeadmapiv1beta1.APIServer{
				CertSANs: []string{},
			},
			UseHyperKubeImage: true,
		},
		NodeRegistration: kubeadmapiv1beta1.NodeRegistrationOptions{},
	}

	if _, ok := d.GetOk("api.0"); ok {
		if external, ok := d.GetOk("api.0.external"); ok {
			initConfig.ControlPlaneEndpoint = addressWithPort(external.(string), defaultAPIServerPort)
		}

		if internal, ok := d.GetOk("api.0.internal"); ok {
			host, port, err := net.SplitHostPort(internal.(string))
			if err != nil {
				return nil, err
			}

			initConfig.LocalAPIEndpoint.AdvertiseAddress = host
			if port != "" {
				i, err := strconv.Atoi(port)
				if err != nil {
					return nil, err
				}
				initConfig.LocalAPIEndpoint.BindPort = int32(i)
			}

			initConfig.ClusterConfiguration.APIServer.CertSANs = append(initConfig.ClusterConfiguration.APIServer.CertSANs, host)
		}

		if altNames, ok := d.GetOk("api.0.alt_names"); ok {
			initConfig.APIServer.CertSANs = append(initConfig.APIServer.CertSANs, altNames.([]string)...)
		}
	}

	if _, ok := d.GetOk("network.0"); ok {
		if podCIDROpt, ok := d.GetOk("network.0.pods"); ok {
			initConfig.Networking.PodSubnet = podCIDROpt.(string)
		}
		if servicesCIDROpt, ok := d.GetOk("network.0.services"); ok {
			initConfig.Networking.ServiceSubnet = servicesCIDROpt.(string)
		}
		if dnsDomainOpt, ok := d.GetOk("network.0.dns_domain"); ok {
			initConfig.Networking.DNSDomain = dnsDomainOpt.(string)
		}
	}

	if _, ok := d.GetOk("images.0"); ok {
		kube_repo := d.Get("images.0.kube_repo").(string)
		initConfig.ClusterConfiguration.ImageRepository = kube_repo

		etcd_repo := d.Get("images.0.etcd_repo").(string)
		etcd_version := d.Get("images.0.etcd_version").(string)
		if etcd_version != "" || etcd_repo != "" {
			initConfig.ClusterConfiguration.Etcd = kubeadmapiv1beta1.Etcd{
				Local: &kubeadmapiv1beta1.LocalEtcd{
					ImageMeta: kubeadmapiv1beta1.ImageMeta{
						ImageRepository: etcd_repo,
						ImageTag:        etcd_version,
					},
				},
			}
		}
	}

	if _, ok := d.GetOk("extra_args.0"); ok {
		if args, ok := d.GetOk("extra_args.0.api_server"); ok {
			initConfig.ClusterConfiguration.APIServer.ExtraArgs = args.(map[string]string)
		}
		if args, ok := d.GetOk("extra_args.0.controller_manager"); ok {
			initConfig.ClusterConfiguration.ControllerManager.ExtraArgs = args.(map[string]string)
		}
		if args, ok := d.GetOk("extra_args.0.scheduler"); ok {
			initConfig.ClusterConfiguration.Scheduler.ExtraArgs = args.(map[string]string)
		}
		if args, ok := d.GetOk("extra_args.0.kubelet"); ok {
			initConfig.NodeRegistration.KubeletExtraArgs = args.(map[string]string)
		}
	}

	if versionOpt, ok := d.GetOk("version"); ok {
		initConfig.KubernetesVersion = versionOpt.(string)
	}

	if _, ok := d.GetOk("etcd.0"); ok {
		if etcdServersLst, ok := d.GetOk("etcd.0.endpoints"); ok {
			if initConfig.Etcd.External == nil {
				initConfig.Etcd.External = &kubeadmapiv1beta1.ExternalEtcd{}
			}
			initConfig.Etcd.External.Endpoints = etcdServersLst.([]string)
		}
	}

	if len(token) > 0 {
		var err error
		bto := kubeadmapiv1beta1.BootstrapToken{}
		bto.Token, err = kubeadmapiv1beta1.NewBootstrapTokenString(token)
		if err != nil {
			return nil, err
		}
		initConfig.BootstrapTokens = []kubeadmapiv1beta1.BootstrapToken{bto}
	}

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

// return an address as host:port (setting a default port p if there was no port specified)
func addressWithPort(name string, p int) string {
	if strings.IndexByte(name, ':') < 0 {
		return net.JoinHostPort(name, fmt.Sprintf("%d", p))
	}
	return name
}
