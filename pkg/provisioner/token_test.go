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

package provisioner

import (
	"testing"
	"time"
)

func TestGetKubeadmTokensFromString(t *testing.T) {
	s := `
TOKEN                     TTL       EXPIRES                USAGES                   DESCRIPTION   EXTRA GROUPS\r
5befc5.a36864a4c9cc2c7d   22h       2019-07-10T15:08:31Z   authentication,signing   <none>        system:bootstrappers:kubeadm:default-node-token\r
9befc8.a36864a4c9cc2c7d   26h       2039-02-10T12:13:24Z   authentication,signing   <none>        system:bootstrappers:kubeadm:default-node-token\r
\r
`

	testCases := map[string]struct {
		isExpired bool
	}{
		"5befc5.a36864a4c9cc2c7d": {
			true,
		},
		"9befc8.a36864a4c9cc2c7d": {
			false,
		},
	}

	tokens := KubeadmTokensSet{}
	err := tokens.FromString(s)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	now, _ := time.Parse(time.RFC822, "01 Jan 20 20:00 UTC")
	t.Logf("now: %s", now)
	for _, token := range tokens {
		testCase, ok := testCases[token.Token]
		t.Logf("testcase: %+v", testCase)
		t.Logf("token: %+v", token)

		if !ok {
			t.Fatalf("error: token %q not found in tests cases table", token.Token)
		}

		if testCase.isExpired != token.IsExpired(now) {
			t.Fatalf("error: token %q reports as 'expired=%t' but we expected '%t'", token.Token, token.IsExpired(now), testCase.isExpired)
		}
	}
}
