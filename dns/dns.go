package dns

import (
	"strings"

	"github.com/wdullaer/dd-dns/types"
)

// Provider provides a common abstraction over the APIs of various DNS services
type Provider interface {
	AddHostnameMapping(mapping *types.DNSMapping) error
	RemoveHostnameMapping(mapping *types.DNSMapping) error
}

func getZoneName(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return hostname
	}
	parts = parts[len(parts)-2:]
	return strings.Join(parts, ".")
}
