package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/hashicorp/terraform/helper/schema"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"
)

var (
	ErrCreatingCerts = errors.New("error creating PKI assets")
)

type CertsConfig struct {
	Secret   string `json:"certs_secret"` // secret for encrypting certificates (NOTE: currently not used)
	Dir      string `json:"certs_dir"`
	CaCrt    string `json:"certs_ca_crt"`
	CaKey    string `json:"certs_ca_key"`
	SaCrt    string `json:"certs_sa_crt"`
	SaKey    string `json:"certs_sa_key"`
	EtcdCrt  string `json:"certs_etcd_crt"`
	EtcdKey  string `json:"certs_etcd_key"`
	ProxyCrt string `json:"certs_proxy_crt"`
	ProxyKey string `json:"certs_proxy_key"`
}

// List of certificates to distribute to other control plane machines, and a placeholder to the certificates
// See https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/#manual-certs
func (c *CertsConfig) DistributionMap() map[string](*string) {
	return map[string](*string){
		kubeadmconstants.CACertName:                   &c.CaCrt,
		kubeadmconstants.CAKeyName:                    &c.CaKey,
		kubeadmconstants.ServiceAccountPublicKeyName:  &c.SaCrt,
		kubeadmconstants.ServiceAccountPrivateKeyName: &c.SaKey,
		kubeadmconstants.EtcdCACertName:               &c.EtcdCrt,
		kubeadmconstants.EtcdCAKeyName:                &c.EtcdKey,
		kubeadmconstants.FrontProxyCACertName:         &c.ProxyCrt,
		kubeadmconstants.FrontProxyCAKeyName:          &c.ProxyKey,
	}
}

// ToMap converts the certs info to a map
func (c *CertsConfig) ToMap() (map[string]string, error) {
	inrec, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	var inInterface map[string]string
	if err := json.Unmarshal(inrec, &inInterface); err != nil {
		return nil, err
	}
	return inInterface, nil
}

// IsFilled returns true if the certs are there
func (c *CertsConfig) IsFilled() bool {
	for _, cert := range c.DistributionMap() {
		if len(*cert) == 0 {
			return false
		}
	}
	return true
}

// FromMap loads the certificates config info from a map
func (c *CertsConfig) FromMap(m map[string]interface{}) error {
	inrec, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(inrec, c); err != nil {
		return err
	}
	return nil
}

// FromResourceData loads the certificates config info from the ResourceData provided
func (c *CertsConfig) FromResourceData(d *schema.ResourceData) error {
	certsMap := d.Get("config").(map[string]interface{})
	if err := c.FromMap(certsMap); err != nil {
		return err
	}
	return nil
}

func (c *CertsConfig) ToDisk(certsDir string) error {
	writeCertOrKey := func(baseName string, certOrKeyData []byte) error {
		certOrKeyPath := path.Join(certsDir, baseName)
		if _, err := keyutil.ParsePublicKeysPEM(certOrKeyData); err == nil {
			return keyutil.WriteKey(certOrKeyPath, certOrKeyData)
		} else if _, err := certutil.ParseCertsPEM(certOrKeyData); err == nil {
			return certutil.WriteCert(certOrKeyPath, certOrKeyData)
		}
		return fmt.Errorf("unknown certificate data found in '%+v...'", string(certOrKeyData[:25]))
	}

	for baseName, cert := range c.DistributionMap() {
		if err := writeCertOrKey(baseName, []byte(*cert)); err != nil {
			log.Printf("[DEBUG] [KUBEADM] could not write certificate %q: %s", baseName, err)
			return err
		}
		log.Printf("[DEBUG] [KUBEADM] saved certificate %q", baseName)
	}
	return nil
}

// FromDisk fills all the CertsConfig certificates from a directory
func (c *CertsConfig) FromDisk(certsDir string) error {
	// fill the c with the certificates contents
	for baseName, addr := range c.DistributionMap() {
		fullPath := path.Join(certsDir, baseName)
		log.Printf("[DEBUG] [KUBEADM] loading the certificate %q", fullPath)
		cert, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] [KUBEADM] ... %d bytes loaded", len(cert))
		*addr = string(cert)
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// CreateCerts creates the certificates in some temporary directory,
// so they can be uploaded to the remote init machine
func CreateCerts(d *schema.ResourceData, initCfg *kubeadmapi.InitConfiguration) (map[string]string, error) {
	// create a temporary directory for the certificates
	certsDir, err := GetSafeTempDirectory()
	if err != nil {
		return nil, err
	}
	defer func() {
		log.Printf("[DEBUG] [KUBEADM] removing the temporary directory for certificates")
		os.RemoveAll(certsDir)
	}()

	// set the cfg.CertificatesDir as this temp dir
	cfgCopy := initCfg.DeepCopy()
	cfgCopy.CertificatesDir = certsDir

	// Some debugging code:
	//
	//cfgBytes, err := InitConfigToYAML(cfgCopy)
	//if err != nil {
	//	return nil, err
	//}
	//
	//log.Printf("[DEBUG] [KUBEADM] configuration for certificates:")
	//log.Printf("[DEBUG] [KUBEADM] ------------------------")
	//log.Printf("[DEBUG] [KUBEADM] \n%s", string(cfgBytes))
	//log.Printf("[DEBUG] [KUBEADM] ------------------------")
	//log.Printf("[DEBUG] [KUBEADM] creating certificates in %q...", certsDir)

	certList := certs.Certificates{
		&certs.KubeadmCertRootCA,
		&certs.KubeadmCertFrontProxyCA,
		&certs.KubeadmCertEtcdCA,
		// the service account certs are handled in a different place
	}

	certTree, err := certList.AsMap().CertTree()
	if err != nil {
		log.Printf("[DEBUG] [KUBEADM] certificates generation failed: %s", err)
		return nil, err
	}

	if err := certTree.CreateTree(cfgCopy); err != nil {
		log.Printf("[DEBUG] [KUBEADM] certificates generation failed: %s", err)
		return nil, err
	}

	// Service accounts are not x509 certs, so handled separately
	if err := certs.CreateServiceAccountKeyAndPublicKeyFiles(cfgCopy.CertificatesDir); err != nil {
		log.Printf("[DEBUG] [KUBEADM] service account key generation failed: %s", err)
		return nil, err
	}

	// create and set the key we will use for encrypting the certs
	key, err := copycerts.CreateCertificateKey()
	if err != nil {
		log.Printf("[DEBUG] [KUBEADM] certificates key generation failed: %s", err)
		return nil, err
	}

	// load the certs from disk and save (some of them) to the schema, so the provisioner can use them
	certsConfig := CertsConfig{
		Secret: key,
		Dir:    certsDir,
	}

	if err := certsConfig.FromDisk(certsDir); err != nil {
		log.Printf("[DEBUG] [KUBEADM] certificates load from %q failed: %s", certsDir, err)
		return nil, err
	}

	// restore the certificates directory before creating the map
	certsConfig.Dir = initCfg.CertificatesDir

	// ... and create the map with the config for the provisioner
	m, err := certsConfig.ToMap()
	if err != nil {
		return nil, err
	}

	//log.Printf("[DEBUG] [KUBEADM] certificates:")
	//log.Printf("[DEBUG] [KUBEADM] ------------------------")
	//log.Printf("[DEBUG] [KUBEADM] \n%s", spew.Sdump(m))
	//log.Printf("[DEBUG] [KUBEADM] ------------------------")

	return m, nil
}
