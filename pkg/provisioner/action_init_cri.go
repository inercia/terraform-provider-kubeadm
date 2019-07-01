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

package provisioner

import (
	"strings"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doPrepareCRI preparse the CRI in the target node
func doPrepareCRI() ssh.ApplyFunc {
	return ssh.DoComposed(
		ssh.DoUploadReaderToFile(strings.NewReader(assets.CNIDefConfCode), common.DefCniLookbackConfPath),
		// we must reload the containers runtime engine after changing the CNI configuration
		ssh.DoIf(
			ssh.CheckServiceExists("crio.service"),
			ssh.DoRestartService("crio.service")),
		ssh.DoIf(
			ssh.CheckServiceExists("docker.service"),
			ssh.DoRestartService("docker.service")),
	)
}
