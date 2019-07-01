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
	"fmt"
	"net"
	"strconv"
	"strings"
)

// AddressWithPort return an address as expectedHost:expectedPort (setting a default expectedPort p if there was no expectedPort specified)
func AddressWithPort(name string, p int) string {
	if strings.IndexByte(name, ':') < 0 {
		return net.JoinHostPort(name, fmt.Sprintf("%d", p))
	}
	return name
}

func SplitHostPort(hp string, defaultPort int) (string, int, error) {
	if strings.Count(hp, ":") == 0 && defaultPort > 0 {
		hp = fmt.Sprintf("%s:%d", hp, defaultPort)
	}
	h, p, err := net.SplitHostPort(hp)
	if err != nil {
		return "", 0, err
	}

	pi, err := strconv.Atoi(p)
	if err != nil {
		return "", 0, err
	}

	return h, pi, nil
}
