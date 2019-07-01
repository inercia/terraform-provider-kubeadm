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
