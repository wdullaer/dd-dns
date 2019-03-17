package store

import (
	"github.com/wdullaer/dd-dns/dns"
	"github.com/wdullaer/dd-dns/types"
)

// Store provides methods to interact with the desired state that will be provisioned on the dns.Provider
type Store interface {
	// CleanUp ensures any pending operations on the store are executed before closing down
	CleanUp()
	// InsertMapping registers that the ContainerID of the DNSMapping supports an A record
	// In case the A record is not present in the current state, the callback will be executed
	// which should create it at the DNSProvider
	// TODO: maybe pass a dns.Provider, rather than a generic callback
	InsertMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error
	// RemoveMapping removes the ContainerID from the list backing the A record
	// In case this was the last ContainerID in the list, the callback will be executed
	// to remove the A record from the DNSProvider
	RemoveMapping(mapping *types.DNSMapping, cb func(*types.DNSMapping) error) error
	// ReplaceMappings will replace the current list of DNSMappings with the supplied list
	// It will interact with the dns.Provider to ensure the remote state is in sync
	// It will perform a diff with the current state to minimize the amount of API calls to the dns.Provider
	ReplaceMappings(mappings []*types.DNSMapping, provider dns.Provider) error
}
