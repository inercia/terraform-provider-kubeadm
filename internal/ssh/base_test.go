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
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////

func TestApply(t *testing.T) {
	counter := 0
	errorMsg := "some error"
	actions := ActionList{
		DoMessage("test"),
		ActionFunc(func(context.Context) Action {
			counter = counter + 1
			return nil
		}),
		nil,
		nil,
		ActionError(errorMsg),
	}

	ctx := NewTestingContext()
	err := actions.Apply(ctx)
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

func TestDoSendingOutput(t *testing.T) {
	expected := "1234"

	var buf bytes.Buffer
	var buf2 bytes.Buffer
	actions := ActionList{
		DoSendingExecOutputToWriter(&buf, ActionList{
			ActionFunc(func(ctx context.Context) Action {
				return doEcho("1")
			}),
			DoSendingExecOutputToWriter(&buf2,
				ActionFunc(func(context.Context) Action {
					return doEcho("(this is another message that should go to another buffer)")
				})),
			ActionFunc(func(context.Context) Action {
				return doEcho("2")
			}),
			DoIfElse(
				CheckLocalFileExists("/tmp/some/file/that/does/not/exist"),
				DoLocalExec("ls /"),
				ActionFunc(func(context.Context) Action {
					return ActionList{
						doEcho("3"),
						doEcho("4"),
					}
				})),
		}),
	}

	ctx := NewTestingContext()
	err := actions.Apply(ctx)
	if err != nil {
		t.Fatalf("Error: error detected: %s", err)
	}
	t.Logf("Received: %q", buf.String())
	if buf.String() != expected {
		t.Fatalf("Error: %q was not expected. We expected %q", buf.String(), expected)
	}
}

//func doLazy(af ActionFunc) func()Action {
//	return func()Action {
//		af.Apply()
//	}
//}

func DoLazy(af ActionFunc) func() Action {
	return func() Action {
		return ActionFunc(func(ctx context.Context) Action {
			return af(ctx)
		})
	}
}

func tFunc(something string) Action {
	return ActionFunc(func(context.Context) Action {
		return doEcho("(OOOO)")
	})
}

func TestDoSendingOutputToFun(t *testing.T) {
	expected := "12345"
	received := ""
	trashBuffer := ""

	//tFunc := func() Action {
	//	return ActionFunc(func(context.Context) Action {
	//		return doEcho("(and to trash)")
	//	})
	//}

	actions := ActionList{
		DoSendingExecOutputToFun(func(s string) {
			received += s
		}, ActionList{
			ActionFunc(func(ctx context.Context) Action {
				return doEcho("1")
			}),
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				}, ActionFunc(func(context.Context) Action {
					return doEcho("(VVVV)")
				})),
			DoSendingExecOutputToDevNull(DoLocalExec("ls", "/tmp")),
			// this works:
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				},
				doEcho("(XXXX)")),
			// this works:
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				},
				ActionList{
					doEcho("(MMMM)"),
					doEcho("(NNNN)"),
					ActionFunc(func(context.Context) Action {
						return doEcho("(LLLL)")
					}),
				}),
			// this doesnt
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				},
				func() Action {
					return ActionFunc(func(context.Context) Action {
						return doEcho("(YYYY)")
					})
				}()),
			// this doesnt
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				},
				tFunc("")),
			// this doesnt
			DoSendingExecOutputToFun(
				func(s string) {
					trashBuffer += s
				},
				DoLazy(
					ActionFunc(func(context.Context) Action {
						return doEcho("(ZZZZ)")
					}))()),
			ActionFunc(func(context.Context) Action {
				return doEcho("2")
			}),
			DoTry(ActionFunc(func(context.Context) Action {
				return doEcho("3")
			})),
			DoTry(ActionError("an error")),
			DoWithCleanup(DoNothing(), ActionFunc(func(context.Context) Action {
				return doEcho("4")
			})),
			DoIfElse(CheckExpr(false),
				nil,
				ActionFunc(func(context.Context) Action {
					return doEcho("5")
				}),
			),
		}),
	}

	ctx := NewTestingContext()
	err := actions.Apply(ctx)
	if err != nil {
		t.Fatalf("Error: error detected: %s", err)
	}
	t.Logf("Received: %q", received)
	if received != expected {
		t.Fatalf("Error: %q was not expected. We expected %q", received, expected)
	}
}

