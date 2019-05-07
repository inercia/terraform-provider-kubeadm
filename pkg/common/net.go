package common

import (
	"fmt"
	"net"
	"strings"
)

// AddressWithPort return an address as host:port (setting a default port p if there was no port specified)
func AddressWithPort(name string, p int) string {
	if strings.IndexByte(name, ':') < 0 {
		return net.JoinHostPort(name, fmt.Sprintf("%d", p))
	}
	return name
}
