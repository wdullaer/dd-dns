package dns

import (
	"net"
	"strings"

	"go.uber.org/zap"

	"github.com/wdullaer/docker-dns-updater/types"
)

type DryrunProvider struct {
	Zone   map[string][]net.IP
	logger *zap.SugaredLogger
}

func NewDryrunProvider(logger *zap.SugaredLogger) (*DryrunProvider, error) {
	return &DryrunProvider{Zone: map[string][]net.IP{}, logger: logger.Named("dryrun-dns")}, nil
}

func (provider *DryrunProvider) AddHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Adding mapping", "mapping", mapping)
	if len(provider.Zone[mapping.Name]) == 0 {
		provider.Zone[mapping.Name] = []net.IP{mapping.IP}
	} else {
		if findIPIndex(provider.Zone[mapping.Name], mapping.IP) == -1 {
			provider.Zone[mapping.Name] = append(provider.Zone[mapping.Name], mapping.IP)
		}
	}
	provider.logger.Infow("Resulting record", "hostname", mapping.Name, "record", stringify(provider.Zone[mapping.Name]))
	return nil
}

func (provider *DryrunProvider) RemoveHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Removing mapping", "mapping", mapping)
	record := provider.Zone[mapping.Name]
	index := findIPIndex(record, mapping.IP)
	if index == -1 {
		// Should never happen
		provider.logger.Warn("Attemting to remove a non mapped IP")
		return nil
	}
	if len(record) == 1 {
		delete(provider.Zone, mapping.Name)
	} else {
		record[index] = record[len(record)-1]
		provider.Zone[mapping.Name] = record[:len(record)-1]
	}
	provider.logger.Infow("Resulting record", "hostname", mapping.Name, "record", stringify(provider.Zone[mapping.Name]))
	return nil
}

// findIPIndex returns the index of a particular IP in an IP slice.
// Returns -1 if the IP is not present in the slice
// Who needs generics, implementing the same function 100x is fun!
func findIPIndex(col []net.IP, item net.IP) int {
	for i := range col {
		if col[i].Equal(item) {
			return i
		}
	}
	return -1
}

// stringify returns a string representation of a slice of net.IP
// Useful for printing log statements
func stringify(col []net.IP) string {
	s := make([]string, len(col))
	for i := range col {
		s[i] = col[i].String()
	}
	return strings.Join(s, ",")
}
