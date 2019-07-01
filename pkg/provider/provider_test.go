package provider

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestKubeadm_basic(t *testing.T) {
	const testAccKubeadm_basic = `
        resource "kubeadm" "k8s" {
        	config_path = "/tmp/kubeconfig"
        	
        	network {
        		services = "10.25.0.0/16"
        	}
        	
            api {
              external = "loadbalancer.external.com"
            }
	
	        cni {
				plugin = "flannel"
	        }
        }`

	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubeadm_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckState("kubeadm.k8s"),
					resource.TestCheckResourceAttr("kubeadm.k8s",
						"config.config_path",
						"/tmp/kubeconfig"),
					resource.TestCheckResourceAttr("kubeadm.k8s",
						"config.cni_plugin",
						"flannel"),
					resource.TestCheckResourceAttrSet("kubeadm.k8s",
						"config.init"),
					resource.TestCheckResourceAttrSet("kubeadm.k8s",
						"config.join"),
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
			log.Printf("%s", s.RootModule())
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		return nil
	}
}
