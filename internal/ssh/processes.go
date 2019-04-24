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
	return CheckCondition(check)
}

// DoRestartService restart a systemctl service
func DoRestartService(service string) ApplyFunc {
	return DoExec(fmt.Sprintf("systemctl restart %s", service))
}

// DoEnableService enables a systemctl service
func DoEnableService(service string) ApplyFunc {
	log.Printf("[DEBUG] Enabling service '%s'", service)
	return DoExec(fmt.Sprintf("systemctl enable %s", service))
}

// CheckServiceExists checks that service exists
func CheckServiceExists(service string) CheckerFunc {
	log.Printf("[DEBUG] Checking if service '%s' exists", service)
	exists := fmt.Sprintf("systemctl status '%s' 2>/dev/null", service)
	return CheckCondition(exists)
}

// CheckServiceActive checks that service exists and is active
func CheckServiceActive(service string) CheckerFunc {
	inactive := fmt.Sprintf("systemctl status '%s' 2>/dev/null | grep Active | grep -q inactive", service)
	return CheckNot(
		CheckAnd(CheckServiceExists(service),
			CheckCondition(inactive)))
}
