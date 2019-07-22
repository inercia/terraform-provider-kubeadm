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

package provisioner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	// the `etcdctl` command we must run
	etcdctlCommand = "ETCDCTL_API=3 etcdctl"

	// the local etcd endpoint used for some commands
	localEtcdEndpointIP   = "127.0.0.1"
	localEtcdEndpointPort = 2379

	// pattern for getting the etcd container
	etcContainerPattern = "k8s_etcd_etcd"

	// common arguments for etcdctl
	// note: these arguments are valid IFF using "ETCDCTL_API=3" is defined in the environment
	argsCommon = "--cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt --key=/etc/kubernetes/pki/etcd/healthcheck-client.key --cacert=/etc/kubernetes/pki/etcd/ca.crt"

	// command for getting the endpoints
	subcmdEndpointsList = "endpoint status"

	// command for getting the members list
	subcmdMembersList = "member list"

	// command for removing a member
	subcmdMemberRemove = "member remove"
)

var (
	// the etcd server is not running or we count not find it
	ErrNoEtcdContainer = errors.New("could not find a etcd container")

	ErrParsingEtcdOutput = errors.New("error parsing etcd output")
)

// runEtcdctlSubcommand runs a etcdctl command
func DoRunEtcdctlSubcommand(subcommand string, args ...string) ssh.Action {
	argEndpoints := fmt.Sprintf("--endpoints=https://%s:%d", localEtcdEndpointIP, localEtcdEndpointPort)

	// build the full `etcdctl` command to run in the container
	fullEtcdctlCommand := fmt.Sprintf("%s %s %s %s %s",
		etcdctlCommand, argsCommon, argEndpoints, subcommand, strings.Join(args, " "))

	return ssh.DoDockerExec(etcContainerPattern, fullEtcdctlCommand)
}

//////////////////////////////////////////////////////////////////////////

type EtcdEndpoint struct {
	ID       string
	Endpoint url.URL
	IsLeader bool
}

func (ep EtcdEndpoint) String() string {
	return fmt.Sprintf("%s %s (leader:%t)", ep.ID, ep.Endpoint.String(), ep.IsLeader)
}

func (ep *EtcdEndpoint) FromString(s string) error {
	// parse something like
	//
	// https://127.0.0.1:2379, e942f75ad6f00855, 3.3.10, 1.8 MB, true, 2, 24139
	//
	// where:
	//+------------------------+------------------+---------+---------+-----------+-----------+------------+
	//|        ENDPOINT        |        ID        | VERSION | DB SIZE | IS LEADER | RAFT TERM | RAFT INDEX |
	//+------------------------+------------------+---------+---------+-----------+-----------+------------+
	//| https://127.0.0.1:2379 | e942f75ad6f00855 |  3.3.10 |  1.8 MB |      true |         2 |      24122 |
	//+------------------------+------------------+---------+---------+-----------+-----------+------------+
	res := strings.Split(s, ",")
	if len(res) != 7 {
		ssh.Debug("cannot parse as endpoint info: %q", s)
		return ErrParsingEtcdOutput
	}

	isLeader, err := strconv.ParseBool(strings.TrimSpace(res[4]))
	if err != nil {
		return err
	}

	u, err := url.Parse(strings.TrimSpace(res[0]))
	if err != nil {
		return err
	}

	ep.Endpoint = *u
	ep.ID = strings.TrimSpace(res[1])
	ep.IsLeader = isLeader

	return nil
}

type EtcdEndpointsSet map[string]EtcdEndpoint

// FromString gets a set of endpoints from a string
func (endpoints *EtcdEndpointsSet) FromString(s string) (err error) {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		ep := EtcdEndpoint{}
		if err := ep.FromString(line); err != nil {
			return err
		}
		ssh.Debug("adding etcd endpoint: %+v", ep)
		(*endpoints)[ep.ID] = ep
	}
	return
}

// GetLocalEndpoint get the info for the local endpoint
// we do that by going through all the endpoint and checking which one
// has 127.0.0.1 in the address...
func (endpoints EtcdEndpointsSet) GetLocalEndpoint() EtcdEndpoint {
	var localEndpoint EtcdEndpoint
	for _, ep := range endpoints {
		if ep.Endpoint.Hostname() == localEtcdEndpointIP {
			// update the local endpoint info, specially the ID
			ssh.Debug("updating info for local etcd instance: %+v", ep)
			localEndpoint = ep
		}
	}
	return localEndpoint
}

/////////////////////////////////////////////////////////////////////////////////////////

// DoGetEndpointsList gets the list of endpoints in the etcd cluster
func DoGetEndpointsList(eps *EtcdEndpointsSet) ssh.Action {
	var buf bytes.Buffer
	return ssh.ActionList{
		ssh.DoSendingExecOutputToWriter(&buf, DoRunEtcdctlSubcommand(subcmdEndpointsList)),
		ssh.ActionFunc(func(ctx context.Context) ssh.Action {
			err := eps.FromString(buf.String())
			if err != nil {
				return ssh.ActionError(err.Error())
			}
			return nil
		}),
	}
}

// doRemoveIfMember removes this node from the etcd cluster iff it was a member
func doRemoveIfMember(d *schema.ResourceData) ssh.Action {
	eps := EtcdEndpointsSet{}
	return ssh.ActionList{
		ssh.DoMessageInfo("Checking if we must delete the node from the etcd cluster..."),
		ssh.DoIfElse(
			ssh.CheckContainerRunning(etcContainerPattern),
			ssh.ActionList{
				DoGetEndpointsList(&eps),
				ssh.ActionFunc(func(ctx context.Context) ssh.Action {
					if len(eps) == 0 {
						return ssh.DoMessageWarn("could not get list of etcd endpoints")
					}
					return nil
				}),
				ssh.ActionFunc(func(ctx context.Context) ssh.Action {
					localEndpoint := eps.GetLocalEndpoint()
					if localEndpoint.ID == "" {
						return ssh.DoMessageWarn("could not find the local etcd endpoint details")
					}

					// now we have the etcd ID for the etcd instance running in this machine
					// we can run the "member remove <ID>"
					return ssh.ActionList{
						ssh.DoMessageInfo("Removing %q from the etcd cluster", localEndpoint.ID),
						DoRunEtcdctlSubcommand(subcmdMemberRemove, localEndpoint.ID),
						ssh.DoMessageInfo("%q has been removed from the etcd cluster", localEndpoint.ID),
					}
				}),
			},
			ssh.ActionList{
				ssh.DoMessageInfo("etcd is not running in this node: no need to remove it from the etcd cluster"),
			},
		),
	}
}

// doPrintEtcdStatus prints the status of etcd, if running
func doPrintEtcdStatus(d *schema.ResourceData) ssh.Action {
	eps := EtcdEndpointsSet{}
	return ssh.DoIfElse(
		ssh.CheckContainerRunning(etcContainerPattern),
		ssh.ActionList{
			ssh.DoMessageInfo("Checking status of etcd (if running)..."),
			DoGetEndpointsList(&eps),
			ssh.ActionFunc(func(ctx context.Context) ssh.Action {
				if len(eps) == 0 {
					return ssh.DoMessageWarn("could not get list of etcd endpoints")
				}
				prints := ssh.ActionList{ssh.DoMessageInfo("%d etcd endpoints available:", len(eps))}
				for _, ep := range eps {
					prints = append(prints, ssh.DoMessageInfo("- %s", ep.String()))
				}
				return prints
			})},
		ssh.ActionList{
			ssh.DoMessageInfo("etcd does not seem to be running in this node"),
		},
	)
}
