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

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform/helper/schema"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

var (
	ErrCreatingCerts = errors.New("error creating PKI assets")
)

type CertsConfig struct {
	CaCrt    string `json:"ca_crt"`
	CaKey    string `json:"ca_key"`
	SaCrt    string `json:"sa_crt"`
	SaKey    string `json:"sa_key"`
	EtcdCrt  string `json:"etcd_crt"`
	EtcdKey  string `json:"etcd_key"`
	ProxyCrt string `json:"proxy_crt"`
	ProxyKey string `json:"proxy_key"`
}

// List of certificates to distribute to other control plane machines, and a placeholder to the certificates
// See https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/#manual-certs
func (c *CertsConfig) DistributionMap() map[string]*string {
	return map[string]*string{
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

// HasAllCertificates returns true if ALL the certs are there
func (c *CertsConfig) HasAllCertificates() bool {
	for _, cert := range c.DistributionMap() {
		if len(*cert) == 0 {
			return false
		}
	}
	return true
}

// HasSomeCertificates returns true if SOME the certs are there
func (c *CertsConfig) HasSomeCertificates() bool {
	for _, cert := range c.DistributionMap() {
		if len(*cert) > 0 {
			return true
		}
	}
	return false
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

// FromResourceDataConfig loads the certificates config info
// from the "config" map in the ResourceData provided
func (c *CertsConfig) FromResourceDataConfig(d *schema.ResourceData) error {
	certsMap := d.Get("config").(map[string]interface{})
	if err := c.FromMap(certsMap); err != nil {
		return err
	}
	return nil
}

// FromResourceDataConfig loads the certificates config info from
// the "config" map in the ResourceData provided
func (c *CertsConfig) FromResourceDataCerts(d *schema.ResourceData) error {
	certsMapOpt, ok := d.GetOk("certs.0")
	if !ok {
		return nil
	}
	certsMap := certsMapOpt.(map[string]interface{})
	if err := c.FromMap(certsMap); err != nil {
		return err
	}
	return nil
}

// ToDisk dumps the certificates to disk
func (c *CertsConfig) ToDisk(certsDir string) error {
	writeCertOrKey := func(baseName string, certOrKeyData []byte) error {
		if len(certOrKeyData) == 0 {
			return nil
		}
		certOrKeyPath := path.Join(certsDir, baseName)
		if _, err := keyutil.ParsePublicKeysPEM(certOrKeyData); err == nil {
			return keyutil.WriteKey(certOrKeyPath, certOrKeyData)
		} else if _, err := certutil.ParseCertsPEM(certOrKeyData); err == nil {
			return certutil.WriteCert(certOrKeyPath, certOrKeyData)
		}
		return fmt.Errorf("unknown certificate data found in '%+v...'", string(certOrKeyData[:25]))
	}

	for baseName, cert := range c.DistributionMap() {
		certContents := []byte(*cert)
		if len(certContents) == 0 {
			ssh.Debug("(empty %q: skipping)", baseName)
			continue
		}
		if err := writeCertOrKey(baseName, certContents); err != nil {
			ssh.Debug("could not write certificate %q: %s", baseName, err)
			return err
		}
		ssh.Debug("saved certificate %q", baseName)
	}
	return nil
}

// FromDisk fills all the CertsConfig certificates from a directory
func (c *CertsConfig) FromDisk(certsDir string) error {
	// fill the c with the certificates contents
	for baseName, addr := range c.DistributionMap() {
		fullPath := path.Join(certsDir, baseName)
		ssh.Debug("loading the certificate %q", fullPath)
		cert, err := ioutil.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				ssh.Debug("(%q does not exist: skipping)", fullPath)
				continue
			}
			return err
		}
		ssh.Debug("... %d bytes loaded", len(cert))
		*addr = string(cert)
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// CreateCerts creates the certificates in some temporary directory,
// so they can be uploaded to the remote init machine
func CreateCerts(d *schema.ResourceData, initCfg *kubeadmapi.InitConfiguration) (map[string]string, error) {
	// create a temporary directory for the certificates
	certsDir, err := GetSafeLocalTempDirectory()
	if err != nil {
		return nil, err
	}
	defer func() {
		ssh.Debug("removing the temporary directory for certificates")
		_ = os.RemoveAll(certsDir)
	}()

	// set the cfg.CertificatesDir as this temp dir
	cfgCopy := initCfg.DeepCopy()
	cfgCopy.CertificatesDir = certsDir

	// load any user-provided certificates
	userCertsConfig := CertsConfig{}
	if err := userCertsConfig.FromResourceDataCerts(d); err != nil {
		ssh.Debug("could not load user-provided certificates: %s", err)
		return nil, err
	}
	if userCertsConfig.HasSomeCertificates() {
		ssh.Debug("user has provided some certificates: saving them to %q", certsDir)
		// .. and save them to the disk
		if err := userCertsConfig.ToDisk(certsDir); err != nil {
			ssh.Debug("could not save user-provided certificates to %q: %s", certsDir, err)
			return nil, err
		}
	}

	// Some debugging code:
	//
	//cfgBytes, err := InitConfigToYAML(cfgCopy)
	//if err != nil {
	//	return nil, err
	//}
	//
	//ssh.Debug("configuration for certificates:")
	//ssh.Debug("------------------------")
	//ssh.Debug("\n%s", string(cfgBytes))
	//ssh.Debug("------------------------")
	//ssh.Debug("creating certificates in %q...", certsDir)

	certList := certs.Certificates{
		&certs.KubeadmCertRootCA,
		&certs.KubeadmCertFrontProxyCA,
		&certs.KubeadmCertEtcdCA,
		// the service account certs are handled in a different place
	}

	certTree, err := certList.AsMap().CertTree()
	if err != nil {
		ssh.Debug("certificates generation failed: %s", err)
		return nil, err
	}

	if err := certTree.CreateTree(cfgCopy); err != nil {
		ssh.Debug("certificates generation failed: %s", err)
		return nil, err
	}

	// Service accounts are not x509 certs, so handled separately
	if err := certs.CreateServiceAccountKeyAndPublicKeyFiles(cfgCopy.CertificatesDir); err != nil {
		ssh.Debug("service account key generation failed: %s", err)
		return nil, err
	}

	// load the certs from disk and save (some of them) to the schema, so the provisioner can use them
	certsConfig := CertsConfig{}
	if err := certsConfig.FromDisk(certsDir); err != nil {
		ssh.Debug("certificates load from %q failed: %s", certsDir, err)
		return nil, err
	}

	// ... and create the map with the config for the provisioner
	m, err := certsConfig.ToMap()
	if err != nil {
		return nil, err
	}

	//ssh.Debug("certificates:")
	//ssh.Debug("------------------------")
	//ssh.Debug("\n%s", spew.Sdump(m))
	//ssh.Debug("------------------------")

	return m, nil
}
