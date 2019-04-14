package kubeadm

//go:generate ../utils/generate.sh --out-var kubeletSysconfigCode --out-package kubeadm --out-file generated_kubelet_sysconfig.go ./assets/kubelet.sysconfig
//go:generate ../utils/generate.sh --out-var kubeadmDropinCode --out-package kubeadm --out-file generated_kubeadm_dropin.go ./assets/kubeadm-dropin.conf
//go:generate ../utils/generate.sh --out-var kubeletServiceCode --out-package kubeadm --out-file generated_kubelet_service.go ./assets/service.conf

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmapiv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
)

const (
	defaultKubeadmVersion = "v1.14.0"
)

var (
	errNoConfig = errors.New("no config provided")
)

func init() {
	spew.Config.Indent = "\t"
}

type ResourceProvisioner struct {
	initConfig *kubeadmapiv1beta1.InitConfiguration
	joinConfig *kubeadmapiv1beta1.JoinConfiguration

	comm    communicator.Communicator
	useSudo bool
}

// Apply runs the provisioner on a specific resource and returns the new
// resource state along with an error. Instead of a diff, the ResourceConfig
// is provided since provisioners only run after a resource has been
// newly created.
func applyFn(ctx context.Context) error {
	provData := ctx.Value(schema.ProvConnDataKey).(*schema.ResourceData)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	// ensure that this is a linux machine
	if d.ConnInfo()["type"] != "ssh" {
		return fmt.Errorf("Unsupported connection type: %s. This provisioner currently only supports linux", s.Ephemeral.ConnInfo["type"])
	}

	join := provData.Get("join").(string)
	config := provData.Get("config").(string)
	preventSudo := provData.Get("prevent_sudo").(bool)
	useSudo := !preventSudo && s.Ephemeral.ConnInfo["user"] != "root"

	log.Printf("[DEBUG] [KUBEADM] kubeadm provisioner: resource data:\n%s", spew.Sdump(provData))

	// build a communicator for the provisioner to use
	comm, err := getCommunicator(ctx, o, s)
	if err != nil {
		o.Output("Error when creating communicator")
		return err
	}

	if _, ok := provData.GetOk("install"); ok {
		auto := provData.Get("install.0.auto").(bool)
		if auto {
			script := provData.Get("install.0.script").(string)
			if err := DoKubeadmSetup(o, comm, script, useSudo); err != nil {
				return err
			}
		}
	}

	// run kubeadm init/join
	if len(join) == 0 {
		initConfig, configFile, err := loadInitConfigFromString(config)
		if err != nil {
			return err
		}
		return DoKubeadmInit(o, comm, initConfig, configFile, useSudo)
	} else {
		joinConfig, configFile, err := loadJoinConfigFromString(config)
		if err != nil {
			return err
		}
		return DoKubeadmJoin(o, comm, joinConfig, configFile, useSudo)
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func loadInitConfigFromString(config string) (*kubeadmapiv1beta1.InitConfiguration, []byte, error) {
	var initConfig *kubeadmapiv1beta1.InitConfiguration
	var clusterConfig *kubeadmapiv1beta1.ClusterConfiguration

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

	return initConfig, configBytes, nil
}

func loadJoinConfigFromString(config string) (*kubeadmapiv1beta1.JoinConfiguration, []byte, error) {
	var joinConfig *kubeadmapiv1beta1.JoinConfiguration

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
		if kubeadmutil.GroupVersionKindsHasJoinConfiguration(k) {
			obj, err := kubeadmutil.UnmarshalFromYamlForCodecs(v, kubeadmapiv1beta1.SchemeGroupVersion, kubeadmscheme.Codecs)
			if err != nil {
				return nil, nil, err
			}

			cfg2, ok := obj.(*kubeadmapiv1beta1.JoinConfiguration)
			if !ok || cfg2 == nil {
				return nil, nil, err
			}

			joinConfig = cfg2
		}
	}

	return joinConfig, configBytes, nil
}

func getCommunicator(ctx context.Context, o terraform.UIOutput, s *terraform.InstanceState) (communicator.Communicator, error) {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}

	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return nil, err
	}

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	return comm, err
}
