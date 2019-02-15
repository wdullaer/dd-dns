package store

import (
	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/types"
)

type Store interface {
	CleanUp()
	// TODO: make the callback take a DNSMapping
	InsertMapping(mapping *types.DNSMapping, cb func(string, string) error) error
	// TODO: make the callback take a DNSMapping
	RemoveMapping(mapping *types.DNSMapping, cb func(string, string) error) error
	// Replaces all of the DNS mappings with the ones passed to this method
	// The Store will try to minimize the amount of calls it makes to the provider
	// by diffing its current state with the required state
	ReplaceMappings(mappings []*types.DNSMapping, provider dns.DNSProvider) error
}
