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
	"log"
)

// CheckProcessRunning checks that a process is running with the help of `ps`
// FIXME: this is not really reliable, as it looks for a string in tne output
//        of `ps`, and that string can be part of some other command...
func CheckProcessRunning(process string) CheckerFunc {
	check := fmt.Sprintf(`[ -n "$(ps ax | grep %s | grep -v grep)" ]`, process)
	return CheckExec(check)
}

// DoRestartService restart a systemctl service
func DoRestartService(service string) Action {
	return ActionList{
		DoMessageInfo(fmt.Sprintf("Restarting service %s", service)),
		DoExec(fmt.Sprintf("systemctl --no-pager restart '%s'", service)),
	}
}

// DoEnableService enables a systemctl service
func DoEnableService(service string) Action {
	return ActionList{
		DoMessageInfo(fmt.Sprintf("Enabling service %s", service)),
		DoExec(fmt.Sprintf("systemctl --no-pager enable '%s'", service)),
	}
}

// CheckServiceExists checks that service exists
func CheckServiceExists(service string) CheckerFunc {
	log.Printf("[DEBUG] Checking if service '%s' exists", service)
	exists := fmt.Sprintf("systemctl --no-pager status '%s' 2>/dev/null", service)
	return CheckExec(exists)
}

// CheckServiceActive checks that service exists and is active
func CheckServiceActive(service string) CheckerFunc {
	inactive := fmt.Sprintf("systemctl --no-pager status '%s' 2>/dev/null | grep Active | grep -q inactive", service)
	return CheckNot(
		CheckAnd(CheckServiceExists(service),
			CheckExec(inactive)))
}
