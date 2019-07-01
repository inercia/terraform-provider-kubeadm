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
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// DoMkdir creates a remote directory
func DoMkdir(path string) ApplyFunc {
	mkdirCmd := fmt.Sprintf("mkdir -p %s", path)
	return DoComposed(
		DoMessage(fmt.Sprintf("Creating directory %s", path)),
		DoExec(mkdirCmd),
	)
}

// CheckDirExists checks that a directory exists
func CheckDirExists(path string) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		const markFound = "YES_IT_IS_THERE"
		found := false
		var interceptor OutputFunc = func(s string) {
			if strings.Contains(s, markFound) {
				found = true
			}
		}

		command := fmt.Sprintf("[ -d '%s' ] && echo '%s'", path, markFound)
		err := DoExec(command).Apply(interceptor, comm, useSudo)
		if err != nil {
			return false, err
		}

		return found, nil
	})
}
