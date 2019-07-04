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
	"testing"
)

func TestTempFilenames(t *testing.T) {
	name1, err := GetTempFilename()
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	name2, err := GetTempFilename()
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	names := []struct {
		name string
		res  bool
	}{
		{
			name: name1,
			res:  true,
		},
		{
			name: name2,
			res:  true,
		},
		{
			name: "/tmp/something.tmp",
			res:  false,
		},
	}

	for _, testCase := range names {
		isTemp := IsTempFilename(testCase.name)
		if isTemp != testCase.res {
			t.Fatalf("Error: %q detected as temp=%t when we expected temp=%t", testCase.name, isTemp, testCase.res)
		}
	}
}
