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

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

func TestKubeadmInitConfigSerialization(t *testing.T) {
	d := schema.ResourceData{}

	token := "82eb2m.999999idy9l74yha"

	d.Set("api.0.internal", "10.10.0.1")
	d.Set("network.0.dns_domain", "my-local.cluster")

	initConfig, err := dataSourceToInitConfig(&d, token)
	if err != nil {
		t.Fatalf("could not create initConfig from dataSource: %s", err)
	}

	if initConfig.BootstrapTokens[0].Token.String() != token {
		t.Fatalf("Error: wrong bootstrap token: %v", initConfig.BootstrapTokens[0].Token.String())
	}

	initConfigBytes, err := common.InitConfigToYAML(initConfig)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("----------------- init configuration ---------------- \n%s", initConfigBytes)

}
