package provisioner

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doLoadHelm loads Helm (if enabled)
func doLoadHelm(d *schema.ResourceData) ssh.ApplyFunc {
	opt, ok := d.GetOk("config.helm_enabled")
	if !ok {
		return ssh.DoMessage("Helm will not be loaded")
	}
	enabled, err := strconv.ParseBool(opt.(string))
	if err != nil {
		panic("couold not parse helm_enabled in provisioner")
	}
	if !enabled {
		return ssh.DoMessage("Helm will not be loaded")
	}
	if common.DefHelmManifest == "" {
		return ssh.DoMessage("no manifest for Helm: Helm will not be loaded")
	}
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("cannot not load Helm: no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, []string{common.DefHelmManifest})
}

// doLoadDashboard loads the dashboard (if enabled)
func doLoadDashboard(d *schema.ResourceData) ssh.ApplyFunc {
	opt, ok := d.GetOk("config.dashboard_enabled")
	if !ok {
		return ssh.DoMessage("the Dashboard will not be loaded")
	}
	enabled, err := strconv.ParseBool(opt.(string))
	if err != nil {
		panic("could not parse dashboard_enabled in provisioner")
	}
	if !enabled {
		return ssh.DoMessage("the Dashboard will not be loaded")
	}
	if common.DefDashboardManifest == "" {
		return ssh.DoMessage("no manifest for Dashboard: the Dashboard will not be loaded")
	}
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("cannot not load the Dashboard: no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, []string{common.DefDashboardManifest})
}

// doLoadManifests loads some extra manifests
func doLoadManifests(d *schema.ResourceData) ssh.ApplyFunc {
	manifestsOpt, ok := d.GetOk("manifests")
	if !ok {
		return ssh.DoNothing()
	}
	manifests := []string{}
	for _, v := range manifestsOpt.([]interface{}) {
		manifests = append(manifests, v.(string))
	}

	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoMessage("ERROR: will not load manifests as no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, manifests)
}
