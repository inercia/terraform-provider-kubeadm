package common

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestCertsSerialization(t *testing.T) {
	etcdCrtContents := "1234567890"

	certsMap := map[string]interface{}{
		"certs_secret":    "some-secret",
		"certs_dir":       "/etc/kubernetes/pki",
		"certs_ca_crt":    "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"certs_ca_key":    "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"certs_sa_crt":    "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"certs_sa_key":    "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"certs_etcd_crt":  etcdCrtContents,
		"certs_etcd_key":  "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
		"certs_proxy_crt": "-- BEGIN PUBLIC KEY ---\n SOME-CERT ...",
		"certs_proxy_key": "-- BEGIN PRIVATE KEY ---\n SOME-KEY ...",
	}

	certsConfig := CertsConfig{}
	if certsConfig.IsFilled() {
		t.Fatalf("Error: certsConfig seems to be filled")
	}

	err := certsConfig.FromMap(certsMap)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("certs config object:\n%s", spew.Sdump(certsConfig))

	if !certsConfig.IsFilled() {
		t.Fatalf("Error: certsConfig seems to be empty")
	}

	certsMap2, err := certsConfig.ToMap()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("certs config map:\n%s", spew.Sdump(certsMap2))
	certContents, ok := certsMap2["certs_etcd_crt"]
	if !ok {
		t.Fatalf("Error: certs_etcd_crt not in map")
	}
	if certContents != etcdCrtContents {
		t.Fatalf("Error: etcd_crt does not match")
	}
}
