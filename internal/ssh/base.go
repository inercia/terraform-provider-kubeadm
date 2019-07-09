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
	"io"
	"log"

	"github.com/gookit/color"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// Action is an action that can be "applied"
type Action interface {
	error
	Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action
}

///////////////////////////////////////////////////////////////////////////////////////////////

// ActionFunc is a function that can be converted to a `Action`
//
// ie: 	ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
// 			return nil
// }),
type ActionFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action

// Apply applies an action
func (f ActionFunc) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
	return f(o, comm, useSudo)
}

func (f ActionFunc) Error() string {
	return ""
}

///////////////////////////////////////////////////////////////////////////////////////////////

// ActionError is an error for an Action
type ActionError string

// Apply applies an action
func (_ ActionError) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
	return nil
}

func (s ActionError) Error() string {
	return string(s)
}

func IsError(a Action) bool {
	if a == nil {
		return false
	}
	t, ok := a.(ActionError)
	if !ok {
		return false
	}
	return t.Error() != ""
}

///////////////////////////////////////////////////////////////////////////////////////////////

// ActionList is a list of Actions
type ActionList []Action

// Apply applies an action
func (actions ActionList) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
	localActions := actions[:]

	// use a queue where we take and put things in the front
	for len(localActions) > 0 {
		// if some action is in the queue, just quit with that error
		for _, action := range localActions {
			if IsError(action) {
				return action
			}
		}

		// otherwise, consume from the queue: pop the first element
		cur := localActions[0]
		localActions = localActions[1:]

		// if it nil, pass to the next element
		if cur == nil {
			continue
		}

		// otherwise, run the action
		res := cur.Apply(o, comm, useSudo)

		// ... and add the resulting actions in front of the queue
		localActions = append([]Action{res}, localActions...)
	}
	return nil
}

func (actions ActionList) Error() string {
	for _, action := range actions {
		if IsError(action) {
			return action.Error()
		}
	}
	return ""
}

func DoLazy(g func() Action) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		return g()
	})
}

///////////////////////////////////////////////////////////////////////////////////////////////

// DoNothing is a dummy action
func DoNothing() Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		return nil
	})
}

func DoMessageRaw(msg string) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		o.Output(msg)
		return nil
	})
}

func DoMessageWithColor(msg string, c color.Color) Action {
	return DoMessageRaw(c.Render(msg))
}

// DoMessage is a dummy action that just prints a message
func DoMessage(format string, args ...interface{}) Action {
	return DoMessageWithColor(fmt.Sprintf(format, args...), color.FgLightGreen)
}

func DoMessageWarn(format string, args ...interface{}) Action {
	return DoMessageWithColor(fmt.Sprintf("WARNING: "+format, args...), color.FgLightGreen)
}

func DoMessageInfo(msg string) Action {
	return DoMessageWithColor(msg, color.FgLightGreen)
}

// DoMessageDebug prints a debug message
func DoMessageDebug(format string, args ...interface{}) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		log.Printf(color.FgLightYellow.Render(fmt.Sprintf(fmt.Sprintf("[DEBUG] [KUBEADM] "+format, args...))))
		return nil
	})
}

// DoAbort is an action that prints an error message and exits
func DoAbort(format string, args ...interface{}) Action {
	coloredMsg := color.Style{color.FgRed, color.OpBold}.Render(fmt.Sprintf(fmt.Sprintf("FATAL: "+format, args...)))

	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		o.Output(coloredMsg)
		return ActionError(fmt.Sprintf(format, args...))
	})
}

///////////////////////////////////////////////////////////////////////////////////////////////

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

// DoWithCleanup runs some action and, despite the result, runs the cleanup function
func DoWithCleanup(action, cleanup Action) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		if action == nil || IsError(action) {
			return action
		}
		res := action.Apply(o, comm, useSudo)
		if cleanup != nil || !IsError(cleanup) {
			_ = cleanup.Apply(o, comm, useSudo)
		}
		return res
	})
}

// DoWithError runs some action and, if some error happens, runs the exception
func DoWithException(action, exc Action) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		if action == nil || IsError(action) {
			return action
		}
		res := action.Apply(o, comm, useSudo)
		if IsError(res) && exc != nil {
			_ = exc.Apply(o, comm, useSudo)
		}
		return res
	})
}

// DoIf runs an action iff the condition is true
func DoIf(condition Checker, action Action) ActionFunc {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return ActionError(err.Error())
		}
		if res {
			if action != nil {
				return action.Apply(o, comm, useSudo)
			}
		}
		return nil
	})
}

// DoIfElse runs an action iff the condition is true, otherwise runs a different action
func DoIfElse(condition Checker, actionIf Action, actionElse Action) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return ActionError(fmt.Sprintf("could not check condition: %s", err.Error()))
		}

		if res {
			if actionIf != nil {
				return actionIf.Apply(o, comm, useSudo)
			}
			return nil
		}
		if actionElse != nil {
			return actionElse.Apply(o, comm, useSudo)
		}
		return nil
	})
}

// DoTry tries to run some actions, but it is ok if some action fails
func DoTry(actions ...Action) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		// use a queue where we take and put things in the front
		localActions := actions[:]
		for len(localActions) > 0 {
			// consume from the queue: pop the first element
			cur := localActions[0]
			localActions = localActions[1:]

			// if it is nil/error, pass to the next element
			if cur == nil {
				continue
			}

			// replace error by warnings
			if IsError(cur) {
				cur = DoMessageWarn("%s (IGNORED)", cur.Error())
			}

			// otherwise, run the action
			res := cur.Apply(o, comm, useSudo)

			// ... and add the resulting actions in front of the queue
			localActions = append([]Action{res}, localActions...)
		}
		return nil
	})
}

// DoSendingOutputToFun runs some action redirecting each line of stdout/stderr to some function
// make sure you trip spaces in the output, as some extra spaces can be before/after
func DoSendingOutputToFun(action Action, interceptor OutputFunc) Action {
	if action == nil || IsError(action) {
		return action
	}
	return ActionFunc(func(_ terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		return action.Apply(interceptor, comm, useSudo)
	})
}

// DoSendingOutputToWriter runs some action redirecting the output to some io.Writer
// make sure you trip spaces in the output, as some extra spaces can be before/after
func DoSendingOutputToWriter(action Action, writer io.Writer) Action {
	return DoSendingOutputToFun(action, func(s string) {
		_, _ = writer.Write([]byte(s))
	})
}

// CheckExpr returns the result of the boolean expression
func CheckExpr(expr bool) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		return expr, nil
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
