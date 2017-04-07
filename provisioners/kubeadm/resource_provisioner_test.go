package kubeadm

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = new(ResourceProvisioner)
}

// builds and returns a terraform.ResourceConfig object pointer from a map of generic types
func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"master": "someurl.com:6443",
		"config": "this is a temporal config",
	})

	r := new(ResourceProvisioner)
	warn, errs := r.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings were not expected")
	}

	if len(errs) > 0 {
		t.Fatalf("Errors were not expected")
	}
}

func TestResourceProvider_Parse_good(t *testing.T) {
	p := new(ResourceProvisioner)
	cfgMaster := testConfig(t, map[string]interface{}{
		"config": "{\"api\":{\"advertiseAddress\":\"\",\"bindPort\":0},\"etcd\":{\"endpoints\":null,\"caFile\":\"\",\"certFile\":\"\",\"keyFile\":\"\"},\"networking\":{\"serviceSubnet\":\"10.25.0.0/16\",\"podSubnet\":\"10.2.0.0/16\",\"dnsDomain\":\"\"},\"kubernetesVersion\":\"\",\"cloudProvider\":\"\",\"authorizationMode\":\"\",\"token\":\"6d1dfe.3b31148157ed2d1d\",\"tokenTTL\":0,\"selfHosted\":false,\"apiServerExtraArgs\":null,\"controllerManagerExtraArgs\":null,\"schedulerExtraArgs\":null,\"apiServerCertSANs\":null,\"certificatesDir\":\"\"}",
	})
	cfgMinion := testConfig(t, map[string]interface{}{
		"master": "someurl.com:6443",
		"config": "{\"caCertPath\":\"\",\"discoveryFile\":\"\",\"discoveryToken\":\"\",\"discoveryTokenAPIServers\":null,\"tlsBootstrapToken\":\"\",\"token\":\"6d1dfe.3b31148157ed2d1d\"}",
	})

	// parse the master configuration
	err := p.loadFromResourceConfig(cfgMaster)
	if err != nil {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("could not get provisioner parameters from config: %s", err)
	}
	if len(p.Master) > 0 {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("master address in master")
	}
	if len(p.masterConfig.Token) == 0 {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("no token obtained in master")
	}
	if p.nodeConfig != nil && len(p.nodeConfig.Token) > 0 {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("token in nodeConfig when parsing a node")
	}

	// parse the minion configuration
	err = p.loadFromResourceConfig(cfgMinion)
	if err != nil {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("could not get provisioner parameters from config: %s", err)
	}
	if p.Master != "someurl.com:6443" {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("could not get master address")
	}
	if p.masterConfig != nil && len(p.masterConfig.Token) > 0 {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("token in masterConfig when parsing a node")
	}
	if len(p.nodeConfig.Token) == 0 {
		t.Logf("provisioner:\n%s", spew.Sdump(p))
		t.Fatalf("no token obtained in node")
	}
}
