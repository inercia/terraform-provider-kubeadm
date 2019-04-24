package ssh

import (
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

type Action interface {
	Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error
}

// ApplyFunc is a function that can be converted to a `Action`
//
// ie: 	ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
// 			return nil
// }),
type ApplyFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error

func (f ApplyFunc) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	return f(o, comm, useSudo)
}

func EmptyAction() ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return nil
	})
}

// RunActions applies a list of actions
func RunActions(actions []Action, o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	for _, action := range actions {
		if err := action.Apply(o, comm, useSudo); err != nil {
			return err
		}
	}
	return nil
}

// Compose a list of actions as a single ApplyFunc
func Composite(actions ...Action) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return RunActions(actions, o, comm, useSudo)
	})
}

// ///////////////////////////////////////////////////////////////////////////////////

// Checker implements a Check method
type Checker interface {
	Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)
}

type CheckerFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)

func (f CheckerFunc) Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
	return f(o, comm, useSudo)
}

// ActionIf runs an action iff the condition is true
func ActionIf(condition Checker, action Action) ApplyFunc {
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

// ActionIfElse runs an action iff the condition is true, otherwise runs a different action
func ActionIfElse(condition Checker, actionIf Action, actionElse Action) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return err
		}

		if res {
			return actionIf.Apply(o, comm, useSudo)
		} else {
			return actionElse.Apply(o, comm, useSudo)
		}
		return nil
	})
}
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
