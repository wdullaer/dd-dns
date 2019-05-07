package types

import (
	"net"

	"github.com/google/go-cmp/cmp"
)

// DNSContainerList is a type that keeps track of which containerIDs are associated with a (hostname, IP) pair
type DNSContainerList struct {
	Name          string
	IP            net.IP
	ContainerList []string
}

// DNSMapping is a type that represents a Container and its associated (hostname, IP) pair
type DNSMapping struct {
	Name        string
	IP          net.IP
	ContainerID string
}

// GetKey produces a byte array that can be used as a unique key for this record for us in eg Boltdb
func (mapping *DNSMapping) GetKey() []byte {
	return []byte(mapping.Name + mapping.IP.String())
}

// HasDNSMapping checks if a slice of DNSMapping pointers contains a particular mapping by value
func HasDNSMapping(col []*DNSMapping, item *DNSMapping) bool {
	for i := range col {
		if cmp.Equal(*col[i], *item) {
			return true
		}
	}
	return false
}
