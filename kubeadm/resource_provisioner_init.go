package kubeadm

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doKubeadmInit performs a `kubeadm init` in the remote host
func doKubeadmInit(d *schema.ResourceData, configFile []byte) ssh.ApplyFunc {
	configPath := path.Join(defKubeadmInitConfPath)
	extraArgs := ""
	extraArgs += " " + getKubeadmIgnoredChecksArg(d)
	extraArgs += " " + getKubeadmNodenameArg(d)

	return ssh.Composite(
		ssh.ActionIf(
			ssh.CheckFileExists(configPath),
			ssh.DoExec("kubeadm reset --force")),
		ssh.DoUploadFile(bytes.NewReader(configFile), configPath),
		ssh.DoExec(fmt.Sprintf("kubeadm init --config=%s %s", configPath, extraArgs)),
	)
}

// doDownloadKubeconfig downloads a kubeconfig from the remote master
func doDownloadKubeconfig(d *schema.ResourceData, configFile []byte) ssh.ApplyFunc {
	kubeconfigOpt, ok := d.GetOk("kubeconfig")
	if !ok {
		return ssh.EmptyAction()
	}

	return ssh.DoDownloadFile(defAdminKubeconfig, kubeconfigOpt.(string))
}

// doLoadManifests loads some extra manifests
func doLoadManifests(d *schema.ResourceData, configFile []byte) ssh.ApplyFunc {
	manifestsOpt, ok := d.GetOk("manifests")
	if !ok {
		return ssh.EmptyAction()
	}
	manifests := manifestsOpt.([]interface{})

	// FIXME: replace this Exec for `kubectl` by a proper API client in Go...

	return ssh.ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		kubeconfigOpt, ok := d.GetOk("kubeconfig")
		if !ok {
			o.Output(fmt.Sprintf("WARNING: no 'kubeconfig' provided: will not load manifests"))
			return nil
		}
		kubeconfig := kubeconfigOpt.(string)

		o.Output(fmt.Sprintf("Loading manifest with kubectl..."))
		for _, manifest := range manifests {
			cmd := exec.Command("kubectl",
				"--kubeconfig", kubeconfig,
				"apply", "-f", manifest.(string))

			cmdReader, err := cmd.StdoutPipe()
			if err != nil {
				o.Output(fmt.Sprintf("Error creating pipe for kubectl: %s", err))
				return err
			}

			scanner := bufio.NewScanner(cmdReader)
			go func() {
				for scanner.Scan() {
					o.Output(fmt.Sprintf("%s\n", scanner.Text()))
				}
			}()

			err = cmd.Start()
			if err != nil {
				o.Output(fmt.Sprintf("Error starting kubectl locally: %s", err))
				return err
			}

			err = cmd.Wait()
			if err != nil {
				o.Output(fmt.Sprintf("Error waiting for kubectl: %s", err))
				return err
			}
		}

		return nil
	})
}

// unmarshallInitConfig unmarshalls the initConfiguration passed from
// the kubeadm `data` resource
func unmarshallInitConfig(d *schema.ResourceData) (*kubeadmapiv1beta1.InitConfiguration, []byte, error) {
	var initConfig *kubeadmapiv1beta1.InitConfiguration
	var clusterConfig *kubeadmapiv1beta1.ClusterConfiguration

	config := d.Get("config").(string)

	// deserialize the configuration saved in the `config`
	configBytes, err := FromTerraformSafeString(config)
	if err != nil {
		return nil, nil, err
	}

	// load the initConfiguration from the `config` field
	objects, err := kubeadmutil.SplitYAMLDocuments(configBytes)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range objects {
		if kubeadmutil.GroupVersionKindsHasInitConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.InitConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			initConfig = cfg2
		} else if kubeadmutil.GroupVersionKindsHasClusterConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.ClusterConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			clusterConfig = cfg2
		}
	}
	initConfig.ClusterConfiguration = *clusterConfig

	initbytes, err := kubeadmutil.MarshalToYamlForCodecs(initConfig, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}
	allFiles := [][]byte{initbytes}

	clusterbytes, err := kubeadmutil.MarshalToYamlForCodecs(&initConfig.ClusterConfiguration, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
	if err != nil {
		return nil, nil, err
	}
	allFiles = append(allFiles, clusterbytes)

	configBytes = bytes.Join(allFiles, []byte(kubeadmconstants.YAMLDocumentSeparator))

	log.Printf("[DEBUG] [KUBEADM] init config:\n%s\n%", configBytes)

	return initConfig, configBytes, nil
}
