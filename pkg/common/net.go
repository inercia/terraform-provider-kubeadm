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
