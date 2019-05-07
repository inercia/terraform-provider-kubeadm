package assets

//go:generate ../../utils/generate.sh --out-var KubeadmSetupScriptCode --out-package assets  --out-file generated_kubeadm_setup.go ./static/kubeadm-setup.sh
//go:generate ../../utils/generate.sh --out-var KubeletSysconfigCode --out-package assets --out-file generated_kubelet_sysconfig.go ./static/kubelet.sysconfig
//go:generate ../../utils/generate.sh --out-var KubeadmDropinCode --out-package assets --out-file generated_kubeadm_dropin.go ./static/kubeadm-dropin.conf
//go:generate ../../utils/generate.sh --out-var KubeletServiceCode --out-package assets --out-file generated_kubelet_service.go ./static/service.conf
//go:generate ../../utils/generate.sh --out-var CNIDefConfCode --out-package assets --out-file generated_cni_conf.go ./static/cni-default.conflist
