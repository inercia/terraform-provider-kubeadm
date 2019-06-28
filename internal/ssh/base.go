package ssh

import (
	"fmt"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// Applyer is an action that can be "applied"
type Applyer interface {
	Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error
}

// ApplyFunc is a function that can be converted to a `Applyer`
//
// ie: 	ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
// 			return nil
// }),
type ApplyFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error

// Apply applies an action
func (f ApplyFunc) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	return f(o, comm, useSudo)
}

// Apply applies a list of actions
func Apply(actions []Applyer, o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	for _, action := range actions {
		if action != nil {
			if err := action.Apply(o, comm, useSudo); err != nil {
				return err
			}
		}
	}
	return nil
}

// DoNothing is a dummy action
func DoNothing() ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return nil
	})
}

// DoMessage is a dummy action that just prints a message
func DoMessage(msg string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		o.Output(msg)
		return nil
	})
}

// DoAbort is an action that prints an error message and exits
func DoAbort(msg string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		o.Output(fmt.Sprintf("ERROR: %s", msg))
		return fmt.Errorf("ERROR: %s", msg)
	})
}

// ///////////////////////////////////////////////////////////////////////////////////

// Checker implements a Check method
type Checker interface {
	Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)
}

// CheckerFunc is a function that implements the Checker interface
type CheckerFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)

// Check implements the Checker interface in CheckerFuncs
func (f CheckerFunc) Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
	return f(o, comm, useSudo)
}

// DoComposed composes from a list of actions a single ApplyFunc
func DoComposed(actions ...Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return Apply(actions, o, comm, useSudo)
	})
}

// DoIf runs an action iff the condition is true
func DoIf(condition Checker, action Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return err
		}

		if res {
			return action.Apply(o, comm, useSudo)
		}
		return nil
	})
}

// DoIfElse runs an action iff the condition is true, otherwise runs a different action
func DoIfElse(condition Checker, actionIf Applyer, actionElse Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return err
		}

		if res {
			return actionIf.Apply(o, comm, useSudo)
		}
		return actionElse.Apply(o, comm, useSudo)
	})
}

// DoTry tries to run an action, but it is ok if the action fails
func DoTry(action Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		_ = action.Apply(o, comm, useSudo)
		return nil
	})
}

// CheckAnd applies a logical And on a group of Checks
func CheckAnd(checks ...Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(o, comm, useSudo)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}
		return true, nil
	})
}

// CheckOr applies a logical Or on a group of Checks
func CheckOr(checks ...Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(o, comm, useSudo)
			if err != nil {
				return false, err
			}
			if pass {
				return true, nil
			}
		}
		return false, nil
	})
}

// CheckNot return the logical Not of a Check
func CheckNot(check Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		res, err := check.Check(o, comm, useSudo)
		if err != nil {
			return false, err
		}
		return !res, nil
	})
}

// //////////////////////////////////////////////////////////////////////////////////////

type OutputFunc func(s string)

func (f OutputFunc) Output(s string) { f(s) }
