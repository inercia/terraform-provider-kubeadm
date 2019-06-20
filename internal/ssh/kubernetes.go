package ssh

import (
	"fmt"
	"log"
	"os/exec"
)

// DoLocalKubectlApply applies some manifests with a local kubectl
func DoLocalKubectlApply(kubeconfig string, manifests []string) ApplyFunc {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		log.Fatal("kubectl not available")
	}

	loaders := []Applyer{}
	for _, manifest := range manifests {
		if len(manifest) == 0 {
			continue
		}

		loaders = append(loaders,
			DoLocalExec(kubectl,
				fmt.Sprintf("--kubeconfig=%s", kubeconfig),
				"apply", "-f", manifest))
	}
	return ApplyComposed(loaders...)
}
