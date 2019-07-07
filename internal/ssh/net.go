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

package ssh

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// a "portable" command that can print the list of IPs ... this should work in "most" linuxes (fingers crossed)
	ipAddressesCmd = `ip addr show | grep -Po 'inet \K[\d.]+' || hostname --all-ip-addresses || hostname -I`
)

// AllMatchesIPv4 matches all the IPs in a string
func AllMatchesIPv4(s string, ipAddresses *[]string) error {
	re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	submatchall := re.FindAllString(s, -1)
	for _, element := range submatchall {
		*ipAddresses = append(*ipAddresses, element)
	}
	return nil
}

// DoGetIpAddresses gets the list of IP addresses
func DoGetIpAddresses(ipAddresses *[]string) Action {
	var buf bytes.Buffer
	return ActionList{
		DoMessageDebug(fmt.Sprintf("Getting list of IP addresses with %q", ipAddressesCmd)),
		DoSendingOutputToWriter(DoExec(ipAddressesCmd), &buf),
		ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			// note: this must be inside of a function for being lazy-evaluated (when we have obtained the IPs)
			if err := AllMatchesIPv4(buf.String(), ipAddresses); err != nil {
				return ActionError(err.Error())
			}
			return nil
		}),
	}
}

func DoPrintIpAddresses() Action {
	ipAddresses := []string{}
	return ActionList{
		DoGetIpAddresses(&ipAddresses),
		ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			// note: this must be inside of a function for being lazy-evaluated (when we have obtained the IPs)
			return DoMessage(fmt.Sprintf("IP addresses detected by the kubeadm provisioner: %s", strings.Join(ipAddresses, ", ")))
		}),
	}
}
