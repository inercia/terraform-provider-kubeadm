package provisioner

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	// the local etcd endpoint used for some commands
	localEtcdEndpoint = "127.0.0.1"

	// docker command for getting the etcd container
	dockerGetEtcdContainer = "docker ps --filter name=^/k8s_etcd_etcd -q"

	// common arguments for etcdctl
	// note: these arguments are valid when using ETCDCTL_API=3
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
)

type EtcdEndpoint struct {
	ID       string
	Endpoint string
	IsLeader bool
}

type EtcdMember struct {
	ID         string
	Name       string
	Status     string
	PeerURL    string
	ClienAddrs string
}

func getEndpointFromString(s string) (EtcdEndpoint, error) {
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
	isLeader, err := strconv.ParseBool(strings.TrimSpace(res[4]))
	if err != nil {
		return EtcdEndpoint{}, err
	}

	return EtcdEndpoint{
		Endpoint: strings.TrimSpace(res[0]),
		ID:       strings.TrimSpace(res[1]),
		IsLeader: isLeader,
	}, nil
}

func getMemberFromString(s string) (EtcdMember, error) {
	// parse something like
	//
	// e942f75ad6f00855, started, kubeadm-master-0, https://172.30.0.2:2380, https://172.30.0.2:2379
	//
	// where:
	//+------------------+---------+------------------+-------------------------+-------------------------+
	//|        ID        | STATUS  |       NAME       |       PEER ADDRS        |      CLIENT ADDRS       |
	//+------------------+---------+------------------+-------------------------+-------------------------+
	//| e942f75ad6f00855 | started | kubeadm-master-0 | https://172.30.0.2:2380 | https://172.30.0.2:2379 |
	//+------------------+---------+------------------+-------------------------+-------------------------+
	res := strings.Split(s, ",")
	return EtcdMember{
		ID:         strings.TrimSpace(res[0]),
		Status:     strings.TrimSpace(res[1]),
		Name:       strings.TrimSpace(res[2]),
		PeerURL:    strings.TrimSpace(res[3]),
		ClienAddrs: strings.TrimSpace(res[4]),
	}, nil
}

// getEtcdContainer returns the etcd container ID
func getEtcdContainer(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (string, error) {
	log.Printf("[DEBUG] [KUBEADM] Getting the etcd container...")

	output := []string{}
	var interceptor ssh.OutputFunc = func(s string) {
		output = append(output, s)
	}

	if err := ssh.DoExec(dockerGetEtcdContainer).Apply(interceptor, comm, useSudo); err != nil {
		return "", err
	}

	if len(output) == 0 {
		log.Printf("[DEBUG] [KUBEADM] etcd does not seem to be running in this machine")
		return "", ErrNoEtcdContainer
	}

	cont := output[0]
	cont = strings.TrimSpace(cont)
	log.Printf("[DEBUG] etcd detected running in container: '%s'", cont)
	return cont, nil
}

// getEndpointsArgFor returns the `etcdctl` argument for a list of etcd endpoints
func getEndpointsArgFor(addrs []string) (string, error) {
	urls := []string{}
	for _, addr := range addrs {
		urls = append(urls, fmt.Sprintf("https://[%s]:2379", addr))
	}
	return "--endpoints=" + strings.Join(urls, ","), nil
}

// getEtcdCtlCommand returns a 'etcdctl' command
func getEtcdCtlCommand() (string, error) {
	return "ETCDCTL_API=3 etcdctl", nil
}

// runEtcdctlSubcommand runs a etcdctl command
func runEtcdctlSubcommand(o terraform.UIOutput, comm communicator.Communicator, useSudo bool, container string, endpoints []string, subcommand string) ([]string, error) {
	output := []string{}

	// build the
	etcdctlCommand, err := getEtcdCtlCommand()
	if err != nil {
		return nil, err
	}

	argEndpoints, err := getEndpointsArgFor(endpoints)
	if err != nil {
		return nil, err
	}

	fullEtcdctlCommand := fmt.Sprintf("%s %s %s %s", etcdctlCommand, argsCommon, argEndpoints, subcommand)
	dockerCommand := fmt.Sprintf("docker exec -ti '%s' /bin/sh -c '%s'", container, fullEtcdctlCommand)

	var interceptor ssh.OutputFunc = func(s string) {
		output = append(output, s)
	}

	log.Printf("[DEBUG] Running command in etcd container: '%s'", dockerCommand)
	if err := ssh.DoExec(dockerCommand).Apply(interceptor, comm, useSudo); err != nil {
		return nil, err
	}

	return output, nil
}

// getMemberList gets the list of members in the etcd cluster
func getMemberList(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) ([]EtcdMember, error) {
	members := []EtcdMember{}

	container, err := getEtcdContainer(o, comm, useSudo)
	if err != nil {
		return nil, err
	}

	output, err := runEtcdctlSubcommand(o, comm, useSudo, container, []string{localEtcdEndpoint}, subcmdMembersList)
	if err != nil {
		return nil, err
	}

	// parse the output and get the members
	for _, line := range output {
		member, err := getMemberFromString(line)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// doPrintEtcdMembers prints the list of etcd members in the etcd cluster
func doPrintEtcdMembers(d *schema.ResourceData) ssh.ApplyFunc {
	return ssh.ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		members, err := getMemberList(o, comm, useSudo)
		if err != nil {
			if err == ErrNoEtcdContainer {
				o.Output("info: etcd is not running in this node")
			} else {
				return nil
			}
		}
		if len(members) == 0 {
			o.Output("WARNING, would not get list of etcd members")
			return nil
		}
		for _, member := range members {
			o.Output(fmt.Sprintf("info: etcd member '%s' at '%s'", member.ID, member.ClienAddrs))
		}
		return nil
	})
}

// doRemoveIfMember removes this node from the etcd cluster iff it was a member
func doRemoveIfMember(d *schema.ResourceData) ssh.ApplyFunc {
	return ssh.ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		// TODO: check if there is a etcd container running in this machine
		// TODO: get the list of endpoints
		// TODO: get the endpoint that has 127.0.0.1 in the endpoint URL
		// TODO: get the ID for that endpoint
		// TODO: run the "member remove", but using all the `--endpoints` previously obtained
		return nil
	})
}
