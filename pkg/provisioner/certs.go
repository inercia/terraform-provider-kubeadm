package provisioner

import (
	"log"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// RetrieveAndUploadCerts checks if the certificates in the existing API server are still there.
// if not, retrieves the certificates from the `d.config.certs` and uploads them to the APi server
// with the same key.
// NOTE: this is currently not used
func RetrieveAndUploadCerts(d *schema.ResourceData, cfg *kubeadmapi.InitConfiguration) error {

	kubeconfig := d.Get("config.config_path").(string)
	client, err := common.GetClientSet(kubeconfig)
	if err != nil {
		return err
	}

	certsConfig := &common.CertsConfig{}
	if err := certsConfig.FromResourceData(d); err != nil {
		return err
	}

	// check if the certificates are still in the API server
	// create a temporary directory for the certificates and try to download them
	certsDir, err := common.GetSafeTempDirectory()
	if err != nil {
		return err
	}
	defer func() {
		log.Printf("[DEBUG] [KUBEADM] removing the temporary directory for certificates")
		_ = os.RemoveAll(certsDir)
	}()

	// set the cfg.CertificatesDir as this temp dir
	cfgCopy := cfg.DeepCopy()
	cfgCopy.CertificatesDir = certsDir

	log.Printf("[DEBUG] [KUBEADM] trying to download certificates to %q", certsDir)
	err = copycerts.DownloadCerts(client, cfg, certsConfig.Secret)
	if err == nil {
		// certificates are still in the API server: we do not need to do anything else...
		// TODO: maybe we should do some other checks on the certificates...
		log.Printf("[DEBUG] [KUBEADM] certificates downloaded from API server")
		return nil
	}

	log.Printf("[DEBUG] [KUBEADM] saving certificates from 'data.config.certs' to %q...", certsDir)
	if err := certsConfig.ToDisk(certsDir); err != nil {
		return err
	}

	// upload the shared certificates: ca.key, ca.crt, sa.key, sa.pub + etcd/ca.key, etcd/ca.crt if local/stacked etcd
	log.Printf("[DEBUG] [KUBEADM] uploading certificates from %q", certsDir)
	err = copycerts.UploadCerts(client, cfg, certsConfig.Secret)
	if err != nil {
		log.Printf("[DEBUG] [KUBEADM] certificates upload failed: %s", err)
		return err
	}

	return nil
}
