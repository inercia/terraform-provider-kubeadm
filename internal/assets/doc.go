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

package assets

//go:generate ../../utils/generate.sh --out-var KubeadmSetupScriptCode --out-package assets  --out-file generated_kubeadm_setup.go ./static/kubeadm-setup.sh
//go:generate ../../utils/generate.sh --out-var KubeletSysconfigCode --out-package assets --out-file generated_kubelet_sysconfig.go ./static/kubelet.sysconfig
//go:generate ../../utils/generate.sh --out-var KubeadmDropinCode --out-package assets --out-file generated_kubeadm_dropin.go ./static/kubeadm-dropin.conf
//go:generate ../../utils/generate.sh --out-var KubeletServiceCode --out-package assets --out-file generated_kubelet_service.go ./static/service.conf
//go:generate ../../utils/generate.sh --out-var CNIDefConfCode --out-package assets --out-file generated_cni_conf.go ./static/cni-default.conflist
//go:generate ../../utils/generate.sh --out-var FlannelManifestCode --out-package assets --out-file generated_flannel_manifest.go ./static/kube-flannel.yml
