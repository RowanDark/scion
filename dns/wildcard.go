package dns

import (
	"fmt"
	"math/rand"
	"net"
)

// DetectWildcard checks if domain has wildcard DNS by resolving a random subdomain.
func DetectWildcard(domain string) (bool, error) {
	probe := fmt.Sprintf("scion-wildcard-check-%d.%s", rand.Int63(), domain)
	addrs, err := net.LookupHost(probe)
	if err != nil {
		// NXDOMAIN or timeout — no wildcard
		return false, nil
	}
	if len(addrs) > 0 {
		return true, nil
	}
	return false, nil
}
