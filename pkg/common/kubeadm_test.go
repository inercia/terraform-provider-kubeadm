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
	"fmt"
	"testing"
)

func TestInitConfigSerialization(t *testing.T) {
	configContents := `
apiVersion: kubeadm.k8s.io/v1beta1
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 82eb2m.999999idy9l74yha
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServer:
  timeoutForControlPlane: 4m0s
apiVersion: kubeadm.k8s.io/v1beta1
certificatesDir: /etc/kubernetes/pki
clusterName: kubernetes
controlPlaneEndpoint: ""
controllerManager: {}
dns:
  type: CoreDNS
etcd:
  local:
    dataDir: /var/lib/etcd
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: v1.14.1
networking:
  dnsDomain: cluster.local
  serviceSubnet: 10.96.0.0/12
`

	initConfig, err := YAMLToInitConfig([]byte(configContents))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if initConfig.BootstrapTokens[0].Token.String() != "82eb2m.999999idy9l74yha" {
		t.Fatalf("Error: wrong bootstrap token: %v", initConfig.BootstrapTokens[0].Token.String())
	}
	if initConfig.KubernetesVersion != "v1.14.1" {
		t.Fatalf("Error: wrong kubernetes version: %v", initConfig.KubernetesVersion)
	}

	configContentsAgain, err := InitConfigToYAML(initConfig)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("----------------- init configuration ---------------- \n%s", configContentsAgain)

	if len(configContentsAgain) == 0 {
		t.Fatalf("wrong serialized contents: %s", configContentsAgain)
	}
}

func TestJoinConfigSerialization(t *testing.T) {
	configContents := `
apiVersion: kubeadm.k8s.io/v1beta1
caCertPath: /etc/kubernetes/pki/ca.crt
discovery:
  bootstrapToken:
    token: e5927b.cd71ba4602956ef3
    unsafeSkipCAVerification: true
  timeout: 15m0s
  tlsBootstrapToken: "e5927b.cd71ba4602956ef3"
kind: JoinConfiguration
controlPlane:
  nodeRegistration:
    localAPIEndpoint:
      advertiseAddress: 10.10.0.1
`

	joinConfig, err := YAMLToJoinConfig([]byte(configContents))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if joinConfig.Discovery.BootstrapToken.Token != "e5927b.cd71ba4602956ef3" {
		t.Fatalf("Error: wrong bootstrap token: %v", joinConfig.Discovery.BootstrapToken.Token)
	}

	configContentsAgain, err := JoinConfigToYAML(joinConfig)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("----------------- join configuration ---------------- \n%s", configContentsAgain)
}
