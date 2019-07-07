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
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

////////////////////////////////////////////////////////////////////////////////////////////////

func TestApply(t *testing.T) {
	counter := 0
	errorMsg := "some error"
	actions := ActionList{
		DoMessage("test"),
		ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			counter = counter + 1
			return nil
		}),
		nil,
		nil,
		ActionError(errorMsg),
	}

	o := DummyOutput{}
	comm := DummyCommunicator{}

	err := actions.Apply(o, comm, false)
	if err == nil {
		t.Fatal("Error: no error detected")
	}
	if err.Error() != errorMsg {
		t.Fatalf("Error: unexpected error message: %q", err.Error())
	}
	if counter > 0 {
		t.Fatal("Error: error was raised after some function was run")
	}

}

func TestDoCatchingOutput(t *testing.T) {
	expected := "this is a test"

	var buf bytes.Buffer
	actions := ActionList{
		DoSendingOutputToWriter(
			ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
				o.Output(expected)
				return nil
			}), &buf),
	}

	o := DummyOutput{}
	comm := DummyCommunicator{}

	err := actions.Apply(o, comm, false)
	if err != nil {
		t.Fatalf("Error: error detected: %s", err)
	}
	if buf.String() != expected {
		t.Fatalf("Error: the output, %q, has not been the expected value: %q", buf.String(), expected)
	}
}

func TestDoLazy(t *testing.T) {
	expected := "12345678"

	path := ""
	doRecordPath := func(num string) Action {
		return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			path += num
			return nil
		})
	}

	actions := ActionList{
		doRecordPath("1"),
		nil,
		DoLazy(func() Action {
			return ActionList{
				doRecordPath("2"),
				doRecordPath("3"),
				DoLazy(func() Action {
					return ActionList{
						doRecordPath("4"),
						nil,
						doRecordPath("5"),
					}
				}),
			}
		}),
		DoLazy(func() Action {
			return ActionList{
				nil,
				doRecordPath("6"),
				DoIfElse(CheckExpr(true),
					DoLazy(func() (res Action) {
						return doRecordPath("7")
					}),
					DoLazy(func() (res Action) {
						return doRecordPath("XXX")
					}),
				),
			}
		}),
		nil,
		nil,
		doRecordPath("8"),
	}

	var buf bytes.Buffer
	o := DummyOutput{}
	comm := DummyCommunicator{}
	res := DoSendingOutputToWriter(&actions, &buf).Apply(o, comm, false)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if path != expected {
		t.Fatalf("Error: unexpected contents: %q, expected: %q", path, expected)
	}
}

func TestIfThenElse(t *testing.T) {
	path := ""
	doRecordPath := func(num string) Action {
		return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			path += num
			return nil
		})
	}

	expected := "000111222"
	actions := ActionList{
		doRecordPath("000"),
		DoIfElse(CheckExpr(true),
			DoLazy(func() (res Action) {
				return doRecordPath("111")
			}),
			DoLazy(func() (res Action) {
				return doRecordPath("XXX")
			}),
		),
		DoIfElse(CheckExpr(true),
			nil,
			DoLazy(func() (res Action) {
				return doRecordPath("YYY")
			}),
		),
		DoIfElse(CheckExpr(false),
			DoLazy(func() (res Action) {
				return doRecordPath("YYY")
			}),
			nil,
		),
		nil,
		nil,
		doRecordPath("222"),
	}

	var buf bytes.Buffer
	o := DummyOutput{}
	comm := DummyCommunicator{}
	res := DoSendingOutputToWriter(&actions, &buf).Apply(o, comm, false)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if path != expected {
		t.Fatalf("Error: unexpected contents: %q, expected: %q", path, expected)
	}
}

func TestDoTry(t *testing.T) {
	expected := "01234"

	path := ""
	doRecordPath := func(num string) Action {
		return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
			path += num
			return nil
		})
	}

	actions := ActionList{
		DoTry(
			doRecordPath("0"),
			DoTry(
				doRecordPath("1"),
				doRecordPath("2"),
			),
			nil,
			nil,
			ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
				return ActionError("some error")
			}),
			// this ActionList is never executed, as the presence of an error makes the whole list errored
			ActionList{
				doRecordPath("XXX"),
				ActionError("some error"),
			},
			DoLazy(func() (res Action) {
				return ActionList{
					doRecordPath("3"),
				}
			}),
			nil,
		),
		nil,
		nil,
		doRecordPath("4"),
	}

	var buf bytes.Buffer
	o := DummyOutput{}
	comm := DummyCommunicator{}
	res := DoSendingOutputToWriter(&actions, &buf).Apply(o, comm, false)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if path != expected {
		t.Fatalf("Error: unexpected contents: %q, expected: %q", path, expected)
	}
}
