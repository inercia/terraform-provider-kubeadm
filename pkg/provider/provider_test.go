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

func TestKubeadm_certs(t *testing.T) {
	const testAccKubeadm_basic = `
        resource "kubeadm" "k8s" {
        	config_path = "/tmp/kubeconfig"

			certs {
				ca_crt =<<EOF
-----BEGIN CERTIFICATE-----
MIICwjCCAaqgAwIBAgIBADANBgkqhkiG9w0BAQsFADASMRAwDgYDVQQDEwdldGNk
LWNhMB4XDTE5MDYyODE1NTM1M1oXDTI5MDYyNTE1NTM1M1owEjEQMA4GA1UEAxMH
ZXRjZC1jYTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALWBzEuUirW2
jJ4S6G2jBeiNWrRVMc87QJdoCjsz6giWJ4LT+IhTi21Q7Qo0Aw8c6PpTVE4VsMYz
Ul2+o4gBn6YrAUaMz036rlzPDW6/mci8O0TDJU9K2gHsP33UngGm3dp37Z/TLAK5
cMUO4W593CCFDRe2AmYfBRRwGXtH6eS9Lj6PucamJvNvX+D+F15WfU7sqVzZnIV+
DC8upfhUGlYaIIWjD6zO1NOrmt8oP5R3GKK7ri62YY/C/uZbWPMsSs9MxTsWrL0x
orV95nUnsLdEk7mBluM9nRD8GtjdoNEtgBx5Z4lPkcGFSY3UQhCh2OZkKD1LzQON
aBOo52Bgf+0CAwEAAaMjMCEwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMB
Af8wDQYJKoZIhvcNAQELBQADggEBAAhRY7nL5JO1IOufu+C+38HmN1R8bxO7B3Yf
+WhBRoMouZ/7uCncfolmTmOki9wL/+oxoyPUdkao67RLgVknVoP0piTglf0tcxSE
ucaU2Q9PUmE5AJFXXbkbBHiES/yV92t8BqRDkVjeZoTxeBsT46ncVYDWBKYLvulO
D8xnvcMweN6wyDniZiKjb7DeJq5xvqnSq3vrF4vjElovxeV8pTC7ckEilP9FApf5
Fv5zVNHFpgjpHH8/IL73ZyYlG6RqyfPeoR53sVzfd4XWdnQi8eK/Pf+uE+iy0O2r
QasRiemsP8IWvOwcKGViNUC2Ag5EEh8S8PMlLP+3/pzkEPfIHgk=
-----END CERTIFICATE-----
EOF

				ca_key =<<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzU/cJiB3/Fr85gW6dpCbDQTyZpVuB8LtyfwFHhsUrCpVJ/U0
B4lfH2n8E8VB62SeGtaXcbYnScNgkaDgQ+SvkHlIDjp16Z3cFjHLOUyF3cVOBbDs
blUOjD/21Pq9LwWs+dSwfHQ1/ZxNNYolRUizwvSI8GW1tJJT41MJS8yp8AC5za8R
Qeev49rZlWYc19nU6fBPZ3xBpR7ofTLQ4YjQmhIEsN1taJrkgYfy7u4mT3iec4kR
O2Wwuxs1OJJsnSyIaw70PRDdebkD4Nioz/egHYjNIcWIsRTFAa3S/zPPFnTU+Dq1
nahJ8kIUxzAMdgV4PWjMhdu5pnT23d5XqwEoIwIDAQABAoIBAQC41URkLqbWUTOM
AWw0gUqVFfcD01MTObHJPVF+IPMja5juOBl3D3zLUybUxajqudJ8ZuRAQrRr+7Bc
anB7rs0/S3BLHuY4Qx13/avvEa0SUiZDiVvQmFJYgN0+L91RD9MBtzCLWjOg9a2s
nYmgLitnP65ofahvv6w14vNjggUbQm48ozXlSbORnj9o6bBmAKvR3cdxONkD3vl4
0Oa/2Ez6VVXxH/pc6lYji0k0e6v8egO6fc6CtaMQcOYR1ji23SIoBkihWjvmDQ27
Jsk/jFDgP4lW6ojEK47VBZ9iNjiWIjS/YO50woDMPz6jl/gO6pFspBpC+nG/Qd15
uulPFMSBAoGBANXyZwaTyx7yLwBJwpPGabmgiPu1pgLLyt4PQNU1/kqA2zR7ieDY
qKD6BEX4cnGP5KqtzOdqfMcq1QtEVzqylyz/3Aw3vD821DM6iyM6k1JeC1jqA3lu
fQR/ugWUzlAIRcbX8QU8qbwQYqppE0oZzAJ02tASK9JKyhcVy5fwGX4ZAoGBAPWq
9MqrVh1fszQJMg6Aef3XDkkK3x89nGRMpXH/5SK0/a0n78WkXwmavRGGK0+zk4jo
5Ddrvc9UaeOL50kMDhSeboD2fwNpEp5IJQdX/0FwsTy4Yibwchx91kc4YxbL+eYY
si5btsfbRvuxopeigSxfCpJgXe2z9XMMYm8nvCebAoGAPuJr89v3BRaMSBpmDcdx
BfWwrcN7kzDRZSm4lbK0FrP/OlLheOxVzFMQdHyNLuHrhVtmcdKz8FqfmhsxRHh/
xONDi3fKZg44mwInKWirKrenwC+wa73VE0BzrfZKGe4EjGimWDK3dSafyZTu7YXd
mA8+zY+5v6rp8ZUfbX5OD+kCgYEA3ejaDERumkP6/RMdW0okZ+4d4k7mszKVFWjC
vdI36Xzx9LqxdKeAjY1wIec/MlR0/WPZ2lIBd8m5iKi0eCBii699BBMlMjB0d/OV
Nyf+0972ynGHf8MMYL4uk9DUeSAxkO5X7VY9KhTh7rNLuos5AZqsUwKndfNr0Mus
EtoitOcCgYEAkFyH5t7qReE3I8g/kY7jIdxP9z6qC/aNFeZxIMd7PKxSn6fnSf/v
LKW7DhCZCgEb3ZDotYPdJmdRt7QBB5jIj4ZoyCzxlkyoAC2WAROWYaUQz9kuVP9f
OflxkYMD4H/BuT3uuX4BR0Ko32wAyNn/AJmgekiPjQ/NGfwG0CS2fGY= 
-----END RSA PRIVATE KEY-----
EOF
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
					testAccCheckAttr("kubeadm.k8s", "certs.0.ca_key"),
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

// check that an attribute exists in the state
func testAccCheckAttr(id string, attr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			log.Printf("%s", s.RootModule())
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		attrs := rs.Primary.Attributes
		if _, ok := attrs[attr]; !ok {
			return fmt.Errorf("No attribute %q found in %q", attr, id)
		}
		return nil
	}
}
