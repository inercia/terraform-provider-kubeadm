package kubeadm

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-kubeadm/internal/kubernetes"
)

const (
	defaultPodCIDR           = "10.244.0.0/16"
	defaultServiceCIDR       = "10.3.0.0/24"
	defaultKubernetesVersion = "v1.6.1"
	defaultMasterPort        = 6443
	defaultDNSDomain         = "cluster.local"
	defaultAuthMode          = "RBAC"
	defaultAPIVersion        = "kubeadm.k8s.io/v1alpha1"
)

func dataSourceKubeadm() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubeadmRead,

		Schema: map[string]*schema.Schema{
			"etcd_servers": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "List of etcd servers URLs including host:port",
			},
			"api_advertised": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "API server advertised IP/name",
			},
			"api_port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     defaultMasterPort,
				Description: "API server binding port",
			},
			"api_alt_names": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "List of SANs to use in api-server certificate. Example: 'IP=127.0.0.1,IP=127.0.0.2,DNS=localhost', If empty, SANs will be extracted from the api_servers",
			},
			"pods_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultPodCIDR,
				Description: "The CIDR range of cluster pods.",
			},
			"services_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultServiceCIDR,
				Description: "The CIDR range of cluster services.",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultKubernetesVersion,
				Description: "Kubernetes version to use (Example: v1.6.0).",
			},
			"dns_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultDNSDomain,
				Description: "The DNS domain.",
			},
			"cloud_provider": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The provider for cloud services.  Empty string for no provider.",
			},
			"authorization_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultAuthMode,
				Description: "Authentication mode (Example: RBAC).",
			},
			"extra_args": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_server": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "List of extra flags for running the API server",
						},
						"controller_manager": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "List of extra flags for running the API server",
						},
						"scheduler": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "List of extra flags for running the API server",
						},
					},
				},
			},
			"config": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"master": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKubeadmRead(d *schema.ResourceData, meta interface{}) error {
	masterConfig := kubernetes.MasterConfiguration{}
	nodeConfig := kubernetes.NodeConfiguration{}

	// Generate a valid token
	log.Printf("[DEBUG] Generating a token")
	token, err := kubernetes.GenerateToken()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] kubeadm token = %s", token)
	masterConfig.Token = token
	nodeConfig.Token = token

	log.Printf("[DEBUG] Parsing kubeadm configuration")
	if podCIDROpt, ok := d.GetOk("pods_cidr"); ok {
		masterConfig.Networking.PodSubnet = podCIDROpt.(string)
	}
	if servicesCIDROpt, ok := d.GetOk("services_cidr"); ok {
		masterConfig.Networking.ServiceSubnet = servicesCIDROpt.(string)
	}
	if cloudProviderOpt, ok := d.GetOk("cloud_provider"); ok {
		masterConfig.CloudProvider = cloudProviderOpt.(string)
	}
	if versionOpt, ok := d.GetOk("version"); ok {
		masterConfig.KubernetesVersion = versionOpt.(string)
	}
	if bindPortOpt, ok := d.GetOk("api_port"); ok {
		masterConfig.API.BindPort = int32(bindPortOpt.(int))
	}
	if advertisedOpt, ok := d.GetOk("api_advertised"); ok {
		masterConfig.API.AdvertiseAddress = advertisedOpt.(string)
	}
	if dnsDomainOpt, ok := d.GetOk("dns_domain"); ok {
		masterConfig.Networking.DNSDomain = dnsDomainOpt.(string)
	}
	if etcdServersLst, ok := d.GetOk("etcd_servers"); ok {
		masterConfig.Etcd.Endpoints = etcdServersLst.([]string)
	}
	if authorizationModeOpt, ok := d.GetOk("authorization_mode"); ok {
		masterConfig.AuthorizationMode = authorizationModeOpt.(string)
	}
	if apiServersAltNamesOpt, ok := d.GetOk("api_alt_names"); ok {
		masterConfig.APIServerCertSANs = apiServersAltNamesOpt.([]string)
	}
	if versionOpt, ok := d.GetOk("version"); ok {
		masterConfig.KubernetesVersion = versionOpt.(string)
	}
	// TODO: parse the extra_args.api_server
	//if apiExtraOpt, ok := d.GetOk("api_alt_names"); ok {
	//	masterConfig.APIServerExtraArgs = apiExtraOpt.([]string)
	//}

	// some defaults we need to set
	masterConfig.APIVersion = defaultAPIVersion
	masterConfig.Kind = "MasterConfiguration"
	masterConfig.SelfHosted = true
	nodeConfig.APIVersion = defaultAPIVersion
	nodeConfig.Kind = "NodeConfiguration"

	log.Printf("[DEBUG] Rendering master config as a JSON")
	mc, err := json.Marshal(masterConfig)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Rendering node config as a JSON")
	nc, err := json.Marshal(nodeConfig)
	if err != nil {
		return err
	}

	config := map[string]string{
		"master": string(mc[:]),
		"node":   string(nc[:]),
	}

	d.Set("config", config)
	d.SetId(token)

	return nil
}
