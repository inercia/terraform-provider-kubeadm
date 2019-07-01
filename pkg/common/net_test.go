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

package common

import (
	"testing"
)

func TestSplitHostPort(t *testing.T) {

	testsCases := []struct {
		addr         string
		defPort      int
		expectedHost string
		expectedPort int
	}{
		{
			"some.place:4545",
			0,
			"some.place",
			4545,
		},
		{
			"some.place",
			25,
			"some.place",
			25,
		},
		{
			"some.place:2525",
			8080,
			"some.place",
			2525,
		},
	}

	for _, testCase := range testsCases {
		h, p, err := SplitHostPort(testCase.addr, testCase.defPort)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if h != testCase.expectedHost {
			t.Fatalf("Error: expectedHost does not match: %q != %q", h, testCase.expectedHost)
		}
		if p != testCase.expectedPort {
			t.Fatalf("Error: expectedPort does not match: %q != %q", p, testCase.expectedPort)
		}
	}
}
