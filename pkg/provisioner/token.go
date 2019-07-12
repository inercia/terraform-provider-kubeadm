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

package provisioner

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	// TTL for tokens created for a new join, when no previous token is available
	newJoinTokenTTL = "1h"
)

var (
	errKubeadmParse = errors.New("error parsing kubeadm output")
)

type KubeadmToken struct {
	Token       string
	TTL         string
	Expires     time.Time
	Usages      string
	Description string
	Extra       string
}

func (token KubeadmToken) IsExpired(now time.Time) bool {
	return now.After(token.Expires)
}

type KubeadmTokensSet map[string]KubeadmToken

func (kt KubeadmTokensSet) FromString(s string) error {
	// Parse something like:
	//
	// TOKEN                     TTL       EXPIRES                USAGES                   DESCRIPTION   EXTRA GROUPS
	// 5befc5.a36864a4c9cc2c7d   22h       2019-07-10T15:08:31Z   authentication,signing   <none>        system:bootstrappers:kubeadm:default-node-token
	//
	lines := strings.Split(s, "\n")
	// first line is the header: do not consider it
	for _, line := range lines {
		lineCleaned := strings.TrimSpace(line)
		if lineCleaned == "" {
			ssh.Debug("empty token line: skipping")
			continue
		}

		components := strings.Fields(lineCleaned)
		if len(components) != 6 {
			ssh.Debug("does not look like a token line (len=%d): %q", len(components), lineCleaned)
			continue
		}
		ssh.Debug("token info components: %s", components)

		maybeToken := strings.TrimSpace(components[0])
		matched, err := regexp.MatchString(common.TokenRegex, maybeToken)
		if err != nil {
			ssh.Debug("match of %q failed: %s", maybeToken, err)
			return err
		}
		if !matched {
			ssh.Debug("%q does not match %q: ignored", maybeToken, common.TokenRegex)
			continue
		}

		// parse the expiration time
		str := strings.TrimSpace(components[2])
		expiration, err := time.Parse(time.RFC3339, str)

		kt[maybeToken] = KubeadmToken{
			Token:       maybeToken,
			TTL:         strings.TrimSpace(components[1]),
			Expires:     expiration,
			Usages:      strings.TrimSpace(components[3]),
			Description: strings.TrimSpace(components[4]),
			Extra:       strings.TrimSpace(components[5]),
		}
	}
	return nil
}

// DoExecKubeadmToken runs a "kubeadm token" command, with a auto-uploaded kubeconfig file
func DoExecKubeadmToken(d *schema.ResourceData, cmd string) ssh.Action {
	// upload the local kubeconfig to some temporary remote file
	remoteKubeconfig, err := ssh.GetTempFilename()
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
	}

	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError(fmt.Sprintf("Could not get the local kubeconfig: %s", err))
	}

	kubeadm := getKubeadmFromResourceData(d)

	return ssh.DoWithCleanup(
		ssh.ActionList{
			ssh.DoTry(ssh.DoDeleteFile(remoteKubeconfig)),
		},
		ssh.ActionList{
			ssh.DoUploadFileToFile(kubeconfig, remoteKubeconfig),
			ssh.DoExec(fmt.Sprintf("%s token --kubeconfig=%s %s", kubeadm, remoteKubeconfig, cmd)),
		})
}

// DoGetCurrentRemoteTokens get the list of remote tokens stored in the API server
func DoGetCurrentRemoteTokens(d *schema.ResourceData, kts KubeadmTokensSet) ssh.Action {
	var buf bytes.Buffer

	// run "kubeadm token list" in the remote host, uploading the kubeconfig before
	return ssh.ActionList{
		ssh.DoSendingExecOutputToWriter(&buf, DoExecKubeadmToken(d, "list")),
		ssh.ActionFunc(func(cfg ssh.Config) ssh.Action {
			ssh.Debug("parsing kubeadm output")
			ssh.Debug("%s", buf.String())
			if err := kts.FromString(buf.String()); err != nil {
				ssh.Debug("error when parsing 'kubeadm token' output: %s", err)
				return ssh.ActionError(fmt.Sprintf("Could not parse kubeadm output: %s", err))
			}
			return nil
		}),
	}
}

// SetNewToken sets a new token in the configuration in the ResourceData
func DoSetNewToken(d *schema.ResourceData, newToken string) ssh.Action {
	return ssh.ActionFunc(func(cfg ssh.Config) ssh.Action {
		// update the token in "config.join"
		ssh.Debug("getting current join configuration")
		joinConfig, _, err := common.JoinConfigFromResourceData(d)
		if err != nil {
			return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
		}
		joinConfig.Discovery.BootstrapToken = &kubeadmapi.BootstrapTokenDiscovery{
			Token:                    newToken,
			UnsafeSkipCAVerification: true,
		}
		joinConfig.Discovery.TLSBootstrapToken = newToken

		if err := common.JoinConfigToResourceData(d, joinConfig); err != nil {
			return ssh.ActionError(err.Error())
		}

		return nil
	})
}

// checkTokenIsValid checks that the current token is still valid
func checkTokenIsValid(d *schema.ResourceData, tokens KubeadmTokensSet) ssh.CheckerFunc {
	currentToken := getTokenFromResourceData(d)

	return ssh.CheckerFunc(func(cfg ssh.Config) (bool, error) {
		action := DoGetCurrentRemoteTokens(d, tokens)
		if err := action.Apply(cfg); ssh.IsError(err) {
			return false, fmt.Errorf("cannot check token is valid: %s", err)
		}

		ssh.Debug("%d tokens obtained", len(tokens))
		for _, token := range tokens {
			if token.Token == currentToken {
				ssh.Debug("current token, %q, found in the list of tokens", currentToken)

				if token.IsExpired(time.Now()) {
					ssh.Debug("token %q seems to be expired", currentToken)
					return false, nil
				}

				return true, nil
			}
		}
		return false, nil
	})
}

// doRefreshToken uses the remote kubeadm for connecting to the API server, checking if the Token is still valid
// and create a new token otherwise
func doRefreshToken(d *schema.ResourceData) ssh.Action {
	curTokenInJoinConfig := getTokenFromResourceData(d)
	curTokens := KubeadmTokensSet{}

	// create a new, random curTokenInJoinConfig
	newToken, err := common.GetRandomToken()
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("cannot create new random token: %s", err))
	}

	return ssh.ActionList{
		ssh.DoMessageInfo("Checking if current token is still valid..."),
		ssh.DoIfElse(
			checkTokenIsValid(d, curTokens),
			ssh.DoMessageInfo("%q is still a valid token", curTokenInJoinConfig),
			ssh.ActionList{
				ssh.DoMessageWarn("%q is not valid token anymore: will create a new token %q...", curTokenInJoinConfig, newToken),
				ssh.DoSendingExecOutputToDevNull(DoExecKubeadmToken(d, fmt.Sprintf("create --ttl=%s %s", newJoinTokenTTL, newToken))),
				DoSetNewToken(d, newToken),
				ssh.DoMessage("New token %q created successfully.", newToken),
			}),
	}
}
