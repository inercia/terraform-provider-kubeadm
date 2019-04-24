package kubeadm

import (
	"fmt"
	"net"
	"strings"
)

// return an address as host:port (setting a default port p if there was no port specified)
func addressWithPort(name string, p int) string {
	if strings.IndexByte(name, ':') < 0 {
		return net.JoinHostPort(name, fmt.Sprintf("%d", p))
	}
	return name
}
