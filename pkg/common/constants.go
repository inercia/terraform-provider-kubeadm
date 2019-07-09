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

package common

import (
	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
)

const (
	DefPodCIDR = "10.244.0.0/16"

	DefServiceCIDR = "10.96.0.0/12"

	DefKubernetesVersion = "v1.14.1"

	DefDNSDomain = "cluster.local"

	DefRuntimeEngine = "docker"

	DefKubeadmInitConfPath = "/etc/kubernetes/kubeadm-init.conf"

	DefKubeadmJoinConfPath = "/etc/kubernetes/kubeadm-join.conf"

	DefCniConfDir = "/etc/cni/net.d"

	DefCniLookbackConfPath = "/etc/cni/net.d/99-loopback.conf"

	DefCniBinDir = "/opt/cni/bin"

	DefFlannelBackend = "vxlan"

	DefFlannelImageVersion = "v0.11.0"

	// Full path where we should upload the kubelet sysconfig file
	DefKubeletSysconfigPath = "/etc/sysconfig/kubelet"

	// Full path where we should upload the kubelet.service file
	DefKubeletServicePath = "/usr/lib/systemd/system/kubelet.service"

	// Full path where we should upload the kubeadm dropin file
	DefKubeadmDropinPath = "/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf"

	// Default PKI dir
	DefPKIDir = "/etc/kubernetes/pki"

	DefAPIServerPort = 6443

	// TODO: add a manifest for loading Helm
	DefHelmManifest = ""

	// manifest for loading the dashboard
	DefDashboardManifest = "https://raw.githubusercontent.com/kubernetes/dashboard/v1.10.1/src/deploy/recommended/kubernetes-dashboard.yaml"

	// kubeadm executable in the machines (we assume it is in some standard path)
	DefKubeadmPath = "kubeadm"

	// kubectl executable in the machines (we assume it is in some standard path)
	DefKubectlPath = "kubectl"
)

var (
	// CNIPluginsManifestsTemplates is the map of manifests for different CNI drivers
	CNIPluginsManifestsTemplates = map[string]string{
		"flannel": assets.FlannelManifestCode,
	}
)

var (
	// DefaultCriSocket info
	DefCriSocket = map[string]string{
		"docker":     "/var/run/dockershim.sock",
		"crio":       "/var/run/crio/crio.sock",
		"containerd": "/var/run/containerd/containerd.sock",
	}

	DefIgnorePreflightChecks = []string{
		"NumCPU",
		"FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
		"Swap",
		"FileExisting-crictl",
		"Port-10250",
		"SystemVerification", // for ignoring docker graph=btrfs
		"IsPrivilegedUser",
		"NumCPU", // we will not always have >=2 CPUs in our VMs
	}

	DefKubeletSettings = map[string]string{
		"network-plugin": "cni",
	}
)
