package kubeadm

const (
	defPodCIDR = "10.244.0.0/16"

	defServiceCIDR = "10.96.0.0/12"

	defKubernetesVersion = "v1.14.1"

	defDNSDomain = "cluster.local"

	defRuntimeEngine = "docker"

	defAdminKubeconfig = "/etc/kubernetes/admin.conf"

	defKubeadmInitConfPath = "/etc/kubernetes/kubeadm-init.conf"

	defKubeadmJoinConfPath = "/etc/kubernetes/kubeadm-join.conf"

	defCniConfDir = "/etc/cni/net.d"

	defCniLookbackConfPath = "/etc/cni/net.d/99-loopback.conf"

	defCniBinDir = "/opt/cni/bin"

	defKubeadmDropinPath = "/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf"

	defAPIServerPort = 6443
)

var (
	// DefaultCriSocket info
	defCriSocket = map[string]string{
		"docker":     "/var/run/dockershim.sock",
		"crio":       "/var/run/crio/crio.sock",
		"containerd": "/var/run/containerd/containerd.sock",
	}

	defIgnorePreflightChecks = []string{
		"NumCPU",
		"FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
		"Swap",
		"FileExisting-crictl",
		"Port-10250",
		"SystemVerification", // for ignoring docker graph=btrfs
		"IsPrivilegedUser",
		"NumCPU", // we will not always have >=2 CPUs in our VMs
	}

	defKubeletSettings = map[string]string{
		"network-plugin": "cni",
	}
)
