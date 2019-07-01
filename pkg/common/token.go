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
	"crypto/rand"
	"encoding/hex"
	"fmt"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

const (
	TokenIDBytes     = 3
	TokenSecretBytes = 8
)

func randBytes(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetRandomToken generates a new token with a token ID that is valid as a
// Kubernetes DNS label.
// For more info, see kubernetes/pkg/util/validation/validation.go.
func GetRandomToken() (string, error) {
	tokenID, err := randBytes(TokenIDBytes)
	if err != nil {
		return "", err
	}

	tokenSecret, err := randBytes(TokenSecretBytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", tokenID, tokenSecret), nil
}

func NewBootstrapToken(token string) (kubeadmapi.BootstrapToken, error) {
	var err error
	bto := kubeadmapi.BootstrapToken{}
	bto.Token, err = kubeadmapi.NewBootstrapTokenString(token)
	if err != nil {
		return kubeadmapi.BootstrapToken{}, err
	}
	return bto, err
}

func NewRandomBootstrapToken() (kubeadmapi.BootstrapToken, error) {
	t, err := GetRandomToken()
	if err != nil {
		return kubeadmapi.BootstrapToken{}, err
	}
	return NewBootstrapToken(t)
}
