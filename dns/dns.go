package dns

import (
	"strings"
)

type DNSProvider interface {
	AddHostnameMapping(hostname string, ip string) error
	RemoveHostnameMapping(hostname string, ip string) error
}

func getZoneName(hostname string) string {
	parts := strings.Split(hostname, ".")
	parts = parts[len(parts)-2 : len(parts)]
	return strings.Join(parts, ".")
}

func findIndex(col []string, item string) int {
	for i := range col {
		if col[i] == item {
			return i
		}
	}
	return -1
}
