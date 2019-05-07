package provisioner

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProvisioners map[string]terraform.ResourceProvisioner
var testAccProvisioner *terraform.ResourceProvisioner

func init() {
	testAccProvisioner = Provisioner().(*terraform.ResourceProvisioner)
	testAccProvisioners = map[string]terraform.ResourceProvisioner{
		"kubeadm": testAccProvisioner,
	}
}

func TestProvider(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provisioner()
}

func testAccPreCheck(t *testing.T) {
}
