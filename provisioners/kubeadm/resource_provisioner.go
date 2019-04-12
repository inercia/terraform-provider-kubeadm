package kubeadm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"

	"github.com/inercia/terraform-kubeadm/internal/kubernetes"
)

const (
	defaultKubeadmVersion = "v1.5"
	defaultKubeadmSetup   = "kubeadm-setup"
	defaultEtcKubelet     = "/etc/kubernetes/kubelet"
)

var (
	errNoConfig = errors.New("no config provided")
)

func init() {
	spew.Config.Indent = "\t"
}

func Provisioner() terraform.ResourceProvisioner {
	return &ResourceProvisioner{}
}

type ResourceProvisioner struct {
	masterConfig *kubernetes.MasterConfiguration
	nodeConfig   *kubernetes.NodeConfiguration

	Master       string `mapstructure:"master"` // something like <master-ip>:<master-port>
	Config       string `mapstructure:"config"`
	SetupScript  string `mapstructure:"setup_script"`
	SetupVersion string `mapstructure:"setup_version"`

	PreventSudo bool `mapstructure:"prevent_sudo"`
}

// decodes configuration from terraform and builds out a provisioner
func (p *ResourceProvisioner) loadFromResourceConfig(c *terraform.ResourceConfig) error {
	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		ZeroFields:       true,
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

	// put the kubeadm config in the right place: master or node
	if len(p.Config) > 0 {
		if len(p.Master) == 0 {
			log.Printf("[DEBUG] parsing master configuration from JSON")
			p.masterConfig = &kubernetes.MasterConfiguration{}
			p.nodeConfig = nil
			err := json.Unmarshal([]byte(p.Config), p.masterConfig)
			if err != nil {
				log.Printf("[ERROR] could not parse master configuration from '%s'", p.Config)
				return err
			}
		} else {
			log.Printf("[DEBUG] parsing node configuration from JSON")
			p.masterConfig = nil
			p.nodeConfig = &kubernetes.NodeConfiguration{}
			err := json.Unmarshal([]byte(p.Config), p.nodeConfig)
			if err != nil {
				log.Printf("[ERROR] could not parse node configuration from '%s'", p.Config)
				return err
			}
		}
	}

	// fix some other default values
	if len(p.SetupVersion) == 0 {
		p.SetupVersion = defaultKubeadmVersion
	}

	return nil
}

