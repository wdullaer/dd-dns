package dns

import (
	"strings"

	"github.com/wdullaer/dd-dns/types"
)

type DNSProvider interface {
	AddHostnameMapping(mapping *types.DNSMapping) error
	RemoveHostnameMapping(mapping *types.DNSMapping) error
}

func getZoneName(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return hostname
	}
	parts = parts[len(parts)-2 : len(parts)]
	return strings.Join(parts, ".")
}
