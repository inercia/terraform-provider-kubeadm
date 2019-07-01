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

func TestStringSliceUnique(t *testing.T) {

	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i, v := range a {
			if v != b[i] {
				return false
			}
		}
		return true
	}

	testsCases := []struct {
		input    []string
		expected []string
	}{
		{
			[]string{"hello", "world"},
			[]string{"hello", "world"},
		},
		{
			[]string{"hello", "hello", "world"},
			[]string{"hello", "world"},
		},
		{
			[]string{"hello", "hello", "world", "world"},
			[]string{"hello", "world"},
		},
	}

	for _, testCase := range testsCases {
		out := StringSliceUnique(testCase.input)
		if !equal(testCase.expected, out) {
			t.Fatalf("Error: expected output does not match: %q != %q", out, testCase.expected)
		}
	}
}
