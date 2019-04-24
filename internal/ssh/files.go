package ssh

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	defaultRemoteTmp = "/tmp"

	markStart = "-- START --"

	markEnd = "-- END --"
)

func randBytes(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// randomPath gets a random Path
func randomPath(prefix, extension string) (string, error) {
	r, err := randBytes(3)
	if err != nil {
		return "", err
	}
	if len(prefix) == 0 || len(extension) == 0 {
		return "", fmt.Errorf("can not use empty Prefix or extension")
	}
	return fmt.Sprintf("%s/%s-%s.%s", defaultRemoteTmp, prefix, r, extension), nil
}

// DoUploadFile uploads a file to a remote path
func DoUploadFile(contents io.Reader, remote string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		dir := filepath.Dir(remote)
		log.Printf("[DEBUG] [KUBEADM] Making sure directory '%s' is there", dir)
		err := DoMkdir(dir).Apply(o, comm, useSudo)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] [KUBEADM] removing previous file '%s'", remote)
		cmd := fmt.Sprintf("rm -f %s", remote)
		err = DoExec(cmd).Apply(o, comm, useSudo)
		if err != nil {
			return err
		}

		allContents, err := ioutil.ReadAll(contents)
		if err != nil {
			return err
		}

		// FIXME: for some unknown reason, we must do this conversion...
		// passing rs.Contents to comm.Upload() leads to an empty file
		c := strings.NewReader(string(allContents))

		log.Printf("[DEBUG] [KUBEADM] Uploading to %s:\n%s\n", remote, allContents)
		o.Output(fmt.Sprintf("Uploading file %s (%d bytes)", remote, len(allContents)))
		return comm.Upload(remote, c)
	})
}

// DoDownloadFileToWriter downloads a file to a writer
func DoDownloadFileToWriter(remote string, contents io.WriteCloser) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		enabled := false
		var interceptor OutputFunc = func(s string) {
			if strings.Contains(s, markStart) {
				enabled = true
				return
			}
			if strings.Contains(s, markEnd) {
				enabled = false
				return
			}

			if enabled {
				contents.Write([]byte(s))
				contents.Write([]byte{'\n'})
			} else {
				o.Output(s)
			}
		}

		// Terraform does not provide a mechanism for copying file from a remote host
		// to the local machine
		// so we must run a remote command that dumps the file Contents to stdout
		// hopefully it will be terminal-friendly
		// otherwise, we could use `cat <FILE> | base64 -`
		o.Output(fmt.Sprintf("Dumping remote file %s", remote))
		command := fmt.Sprintf("echo '%s' && cat '%s' && echo '%s'",
			markStart, remote, markEnd)
		err := DoExec(command).Apply(interceptor, comm, useSudo)
		if err != nil {
			return err
		}
		contents.Close()

		return nil
	})
}

// DoDownloadFile downloads a remote file to a local file
func DoDownloadFile(remote, local string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		localFile, err := os.Create(local)
		if err != nil {
			return err
		}

		o.Output(fmt.Sprintf("Downloading remote file %s -> %s", remote, local))
		return DoDownloadFileToWriter(remote, localFile).Apply(o, comm, useSudo)
	})
}

// CheckFileExists checks that a remote file exists
func CheckFileExists(path string) CheckerFunc {
	return CheckCondition(fmt.Sprintf("[ -f '%s' ]", path))
}
