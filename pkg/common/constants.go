package common

const (
	DefPodCIDR = "10.244.0.0/16"

	DefServiceCIDR = "10.96.0.0/12"

	DefKubernetesVersion = "v1.14.1"

	DefDNSDomain = "cluster.local"

	DefRuntimeEngine = "docker"

	DefAdminKubeconfig = "/etc/kubernetes/admin.conf"

	DefKubeadmInitConfPath = "/etc/kubernetes/kubeadm-init.conf"

	DefKubeadmJoinConfPath = "/etc/kubernetes/kubeadm-join.conf"

	DefCniConfDir = "/etc/cni/net.d"

	DefCniLookbackConfPath = "/etc/cni/net.d/99-loopback.conf"

	DefCniBinDir = "/opt/cni/bin"

	// Full path where we should upload the kubelet sysconfig file
	DefKubeletSysconfigPath = "/etc/sysconfig/kubelet"

	// Full path where we should upload the kubelet.service file
	DefKubeletServicePath = "/usr/lib/systemd/system/kubelet.service"

	// Full path where we should upload the kubeadm dropin file
	DefKubeadmDropinPath = "/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf"

	DefAPIServerPort = 6443

	// TODO: add a manifest for loading Helm
	DefHelmManifest = ""

	// manifest for loading the dashboard
	DefDashboardManifest = "https://raw.githubusercontent.com/kubernetes/dashboard/master/aio/deploy/recommended/kubernetes-dashboard.yaml"

	// kubeadm executable in the machines (we assume it is in some standard path)
	DefKubeadmPath = "kubeadm"
)

var (
	// CNIPluginsManifests is the map of manifests for different CNI drivers
	CNIPluginsManifests = map[string]string{
		"flannel": "https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml",
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
