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

package ssh

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	defMaxPathLength = 4096

	defTemporaryFilenamePrefix = "tmpfile"

	defTemporaryFilenameExt = "tmp"

	defaultRemoteTmp = "/tmp"

	markStart = "-- START --"

	markEnd = "-- END --"
)

// LocalFileExists reports whether the named file or directory exists.
func LocalFileExists(name string) bool {
	if len(name) > defMaxPathLength {
		return false
	}
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

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

// GetTempFilename returns a temporary filename
func GetTempFilename() (string, error) {
	return randomPath(defTemporaryFilenamePrefix, defTemporaryFilenameExt)
}

// IsTempFilename returns true if it is a temporary filename
func IsTempFilename(filename string) bool {
	base := path.Base(filename)
	if !strings.HasPrefix(base, defTemporaryFilenamePrefix) {
		return false
	}
	if !strings.HasSuffix(base, defTemporaryFilenameExt) {
		return false
	}
	return true
}

// doRealUploadFile uploads a file to a remote path
func doRealUploadFile(contents io.Reader, remote string) Applyer {
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
		return comm.Upload(remote, c)
	})
}

// DoUploadReaderToFile uploads a file to a remote path, using a temporary file in /tmp
// and then moving it to the final destination with `sudo`.
// It is important to use a temporary file as uploads are performed as a regular
// user, while the `mv` is done with `sudo`
func DoUploadReaderToFile(contents io.Reader, remote string) Applyer {
	if len(remote) == 0 {
		panic("empty remote path")
	}

	// do not create temporary files for files that are already temporary
	if IsTempFilename(remote) {
		return DoComposed(
			DoMessageInfo(fmt.Sprintf("Uploading to %q", remote)),
			DoMkdir(filepath.Dir(remote)),
			doRealUploadFile(contents, remote))
	}

	tmpPath, err := GetTempFilename()
	if err != nil {
		panic(err)
	}

	// for regular files, upload to a temp file and then move the temp file to the final destination
	// (uploading directly to destination could need root permissions, while we can "mv" with "sudo")
	return DoWithCleanup(
		DoComposed(
			DoMessageInfo(fmt.Sprintf("Uploading to %q", remote)),
			DoMessageDebug(fmt.Sprintf("Uploading to temporary file %q", tmpPath)),
			doRealUploadFile(contents, tmpPath),
			DoMkdir(filepath.Dir(remote)),
			DoMessageDebug(fmt.Sprintf("... and moving to final destination %s", remote)),
			DoMoveFile(tmpPath, remote)),
		DoDeleteFile(tmpPath),
	)
}

func DoUploadFileToFile(local string, remote string) Applyer {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		// note: we muyst do the "Open" inside the ApplyFunc, as we must delay the operation
		// just in case the file does not exists yet
		f, err := os.Open(local)
		if err != nil {
			return ApplyError(fmt.Sprintf("could not open local file %q for uploading to %q: %s", local, remote, err))
		}

		return DoUploadReaderToFile(f, remote).Apply(o, comm, useSudo)
	})
}

// DoDownloadFileToWriter downloads a file to a writer
func DoDownloadFileToWriter(remote string, contents io.WriteCloser) Applyer {
	return DoComposed(
		DoMessageDebug(fmt.Sprintf("Dumping remote file %s", remote)),
		ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
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
			command := fmt.Sprintf("sh -c \"echo '%s' && cat '%s' && echo '%s'\"",
				markStart, remote, markEnd)
			err := DoExec(command).Apply(interceptor, comm, useSudo)
			if err != nil {
				return err
			}
			contents.Close()

			return nil
		}))
}

// DoWriteLocalFile writes some string in a local file
func DoWriteLocalFile(filename string, contents string) Applyer {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		localFile, err := os.Create(filename)
		if err != nil {
			return err
		}

		if _, err := localFile.WriteString(contents); err != nil {
			return err
		}

		return nil
	})
}

// DoDeleteFile removes a file
func DoDeleteFile(path string) Applyer {
	return DoExec(fmt.Sprintf("rm -f %s", path))
}

// DoDeleteLocalFile removes a local file
func DoDeleteLocalFile(path string) Applyer {
	return DoLocalExec("rm", "-f", path)
}

// DoMoveFile moves a file
func DoMoveFile(src, dst string) Applyer {
	return DoExec(fmt.Sprintf("mv -f %s %s", src, dst))
}

// DoMoveLocalFile moves a local file
func DoMoveLocalFile(src, dst string) Applyer {
	return DoLocalExec("mv", "-f", src, dst)
}

// DoDownloadFile downloads a remote file to a local file
func DoDownloadFile(remote, local string) Applyer {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		localFile, err := os.Create(local)
		if err != nil {
			return err
		}

		return DoComposed(
			DoMessageInfo(fmt.Sprintf("Downloading remote file %s -> %s", remote, local)),
			DoDownloadFileToWriter(remote, localFile)).Apply(o, comm, useSudo)
	})
}

// CheckFileExists checks that a remote file exists
func CheckFileExists(path string) CheckerFunc {
	return CheckExec(fmt.Sprintf("[ -f '%s' ]", path))
}

// CheckFileAbsent checks that a remote file does not exists
func CheckFileAbsent(path string) CheckerFunc {
	return CheckNot(CheckFileExists(path))
}

func CheckLocalFileExists(path string) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		if _, err := os.Stat(path); err == nil {
			return true, nil
		}
		return false, nil
	})
}
