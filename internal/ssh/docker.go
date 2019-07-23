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
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	// docker command for getting a container id
	dockerGetContainer = "docker ps --filter name=^/%s -q"
)

var (
	ErrContainerNotFound = errors.New("container not found")
)

// getContainer returns the ID of a container
func GetContainer(ctx context.Context, pattern string) (string, error) {

	cmd := fmt.Sprintf(dockerGetContainer, pattern)
	var buf bytes.Buffer
	if err := DoSendingExecOutputToWriter(DoExec(cmd), &buf).Apply(ctx); IsError(err) {
		return "", err
	}

	output := buf.String()
	if len(output) == 0 {
		return "", ErrContainerNotFound
	}

	output = strings.ReplaceAll(output, "\n", "")
	output = strings.TrimSpace(output)

	Debug("GetContainer(%s) output: %q", pattern, output)
	return output, nil
}

// DoDockerExec runs a `docker exec` command in a container
func DoDockerExec(pattern string, command string) Action {
	return ActionFunc(func(ctx context.Context) Action {
		cid, err := GetContainer(ctx, pattern)
		if err != nil {
			return ActionError(err.Error())
		}

		// build the full `docker exec` command to run
		dockerCommand := fmt.Sprintf("docker exec -ti '%s' /bin/sh -c '%s'", cid, command)

		Debug("Running command in container %q: '%s'", cid, dockerCommand)
		return DoExec(dockerCommand)
	})
}

// CheckContainerRunning checks if we can get the CID for a pattern
func CheckContainerRunning(pattern string) CheckerFunc {
	return CheckerFunc(func(ctx context.Context) (bool, error) {
		cid, err := GetContainer(ctx, pattern)
		if err != nil {
			return false, nil
		}
		if cid == "" {
			return false, nil
		}
		return true, nil
	})
}