// Apply runs the provisioner on a specific resource and returns the new
// resource state along with an error. Instead of a diff, the ResourceConfig
// is provided since provisioners only run after a resource has been
// newly created.
func (r ResourceProvisioner) Apply(o terraform.UIOutput, s *terraform.InstanceState, c *terraform.ResourceConfig) error {
	if err := r.loadFromResourceConfig(c); err != nil {
		o.Output("Error when getting config in provisioner")
		return err
	}

	log.Printf("[DEBUG] kubeadm resource provisioner:\n%s", spew.Sdump(r))

	// check we have everything we need
	if len(r.Config) == 0 {
		return errNoConfig
	}

	// ensure that this is a linux machine
	if s.Ephemeral.ConnInfo["type"] != "ssh" {
		return fmt.Errorf("Unsupported connection type: %s. This provisioner currently only supports linux", s.Ephemeral.ConnInfo["type"])
	}

	useSudo := !r.PreventSudo && s.Ephemeral.ConnInfo["user"] != "root"

	// build a communicator for the provisioner to use
	comm, err := communicator.New(s)
	if err != nil {
		o.Output("Error when creating communicator")
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

	// setup kubeadm
	remoteSetupScript := newRemoteScript(o, comm)
	if len(r.SetupScript) > 0 {
		o.Output(fmt.Sprintf("Uploading custom kubeadm script from %s...", r.SetupScript))
		f, err := os.Open(r.SetupScript)
		if err != nil {
			return err
		}
		if err := remoteSetupScript.UploadScript(f, defaultKubeadmSetup); err != nil {
			return err
		}
	} else {
		o.Output("Uploading default kubeadm setup script...")
		if err := remoteSetupScript.UploadScript(strings.NewReader(setupScriptCode), defaultKubeadmSetup); err != nil {
			return err
		}
	}
	o.Output("Running setup script")
	if err := remoteSetupScript.Run("", useSudo); err != nil {
		return err
	}

	// run kukbeadm init/join
	if len(r.Master) == 0 {
		// TODO: kubeadm v1.5.5 does not work fine with config files
		//r.uploadConfig(r.masterConfig, "master", o, comm)

		o.Output("Uploading kubelet config")
		cfg := newRemoteFile(o, comm)
		if err := cfg.UploadTo(bytes.NewBufferString(kubeletMasterCode), defaultEtcKubelet); err != nil {
			return err
		}

		// build a command to run "kubeadm init" in the master
		o.Output(fmt.Sprintf("Initializing kubadm [token=%s]", r.masterConfig.Token))
		initCommand := fmt.Sprintf("kubeadm init --skip-preflight-checks --token=%s", r.masterConfig.Token)
		if len(r.masterConfig.Networking.PodSubnet) > 0 {
			initCommand += fmt.Sprintf(" --pod-network-cidr %s", r.masterConfig.Networking.PodSubnet)
		}
		if len(r.masterConfig.Networking.ServiceSubnet) > 0 {
			initCommand += fmt.Sprintf(" --service-cidr %s", r.masterConfig.Networking.ServiceSubnet)
		}
		if len(r.masterConfig.Networking.DNSDomain) > 0 {
			initCommand += fmt.Sprintf(" --service-dns-domain %s", r.masterConfig.Networking.DNSDomain)
		}
		if r.masterConfig.API.BindPort != 0 {
			initCommand += fmt.Sprintf(" --api-port %d", r.masterConfig.API.BindPort)
		}
		commands := []string{
			"kubeadm reset || /bin/true",
			"systemctl restart kubelet || /bin/true",
			initCommand,
		}
		if err := runCommands(o, comm, useSudo, commands); err != nil {
			return err
		}
	} else {
		// TODO: kubeadm v1.5.5 does not work fine with config files
		//r.uploadConfig(r.nodeConfig, "node", o, comm)

		o.Output("Uploading kubelet config")
		cfg := newRemoteFile(o, comm)
		if err := cfg.UploadTo(bytes.NewBufferString(kubeletNodeCode), defaultEtcKubelet); err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Joining cluster at %s [token=%s]", r.Master, r.nodeConfig.Token))
		commands := []string{
			"kubeadm reset || /bin/true",
			"systemctl stop kubelet || /bin/true",
			fmt.Sprintf("kubeadm join --skip-preflight-checks --token=%s %s", r.nodeConfig.Token, r.Master),
			"systemctl start kubelet",
		}
		if err := runCommands(o, comm, useSudo, commands); err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceProvisioner) uploadConfig(ci interface{}, name string, o terraform.UIOutput, comm communicator.Communicator) error {
	log.Printf("[DEBUG] Rendering %s config as YAML", name)
	cfgContents, err := yaml.Marshal(ci)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] YAML file:\n%s\n", cfgContents)

	o.Output(fmt.Sprintf("Uploading kubeadm configuration for %s...", name))
	cfg := newRemoteFile(o, comm)
	if err := cfg.Upload(bytes.NewBuffer(cfgContents), fmt.Sprintf("kubeadm-%s-cfg", name), "yaml"); err != nil {
		return err
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

func (r *ResourceProvisioner) Stop() error {
	// TODO
	return nil
}

func retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("[DEBUG] retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}

// return an address as host:port (setting a default port p if there was no port specified)
func addressWithPort(name string, p int) string {
	if strings.IndexByte(name, ':') < 0 {
		return net.JoinHostPort(name, fmt.Sprintf("%d", p))
	}
	return name
}
