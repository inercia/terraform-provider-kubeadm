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
	"io/ioutil"
	"net/url"
	"strings"
)

const (
	isURL = iota
	isFile
)

// GetFileType identifies if a string represents a file or a URL
func GetFileType(r string) (int, error) {
	switch {
	case strings.HasPrefix(strings.ToLower(r), "http://") || strings.HasPrefix(strings.ToLower(r), "https://"):
		if _, err := url.ParseRequestURI(r); err != nil {
			return 0, err
		}
		return isURL, nil

	default:
		return isFile, nil
	}
}

// GetSafeLocalTempDirectory returns a temporary, safe directory
func GetSafeLocalTempDirectory() (string, error) {
	// create a temporary directory for the certificates and try to download them
	// TODO: maybe we should use os.UserCacheDir() for the dir...
	t, err := ioutil.TempDir("", "terraform")
	if err != nil {
		return "", err
	}
	return t, nil
}
