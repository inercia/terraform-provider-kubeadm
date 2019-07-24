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
)

// DoMkdir creates a remote directory
func DoMkdir(path string) Action {
	mkdirCmd := fmt.Sprintf("mkdir -p %s", path)
	return ActionList{
		DoMessageDebug(fmt.Sprintf("Making sure directory %q exists", path)),
		DoExec(mkdirCmd),
	}
}

// DoMkdirOnce creates a remote directory (only once).
// We don't really delete any remote directory, so running a `mkdir` for a
// remote directory only once can be considered safe.
func DoMkdirOnce(dir string) Action {
	return DoOnce(
		CacheRemoteDirExistsPrefix+"-"+dir,
		DoMkdir(dir))
}

// CheckDirExists checks that a directory exists
func CheckDirExists(path string) CheckerFunc {
	return CheckExec(fmt.Sprintf("[ -d '%s' ]", path))
}
