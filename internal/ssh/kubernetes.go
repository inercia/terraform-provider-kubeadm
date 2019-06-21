package ssh

import (
	"fmt"
	"log"
	"os/exec"
)

// DoLocalKubectl runs a local kubectl command
func DoLocalKubectl(kubeconfig string, args ...string) ApplyFunc {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		log.Fatal("kubectl not available")
	}

	f := append([]string{fmt.Sprintf("--kubeconfig=%s", kubeconfig)}, args...)
	return DoLocalExec(kubectl, f...)
}

// DoLocalKubectlApply applies some manifests with a local kubectl
func DoLocalKubectlApply(kubeconfig string, manifests []string) ApplyFunc {
	loaders := []Applyer{}
	for _, manifest := range manifests {
		if len(manifest) == 0 {
			continue
		}
		loaders = append(loaders, DoLocalKubectl(kubeconfig, "apply", "-f", manifest))
	}
	return ApplyComposed(loaders...)
}
