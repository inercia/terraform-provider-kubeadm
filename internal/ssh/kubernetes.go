package ssh

// DoLocalKubectlApply applies some manifests with a local kubectl
func DoLocalKubectlApply(kubeconfig string, manifests []string) ApplyFunc {
	loaders := []Applyer{}
	for _, manifest := range manifests {
		if len(manifest) == 0 {
			continue
		}
		localExecFunc := DoLocalExec("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", manifest)
		loaders = append(loaders, localExecFunc)
	}
	return ApplyComposed(loaders...)
}
