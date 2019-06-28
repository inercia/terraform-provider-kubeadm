package provisioner

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	defaultKubeadmSetup = "kubeadm-setup"
)

// doKubeadmSetup tries to install kubeadm in the remote machine
func doKubeadmSetup(d *schema.ResourceData, o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	if _, ok := d.GetOk("install"); ok {
		auto := d.Get("install.0.auto").(bool)
		if auto {
			code := ""

			inline := d.Get("install.0.inline").(string)
			script := d.Get("install.0.script").(string)
			if len(inline) > 0 {
				code = inline
			} else if len(script) > 0 {
				o.Output(fmt.Sprintf("Uploading custom kubeadm installation script from %s...", script))
				contents, err := ioutil.ReadFile(script)
				if err != nil {
					o.Output(fmt.Sprintf("Error when reading installation script %q", script))
					return err
				}
				code = string(contents)
			}

			if len(code) > 0 {
				o.Output("Running kubeadm installation script...")
				return ssh.DoExecScript(strings.NewReader(code), defaultKubeadmSetup).Apply(o, comm, useSudo)
			}
		}
	}
	return nil
}
