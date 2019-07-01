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
