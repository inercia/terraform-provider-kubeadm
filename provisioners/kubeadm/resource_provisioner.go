package kubeadm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"

	"github.com/inercia/terraform-kubeadm/internal/kubernetes"
	"strconv"
)

const (
	defaultMasterPort = 6443
)

var (
	errNoConfig         = errors.New("no config provided")
	errPortParsingError = errors.New("could not parse port number")
)

func init() {
	spew.Config.Indent = "\t"
}

func Provisioner() terraform.ResourceProvisioner {
	return &ResourceProvisioner{}
}

type ResourceProvisioner struct {
	useSudo bool

	masterConfig kubernetes.MasterConfiguration
	nodeConfig   kubernetes.NodeConfiguration

	Master  string `mapstructure:"master"` // something like <master-ip>:<master-port>
	Config  string `mapstructure:"config"`
	Kubeadm string `mapstructure:"kubeadm"`
}

// Apply runs the provisioner on a specific resource and returns the new
// resource state along with an error. Instead of a diff, the ResourceConfig
// is provided since provisioners only run after a resource has been
// newly created.
func (r *ResourceProvisioner) Apply(o terraform.UIOutput, s *terraform.InstanceState, c *terraform.ResourceConfig) error {
	if err := configToProvisioner(r, c); err != nil {
		o.Output("Error when getting provisioning config")
		return err
	}

	prettyResource := spew.Sdump(r)
	log.Printf("kubeadm resource provisioner:\n%s", prettyResource)

	// check we have everything we need
	if len(r.Config) == 0 {
		return errNoConfig
	}

	// ensure that this is a linux machine
	if s.Ephemeral.ConnInfo["type"] != "ssh" {
		return fmt.Errorf("Unsupported connection type: %s. This provisioner currently only supports linux", s.Ephemeral.ConnInfo["type"])
	}

	// build a communicator for the provisioner to use
	comm, err := communicator.New(s)
	if err != nil {
		o.Output("Error on communicator.New")
		return err
	}

	err = retryFunc(comm.Timeout(), func() error {
		err := comm.Connect(o)
		return err
	})
	if err != nil {
		return err
	}
	defer comm.Disconnect()

	// we only try to install kubeadm when no `kubeadm` parameter has been provided
	if len(r.Kubeadm) == 0 {
		o.Output("Trying to install kubeadm...")

		// make sure kubeadm is installed in the remote machine, installing it otherwise
		ss := newRemoteScript(o, comm)
		if err := ss.Upload(strings.NewReader(setupScript), "kubeadm", "sh"); err != nil {
			return err
		}
		defer ss.Cleanup()
		if err := ss.Run(r.useSudo); err != nil {
			return err
		}
		r.Kubeadm = defaultKubeadmExe
	}

	// upload the kubeadm config to the remote machine
	cfg := newRemoteFile(o, comm)
	if err := cfg.Upload(strings.NewReader(r.Config), "kubeadm", "cfg"); err != nil {
		return err
	}
	defer cfg.Cleanup()

	if len(r.Master) == 0 {
		// build a command to run "kubeadm init" in the master
		command := fmt.Sprintf("%s init --token=%s --config=%s", r.Kubeadm, r.masterConfig.Token, cfg.Path)
		if err := runCommand(o, comm, r.useSudo, command); err != nil {
			return err
		}
	} else {
		master := r.Master
		port := defaultMasterPort
		host, portStr, err := net.SplitHostPort(master)
		if err != nil {
			return err
		}
		if len(portStr) > 0 {
			port, err = strconv.Atoi(portStr)
			if err != nil {
				return errPortParsingError
			}
		}
		master = net.JoinHostPort(host, strconv.Itoa(port))

		// build a command to run "kubeadm join" in a node
		command := fmt.Sprintf("%s join --token=%s --config=%s %s", r.Kubeadm, r.nodeConfig.Token, cfg.Path, master)
		if err := runCommand(o, comm, r.useSudo, command); err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	// Validate is called once at the beginning with the raw
	// configuration (no interpolation done) and can return a list of warnings
	// and/or errors.
	//
	// This is called once per resource.
	//
	// This should not assume any of the values in the resource configuration
	// are valid since it is possible they have to be interpolated still.
	// The primary use case of this call is to check that the required keys
	// are set and that the general structure is correct.

	return ws, es
}

func retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}

// decodes configuration from terraform and builds out a provisioner
func configToProvisioner(p *ResourceProvisioner, c *terraform.ResourceConfig) error {
	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           p,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	// build a map of all configuration values, by default this is going to
	// pass in all configuration elements for the base configuration as
	// well as extra values. Build a single value and then from there, continue forth!
	m := make(map[string]interface{})
	for k, v := range c.Raw {
		m[k] = v
	}
	for k, v := range c.Config {
		m[k] = v
	}
	if err = decoder.Decode(m); err != nil {
		return err
	}

	if len(p.Config) > 0 {
		if len(p.Master) == 0 {
			log.Printf("[DEBUG] provisioner-kubeadm: parsing master configuration")
			err := json.Unmarshal([]byte(p.Config), &p.masterConfig)
			if err != nil {
				log.Printf("[ERROR] provisioner-kubeadm: could not parse master configuration from '%s'", p.Config)
				return err
			}
		} else {
			log.Printf("[DEBUG] provisioner-kubeadm: parsing node configuration")
			err := json.Unmarshal([]byte(p.Config), &p.nodeConfig)
			if err != nil {
				log.Printf("[ERROR] provisioner-kubeadm: could not parse node configuration from '%s'", p.Config)
				return err
			}
		}
	}

	return nil
}
