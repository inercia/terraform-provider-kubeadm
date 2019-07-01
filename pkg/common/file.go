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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	isURL = iota
	isFile
)

// GetFileType identifies if a strring represents a file or a URL
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

// ReadHTTPFile reads file from a http url
func ReadHTTPFile(targetURL *url.URL) (string, error) {
	var client http.Client
	resp, err := client.Get(targetURL.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	return "", fmt.Errorf("could not fetch data from %s", targetURL.String())
}

// ReadLocalFile reads file from a local path and returns as string
func ReadLocalFile(sourcePath string) (string, error) {
	// If sourcePath starts with ~ we search for $HOME
	// and preppend it to the absolutePath overwriting the first character
	// TODO: Add Windows support
	if strings.HasPrefix(sourcePath, "~") {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return "", fmt.Errorf("Could not find $HOME")
		}
		sourcePath = filepath.Join(homeDir, sourcePath[1:])
	}

	bytes, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetSafeTempDirectory returns a temporary, safe directory
func GetSafeTempDirectory() (string, error) {
	// create a temporary directory for the certificates and try to download them
	// TODO: maybe we should use os.UserCacheDir() for the dir...
	t, err := ioutil.TempDir("", "terraform")
	if err != nil {
		return "", err
	}
	return t, nil
}
