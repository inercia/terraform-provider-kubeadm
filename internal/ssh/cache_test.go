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
	"context"
	"testing"
)

func TestDoOnce(t *testing.T) {
	count := 0
	inc := ActionFunc(func(context.Context) Action {
		t.Log("incrementing the counter...")
		count++
		return nil
	})

	actions := ActionList{
		DoOnce("increment", inc),
		DoOnce("increment", inc),
		DoOnce("increment", inc),
		DoOnce("increment", inc),
	}

	ctx := NewTestingContext()
	res := actions.Apply(ctx)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if count != 1 {
		t.Fatalf("Error: unexpected number of increments: %d, expected: %d", count, 1)
	}
}

func TestDoOnceWithError(t *testing.T) {
	count := 0
	failed := 0

	inc := ActionFunc(func(context.Context) Action {
		t.Log("incrementing the counter...")
		count++
		return nil
	})

	incFailed := ActionFunc(func(context.Context) Action {
		t.Log("returning an error")
		failed++
		return ActionError("failed to increase the counter")
	})

	ctx := NewTestingContext()

	actions := ActionList{
		// failed actions do not store anything on the cache
		DoOnce("increment", incFailed),
		// ... thihs shoult not be run: the previous error was returned
		DoOnce("increment", inc),
	}
	res := actions.Apply(ctx)
	if !IsError(res) {
		t.Fatalf("Error: no error detected: %s", res)
	}
	if failed != 1 {
		t.Fatalf("Error: unexpected number of failed: %d, expected: %d", failed, 1)
	}
	if count != 0 {
		t.Fatalf("Error: unexpected number of increments: %d, expected: %d", count, 0)
	}

	count = 0
	failed = 0
	actions = ActionList{
		// ... then we run once
		DoOnce("increment", inc),
		// ... and these actions should not be run
		DoOnce("increment", inc),
		DoOnce("increment", inc),
	}
	res = actions.Apply(ctx)
	if IsError(res) {
		t.Fatalf("Error: error detected: %s", res)
	}
	if count != 1 {
		t.Fatalf("Error: unexpected number of increments: %d, expected: %d", count, 1)
	}
}
