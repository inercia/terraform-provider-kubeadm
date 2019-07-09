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
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

const (
	DefAdminKubeconfig = "/etc/kubernetes/admin.conf"
)

const (
	kubectlNodesIPsCmd = `get nodes --output=jsonpath='{range .items[*]}{.metadata.name} {.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}'`
)

type Manifest struct {
	Path   string
	URL    string
	Inline string
}

func NewManifest(m string) Manifest {
	if isValidUrl(m) {
		return Manifest{URL: m}
	}
	if LocalFileExists(m) {
		return Manifest{Path: m}
	}
	return Manifest{Inline: m}
}

func (m Manifest) IsEmpty() bool {
	if m.Inline == "" && m.Path == "" && m.URL == "" {
		return true
	}
	return false
}

// isValidUrl tests a string to determine if it is a url or not.
func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	} else {
		return true
	}
}

/////////////////////////////////////////////////////////////////////////////////

// DoGetNodesAndIPs gets a map with (IPs, NAME), with the IPs and the nodename in that IP
// this is done by running some magic kubectl command
func DoGetNodesAndIPs(kubectl string, kubeconfig string, ipAddresses map[string]string) Action {
	return DoSendingOutputToFun(
		DoRemoteKubectl(kubectl, kubeconfig, kubectlNodesIPsCmd), func(s string) {
			// parse "<NAME> <IP>"
			r := strings.Split(s, " ")
			if len(r) == 2 {
				ipAddresses[strings.TrimSpace(r[1])] = strings.TrimSpace(r[0])
			} else {
				// TODO: print some log message
			}
		})
}

// DoLocalKubectl runs a local kubectl command
func DoLocalKubectl(kubeconfig string, args ...string) Action {
	// we can look for the "kubectl" binary in advance...
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return ActionError("kubectl not available")
	}

	f := append([]string{fmt.Sprintf("--kubeconfig=%s", kubeconfig)}, args...)
	return DoLocalExec(kubectl, f...)
}

// DoRemoteKubectl runs a remote kubectl command in a remote machine
// it takes care about uploading a valid kubeconfig file if not present in the remote machine
func DoRemoteKubectl(kubectl string, kubeconfig string, args ...string) Action {
	return DoIfElse(
		CheckFileExists(DefAdminKubeconfig),
		DoExec(fmt.Sprintf("kubectl --kubeconfig=%s %s", DefAdminKubeconfig, strings.Join(args, " "))),
		DoLazy(func() Action {
			if kubeconfig == "" {
				return ActionError("no kubeconfig provided, and no remote admin.conf found")
			}

			// upload the local kubeconfig to some temporary remote file
			remoteKubeconfig, err := GetTempFilename()
			if err != nil {
				return ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
			}

			return DoWithCleanup(
				ActionList{
					DoUploadFileToFile(kubeconfig, remoteKubeconfig),
					DoExec(fmt.Sprintf("%s --kubeconfig=%s %s", kubectl, remoteKubeconfig, strings.Join(args, " "))),
				},
				DoTry(DoDeleteFile(remoteKubeconfig)))
		}))
}

// DoLocalKubectlApply applies some manifests with a local kubectl
func DoLocalKubectlApply(kubeconfig string, manifests []Manifest) Action {
	actions := ActionList{}
	for _, manifest := range manifests {
		switch {
		case manifest.Inline != "":
			actions = append(actions, DoLazy(func() Action {
				localManifest, err := GetTempFilename()
				if err != nil {
					return ActionError(fmt.Sprintf("Could not create temporary filename: %s", err))
				}
				return DoWithCleanup(
					ActionList{
						DoWriteLocalFile(localManifest, manifest.Inline),
						DoLocalKubectl(kubeconfig, "apply", "-f", localManifest),
					},
					DoTry(DoDeleteLocalFile(localManifest)))
			}))
		case manifest.Path != "":
			actions = append(actions, DoLocalKubectl(kubeconfig, "apply", "-f", manifest.Path))
		case manifest.URL != "":
			actions = append(actions, DoLocalKubectl(kubeconfig, "apply", "-f", manifest.URL))
		}
	}
	return actions
}

// DoRemoteKubectlApply applies some manifests with a remote kubectl
// manifests can be 1) a local file 2) a URL 3) in a string
func DoRemoteKubectlApply(kubectl string, kubeconfig string, manifests []Manifest) Action {
	actions := ActionList{}

	for _, manifest := range manifests {
		switch {
		case manifest.Inline != "":
			actions = append(actions, DoLazy(func() Action {
				remoteManifest, err := GetTempFilename()
				if err != nil {
					return ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
				}
				return DoWithCleanup(
					ActionList{
						DoUploadReaderToFile(strings.NewReader(manifest.Inline), remoteManifest),
						DoRemoteKubectl(kubectl, kubeconfig, "apply", "-f", remoteManifest),
					},
					DoTry(DoDeleteFile(remoteManifest)))
			}))

		case manifest.Path != "":
			actions = append(actions, DoLazy(func() Action {
				// it is a file: upload the file to a temporary, remote file and then `kubectl apply -f` it
				remoteManifest, err := GetTempFilename()
				if err != nil {
					return ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
				}
				return DoWithCleanup(
					ActionList{
						DoUploadFileToFile(manifest.Path, remoteManifest),
						DoRemoteKubectl(kubectl, kubeconfig, "apply", "-f", remoteManifest),
					},
					DoTry(DoDeleteFile(remoteManifest)))
			}))

		case manifest.URL != "":
			// it is an URL: just run the `kubectl apply`
			actions = append(actions, DoRemoteKubectl(kubectl, kubeconfig, "apply", "-f", manifest.URL))
		}
	}

	return actions
}