// TestDoSendingOutputToFunWithError checks that we can send output to
// a function and an Error aborts the whole execution
func TestDoSendingOutputToFunWithError(t *testing.T) {
	received := ""
	someOtherBuffer := ""

	actions := ActionList{
		DoSendingExecOutputToFun(func(s string) {
			received += s
		}, ActionList{
			ActionFunc(func(ctx context.Context) Action {
				return doEcho("1")
			}),
			DoSendingExecOutputToFun(
				func(s string) {
					someOtherBuffer += s
				},
				ActionFunc(func(context.Context) Action {
					return doEcho("'this is another message that should go to another buffer'")
				})),
			DoTry(ActionError("this should be ignored")),
			DoIfElse(CheckExpr(false),
				nil,
				ActionFunc(func(context.Context) Action {
					return doEcho("2")
				}),
			),
			ActionError("some error"),
		}),
	}

	ctx := NewTestingContext()
	err := actions.Apply(ctx)
	if err == nil {
		t.Fatalf("Error: no error detected (and we expected one)")
	}
	if received != "" {
		t.Fatalf("Error: we received something (when execution was supposed to be aborted immediately)")
	}
	t.Logf("good! an error has been received (and it was expected): %s", err)
}

// TestActionList checks that an ActionList respects the order
// of actions.
func TestActionList(t *testing.T) {
	expected := "12345678"

	path := ""
	doRecordPath := func(num string) Action {
		return ActionFunc(func(context.Context) Action {
			path += num
			return nil
		})
	}

	actions := ActionList{
		doRecordPath("1"),
		nil,
		ActionFunc(func(context.Context) Action {
			return ActionList{
				doRecordPath("2"),
				doRecordPath("3"),
				ActionFunc(func(context.Context) Action {
					return ActionList{
						doRecordPath("4"),
						nil,
						doRecordPath("5"),
					}
				}),
			}
		}),
		ActionFunc(func(context.Context) Action {
			return ActionList{
				nil,
				doRecordPath("6"),
				DoIfElse(CheckExpr(true),
					ActionFunc(func(context.Context) Action {
						return doRecordPath("7")
					}),
					ActionFunc(func(context.Context) Action {
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
	ctx := NewTestingContext()
	res := DoSendingExecOutputToWriter(&buf, &actions).Apply(ctx)
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
		return ActionFunc(func(context.Context) Action {
			path += num
			return nil
		})
	}

	expected := "000111222"
	actions := ActionList{
		doRecordPath("000"),
		DoIfElse(CheckExpr(true),
			ActionFunc(func(context.Context) Action {
				return doRecordPath("111")
			}),
			ActionFunc(func(context.Context) Action {
				return doRecordPath("XXX")
			}),
		),
		DoIfElse(CheckExpr(true),
			nil,
			ActionFunc(func(context.Context) Action {
				return doRecordPath("YYY")
			}),
		),
		DoIfElse(CheckExpr(false),
			ActionFunc(func(context.Context) Action {
				return doRecordPath("YYY")
			}),
			nil,
		),
		nil,
		nil,
		doRecordPath("222"),
	}

	var buf bytes.Buffer
	ctx := NewTestingContext()
	res := DoSendingExecOutputToWriter(&buf, &actions).Apply(ctx)
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
		return ActionFunc(func(context.Context) Action {
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
			ActionFunc(func(context.Context) Action {
				return ActionError("some error")
			}),
			// this ActionList is never executed, as the presence of an error makes the whole list errored
			ActionList{
				doRecordPath("XXX"),
				ActionError("some error"),
			},
			ActionFunc(func(context.Context) Action {
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
	ctx := NewTestingContext()
	res := DoSendingExecOutputToWriter(&buf, &actions).Apply(ctx)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if path != expected {
		t.Fatalf("Error: unexpected contents: %q, expected: %q", path, expected)
	}
}

func TestDoRetry(t *testing.T) {
	count := 0
	actions := ActionList{
		DoRetry(Retry{Times: 3, Interval: 100 * time.Millisecond},
			ActionFunc(func(context.Context) Action {
				count += 1
				return ActionError("an error")
			}),
		),
	}

	ctx := NewTestingContext()
	res := actions.Apply(ctx)
	if !IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if count != 3 {
		t.Fatalf("Error: unexpected number of retries: %d, expected: %q", count, 3)
	}
}

func doEcho(msg string) Action {
	return DoLocalExec("/bin/echo", msg)
}
