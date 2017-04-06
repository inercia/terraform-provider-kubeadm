package kubeadm

// note: we could use go-bindata too, but that would mean we would have to include an extra dependency
//       just for this... I think it is not worth.
//go:generate ../../utils/generate.sh --out-var setupScript --out-package kubeadm  setup.sh

const (
	// the path where kubeadm will be installed/linked
	defaultKubeadmExe = "/tmp/kubeadm"
)

