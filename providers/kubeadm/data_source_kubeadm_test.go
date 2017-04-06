package kubeadm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestKubeadm_basic(t *testing.T) {
	const testAccKubeadm_basic = `
data "kubeadm_config" "k8s" {
    name = "test"
}`

	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubeadm_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckState("kubeadm_config.k8s"),
				),
			},
		},
	})
}

// check that a key exists in the state
func testAccCheckState(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		return nil
	}
}
