package store

import (
	"github.com/wdullaer/dd-dns/dns"
	"github.com/wdullaer/dd-dns/types"
)

type Store interface {
	CleanUp()
	InsertMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error
	RemoveMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error
	// Replaces all of the DNS mappings with the ones passed to this method
	// The Store will try to minimize the amount of calls it makes to the provider
	// by diffing its current state with the required state
	ReplaceMappings(mappings []*types.DNSMapping, provider dns.Provider) error
}
