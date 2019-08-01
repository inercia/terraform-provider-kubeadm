// Copyright © 2019 Alvaro Saurin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

func dataSourceKubeadm() *schema.Resource {
	return &schema.Resource{
		Create: dataSourceKubeadmCreate,
		Read:   dataSourceKubeadmRead,
		Delete: dataSourceKubeadmDelete,
		//Update: dataSourceKubeadmUpdate,
		Exists: dataSourceKubeadmExists,

		Schema: map[string]*schema.Schema{
			"config_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A local copy of the kubeconfig",
			},
			"api": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"external": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "stable IP/DNS (and port) for the control plane (for example, the load balancer)",
							ValidateFunc: common.ValidateDNSNameOrIP,
						},
						"internal": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "IP/DNS and port the local API server advertises it's accessible",
							ValidateFunc: common.ValidateDNSNameOrIP,
						},
						"alt_names": {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "List of SANs to use in api-server certificate. Example: 'IP=127.0.0.1,IP=127.0.0.2,DNS=localhost', If empty, SANs will be obtained from the external and internal names/IPs",
						},
					},
				},
			},
			"addons": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"helm": {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "install Helm",
						},
						"dashboard": {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "install the Kubernetes Dashboard",
						},
					},
				},
			},
			"cni": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"plugin": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "",
							Description:  "CNI plugin to install. Currently supported: flannel",
							ValidateFunc: validation.StringInSlice(common.CNIPluginsList, true),
						},
						"plugin_manifest": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Use a specific manifest for the CNI driver instead of the pre-defined manifests",
						},
						"bin_dir": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefCniBinDir,
							Description:  "Binaries directory for CNI",
							ValidateFunc: common.ValidateAbsPath,
						},
						"conf_dir": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefCniConfDir,
							Description:  "Configuration directory for CNI",
							ValidateFunc: common.ValidateAbsPath,
						},
						"flannel": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"backend": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  common.DefFlannelBackend,
										// see https://github.com/coreos/flannel/blob/master/Documentation/backends.md
										Description:  "Flannel backend: vxlan, host-gw, udp, ali-vpc, aws-vpc, gce, ipip, ipsec",
										ValidateFunc: validation.StringInSlice([]string{"vxlan", "host-gw", "udp", "ali-vpc", "aws-vpc", "gce", "ipip", "ipsec"}, true),
									},
									"version": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  common.DefFlannelBackend,
										// see https://github.com/coreos/flannel/blob/master/Documentation/backends.md
										Description:  "Flannel backend: vxlan, host-gw, udp, ali-vpc, aws-vpc, gce, ipip, ipsec",
										ValidateFunc: validation.StringInSlice([]string{"vxlan", "host-gw", "udp", "ali-vpc", "aws-vpc", "gce", "ipip", "ipsec"}, true),
									},
								},
							},
						},
						"weave": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									// no options yet
								},
							},
						},
					},
				},
			},
			"network": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"services": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefServiceCIDR,
							Description:  "subnet used by k8s services. Defaults to 10.96.0.0/12.",
							ValidateFunc: validation.CIDRNetwork(0, 32),
						},
						"pods": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefPodCIDR,
							Description:  "subnet used by pods",
							ValidateFunc: validation.CIDRNetwork(0, 32),
						},
						"dns_domain": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefDNSDomain,
							Description:  "DNS domain used by k8s services. Defaults to cluster.local.",
							ValidateFunc: common.ValidateDNSName,
						},
					},
				},
			},
			"images": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kube_repo": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "the kubernetes images repository",
						},
						"etcd_repo": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "the etcd image repository",
						},
						"etcd_version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "the etcd version",
						},
					},
				},
			},
			"etcd": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"endpoints": {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "list of etcd servers URLs including host:port",
						},
					},
				},
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     common.DefKubernetesVersion,
				ForceNew:    true,
				Description: "Kubernetes version to use (Example: v1.15.0).",
			},
			"cloud": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  fmt.Sprintf("cloud provider name: %s", strings.Join(common.DefSupportedCloudProviders, ",")),
							ValidateFunc: validation.StringInSlice(common.DefSupportedCloudProviders, true),
						},
						"config": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: fmt.Sprintf("cloud provider configuration (mandatory for: %s)", strings.Join(common.DefCloudConfigMandatory, ",")),
						},
						"manager_flags": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "additional arguments for the cloud-controller-manager",
						},
					},
				},
			},
			"runtime": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"engine": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      common.DefRuntimeEngine,
							Description:  "runtime engine: docker, containerd or crio",
							ValidateFunc: validation.StringInSlice([]string{"crio", "containerd", "docker"}, true),
						},
						"extra_args": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"api_server": {
										Type:        schema.TypeMap,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Optional:    true,
										Description: "Map of extra flags for running the API server",
									},
									"controller_manager": {
										Type:        schema.TypeMap,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Optional:    true,
										Description: "Map of extra flags for running the Controller Manager",
									},
									"scheduler": {
										Type:        schema.TypeMap,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Optional:    true,
										Description: "Map of extra flags for running the Scheduler",
									},
									"kubelet": {
										Type:        schema.TypeMap,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Optional:    true,
										Description: "Map of extra flags for running the Kubelet",
									},
								},
							},
						},
					},
				},
			},
			"certs": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_crt": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"ca_key": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"sa_crt": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"sa_key": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"etcd_crt": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"etcd_key": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"proxy_crt": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"proxy_key": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
					},
				},
			},
			// the "config" must be a map of string that will be passed to the "provisioner"
			"config": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: common.ProvisionerConfigElements,
				},
			},
		},
	}
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"kubeadm": dataSourceKubeadm(),
		},
	}
}
