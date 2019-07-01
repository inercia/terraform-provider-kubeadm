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
	"log"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	tokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
)

var (
	ErrLoadingKubeconfig = errors.New("could not load admin kubeconfig file")
)

func GetClientSet(kubeconfig string) (*clientset.Clientset, error) {
	client, err := kubeconfigutil.ClientSetFromFile(kubeconfig)
	if err != nil {
		return nil, ErrLoadingKubeconfig
	}
	return client, nil
}

// IsClusterAlive returns true if the kubeconfig provided points to a API server that responds to requests
func IsClusterAlive(kubeconfig string) bool {
	// try to load the kubeconfig and access the API server
	client, err := GetClientSet(kubeconfig)
	if err != nil {
		log.Printf("[DEBUG] [KUBEADM] could not load the kubeconfig %s: %s", kubeconfig, err)
		return false
	}
	if _, err = client.CoreV1().Nodes().List(metav1.ListOptions{}); err != nil {
		log.Printf("[DEBUG] [KUBEADM] could not load get the nodes list from the cluster: %s", err)
		return false
	}
	return true
}

// CreateOrRefreshToken creates a new token or refreshes the old one
func CreateOrRefreshToken(client *clientset.Clientset, token string) error {
	var err error

	bto := kubeadmapi.BootstrapToken{}
	bto.Token, err = kubeadmapi.NewBootstrapTokenString(token)
	if err != nil {
		return err
	}
	tokens := []kubeadmapi.BootstrapToken{bto}

	log.Printf("[DEBUG] [KUBEADM] creating (or refreshing an existing) token")
	if err := tokenphase.UpdateOrCreateTokens(client, false, tokens); err != nil {
		log.Printf("[DEBUG] [KUBEADM] error when refreshing token: %s", err)
		return err
	}

	return nil
}
