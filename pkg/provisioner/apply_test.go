package provisioner

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"config": (interface{})(map[string]interface{}{
			"cni_plugin": "flannel",
		}),
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"ups": "something",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) == 0 && len(errs) == 0 {
		t.Fatalf("Errors: %v Warnings: %v", errs, warn)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
