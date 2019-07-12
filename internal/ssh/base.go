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
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// a prefix for all the outputs
	commonMsgPrefix = ""
)

type Config struct {
	UserOutput terraform.UIOutput
	ExecOutput terraform.UIOutput
	Comm       communicator.Communicator
	UseSudo    bool
}

func (cfg Config) GetExecOutput() terraform.UIOutput {
	if cfg.ExecOutput != nil {
		return cfg.ExecOutput
	}
	return cfg.UserOutput
}

///////////////////////////////////////////////////////////////////////////////////////////////

// Action is an action that can be "applied"
type Action interface {
	error
	Apply(Config) Action
}

///////////////////////////////////////////////////////////////////////////////////////////////

// ActionFunc is a function that can be converted to a `Action`
//
// ie: 	ActionFunc(func(Config) error {
// 			return nil
// }),
type ActionFunc func(Config) Action

// Apply applies an action
func (f ActionFunc) Apply(c Config) Action {
	return f(c)
}

func (f ActionFunc) Error() string {
	return ""
}

///////////////////////////////////////////////////////////////////////////////////////////////

// ActionError is an error for an Action
type ActionError string

// Apply applies an action
func (ae ActionError) Apply(Config) Action {
	return ae
}

func (ae ActionError) Error() string {
	return string(ae)
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

// applyList runs a list of actions, optionally ignoring errors
func applyList(actions ActionList, ignoreErrors bool, cfg Config) Action {
	// use a queue where we take and put things in the front
	// note that we operate on a copy of the original actions list
	for len(actions) > 0 {
		if !ignoreErrors {
			// if some error is in the queue, just quit with that error
			for _, action := range actions {
				if IsError(action) {
					return action
				}
			}
		}

		// otherwise, consume from the queue: pop the first element
		cur := actions[0]
		actions = actions[1:]

		// if it nil, pass to the next element
		if cur == nil {
			continue
		}

		// if it is an error, it depends on if we ignore them
		if IsError(cur) {
			if ignoreErrors {
				continue
			} else {
				return cur
			}
		}

		// otherwise, run the action
		res := cur.Apply(cfg)
		// ... and add the resulting actions in front of the queue
		switch v := res.(type) {
		case ActionList:
			// if it is a list, expand it
			actions = append(v, actions...)
		default:
			actions = append([]Action{res}, actions...)
		}
	}
	return nil
}

// ActionList is a list of Actions
type ActionList []Action

// Apply applies an action
func (actions ActionList) Apply(cfg Config) Action {
	return applyList(actions, false, cfg)
}

func (actions ActionList) Error() string {
	for _, action := range actions {
		if IsError(action) {
			return action.Error()
		}
	}
	return ""
}

///////////////////////////////////////////////////////////////////////////////////////////////

// DoNothing is a dummy action
func DoNothing() Action {
	return ActionFunc(func(Config) Action {
		return nil
	})
}

func Debug(format string, args ...interface{}) {
	log.Printf("[DEBUG] [KUBEADM] "+format, args...)
}

func DoMessageRaw(msg string) Action {
	return ActionFunc(func(cfg Config) Action {
		cfg.UserOutput.Output(msg)
		return nil
	})
}

func DoMessageWithColor(msg string, c color.Color) Action {
	return DoMessageRaw(commonMsgPrefix + c.Render(msg))
}

// DoMessage is a dummy action that just prints a message
func DoMessage(format string, args ...interface{}) Action {
	return DoMessageWithColor(fmt.Sprintf(format, args...), color.FgLightGreen)
}

func DoMessageWarn(format string, args ...interface{}) Action {
	msg := fmt.Sprintf("WARNING: "+format, args...)
	return DoMessageWithColor(msg, color.FgRed)
}

func DoMessageInfo(format string, args ...interface{}) Action {
	return DoMessageWithColor(fmt.Sprintf(format, args...), color.FgGreen)
}

// DoMessageDebug prints a debug message
func DoMessageDebug(format string, args ...interface{}) Action {
	return ActionFunc(func(cfg Config) Action {
		Debug(format, args...)
		return nil
	})
}

// DoAbort is an action that prints an error message and exits
func DoAbort(format string, args ...interface{}) Action {
	msg := fmt.Sprintf("FATAL: "+format, args...)
	coloredMsg := color.Style{color.FgRed, color.OpBold}.Render(msg)
	return ActionList{
		DoMessageRaw(coloredMsg),
		ActionError(fmt.Sprintf(format, args...)),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////

// Checker implements a Check method
type Checker interface {
	Check(Config) (bool, error)
}

// CheckerFunc is a function that implements the Checker interface
type CheckerFunc func(Config) (bool, error)

// Check implements the Checker interface in CheckerFuncs
func (f CheckerFunc) Check(cfg Config) (bool, error) {
	return f(cfg)
}

// DoWithCleanup runs some action and, despite the result, runs the cleanup function
// It returns the actions result.
func DoWithCleanup(cleanup Action, actions Action) Action {
	return ActionFunc(func(cfg Config) Action {
		res := ActionList{actions}.Apply(cfg)
		_ = ActionList{cleanup}.Apply(cfg)
		return res
	})
}

// DoWithError runs some action and, if some error happens, runs the exception
func DoWithException(exc Action, actions Action) Action {
	return ActionFunc(func(cfg Config) Action {
		res := ActionList{actions}.Apply(cfg)
		if IsError(res) {
			_ = ActionList{exc}.Apply(cfg)
		}
		return res
	})
}

// DoIf runs an action iff the condition is true
func DoIf(condition Checker, action Action) Action {
	return ActionFunc(func(cfg Config) Action {
		checkPassed, err := condition.Check(cfg)
		if err != nil {
			return ActionError(err.Error())
		}
		if checkPassed {
			return ActionList{action}.Apply(cfg)
		}
		return nil
	})
}

// DoIfElse runs an action iff the condition is true, otherwise runs a different action
func DoIfElse(condition Checker, actionIf Action, actionElse Action) Action {
	return ActionFunc(func(cfg Config) Action {
		checkPassed, err := condition.Check(cfg)
		if err != nil {
			return ActionError(fmt.Sprintf("could not check condition: %s", err.Error()))
		}

		if checkPassed {
			return ActionList{actionIf}.Apply(cfg)
		}
		return ActionList{actionElse}.Apply(cfg)
	})
}

// DoTry tries to run some actions, but it is ok if some action fails
func DoTry(actions ...Action) Action {
	return ActionFunc(func(cfg Config) Action {
		return applyList(actions, true, cfg)
	})
}

// DoRetry runs an action `n` times until it succeedes
func DoRetry(times int, actions ...Action) ActionFunc {
	return ActionFunc(func(cfg Config) Action {
		count := times
		var res Action
		for count > 0 {
			res = ActionList(actions).Apply(cfg)
			if IsError(res) {
				time.Sleep(1 * time.Second)
				count -= 1
			} else {
				return res
			}
		}
		return res
	})
}

// DoSendingExecOutputToFun runs some action redirecting all the Do***Exec outputs
// to some function
// Some notes:
// * make sure you strip spaces in the output, as some extra spaces can be before/after
func DoSendingExecOutputToFun(interceptor OutputFunc, action ...Action) Action {
	return ActionFunc(func(cfg Config) Action {
		t := cfg
		t.ExecOutput = interceptor
		return ActionList(action).Apply(t)
	})
}

// DoSendingExecOutputToWriter runs some action redirecting all the Do***Exec outputs
// to some io.Writer
// Some notes:
// * make sure you strip spaces in the output, as some extra spaces can be before/after
func DoSendingExecOutputToWriter(writer io.Writer, action Action) Action {
	return DoSendingExecOutputToFun(func(s string) {
		c := strings.ReplaceAll(s, "\r", "\n")
		_, _ = writer.Write([]byte(c))
	}, action)
}

// DoSendingExecOutputToDevNull runs some action redirecting all the Do***Exec outputs
// to /dev/null
// Some notes:
// * make sure you trip spaces in the output, as some extra spaces can be before/after
func DoSendingExecOutputToDevNull(action Action) Action {
	return DoSendingExecOutputToFun(func(s string) {
		// do nothing with "s"
	}, action)
}

// CheckExpr returns the result of the boolean expression
func CheckExpr(expr bool) CheckerFunc {
	return CheckerFunc(func(Config) (bool, error) {
		return expr, nil
	})
}

// CheckAction returns true if the Action does not return an error
func CheckAction(action Action) CheckerFunc {
	return CheckerFunc(func(cfg Config) (bool, error) {
		if res := action.Apply(cfg); IsError(res) {
			return false, nil
		}
		return true, nil
	})
}

func CheckFailed() CheckerFunc {
	return CheckExpr(false)
}

func CheckError(err error) CheckerFunc {
	return CheckerFunc(func(Config) (bool, error) {
		return false, err
	})
}

// CheckAnd applies a logical And on a group of Checks
func CheckAnd(checks ...Checker) CheckerFunc {
	return CheckerFunc(func(cfg Config) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(cfg)
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
	return CheckerFunc(func(cfg Config) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(cfg)
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
	return CheckerFunc(func(cfg Config) (bool, error) {
		res, err := check.Check(cfg)
		if err != nil {
			return false, err
		}
		return !res, nil
	})
}

// //////////////////////////////////////////////////////////////////////////////////////

type OutputFunc func(s string)

func (f OutputFunc) Output(s string) { f(s) }
