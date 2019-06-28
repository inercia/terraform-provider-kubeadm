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
