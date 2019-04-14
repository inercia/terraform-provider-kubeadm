package kubeadm

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"text/template"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	defaultRemoteTmp = "/tmp"
)

// randomPath gets a random path
func randomPath(prefix, extension string) (string, error) {
	r, err := randBytes(3)
	if err != nil {
		return "", err
	}
	if len(prefix) == 0 || len(extension) == 0 {
		return "", fmt.Errorf("can not use empty prefix or extension")
	}
	return fmt.Sprintf("%s/%s-%s.%s", defaultRemoteTmp, prefix, r, extension), nil
}

// ///////////////////////////////////////////////////////////////////////////////////////////////

type remoteFile struct {
	Output terraform.UIOutput
	Comm   communicator.Communicator
}

func newRemoteFile(o terraform.UIOutput, comm communicator.Communicator) remoteFile {
	return remoteFile{Output: o, Comm: comm}
}

func (rs *remoteFile) Upload(contents io.Reader, prefix, extension string) error {
	p, err := randomPath(prefix, extension)
	if err != nil {
		return err
	}
	return rs.UploadTo(contents, p)
}

func (rs *remoteFile) UploadTo(contents io.Reader, p string) error {
	log.Printf("[DEBUG] Uploading %s", p)
	return rs.Comm.Upload(p, contents)
}

// UploadScript uploads a script to some random path (in /tmp) before executing
func (rs *remoteFile) UploadScript(contents io.Reader, prefix string) error {
	p, err := randomPath(prefix, "sh")
	if err != nil {
		return err
	}

	return rs.Comm.UploadScript(p, contents)
}

// UploadTemplateTo uploads a template (were the `data` contents are replaced)
// to the `dest` destination path.
func (rs *remoteFile) UploadTemplateTo(contents io.Reader, data interface{}, dest string) error {
	log.Printf("[DEBUG] Rendering %s config as a template", dest)
	allContents, err := ioutil.ReadAll(contents)
	if err != nil {
		return err
	}

	replaced := &bytes.Buffer{}
	t, err := template.New(dest).Parse(string(allContents))
	if err != nil {
		log.Println("parsing template:", err)
		return err
	}
	if err := t.Execute(replaced, data); err != nil {
		log.Println("executing template with input data:", err)
		return err
	}

	return rs.UploadTo(replaced, dest)
}

// /////////////////////////////////////////////////////////////////////////////////

type remoteTemplate struct {
	contents io.Reader
	descr    string
	path     string
}

type remoteTemplates struct {
	remoteFile
}

func newRemoteTemplates(o terraform.UIOutput, comm communicator.Communicator) remoteTemplates {
	return remoteTemplates{remoteFile{Output: o, Comm: comm}}
}

func (rt remoteTemplates) Upload(templates []remoteTemplate, data interface{}) error {
	for _, t := range templates {
		log.Printf("[DEBUG] Uploading %s to %s", t.descr, t.path)
		if err := rt.UploadTemplateTo(t.contents, data, t.path); err != nil {
			return err
		}
	}
	return nil
}
