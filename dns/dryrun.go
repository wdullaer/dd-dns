package dns

import (
	"log"
	"strings"

	"github.com/wdullaer/docker-dns-updater/stringslice"
	"github.com/wdullaer/docker-dns-updater/types"
)

type DryrunProvider struct {
	Zone map[string][]string
}

func NewDryrunProvider() (*DryrunProvider, error) {
	return &DryrunProvider{Zone: map[string][]string{}}, nil
}

func (provider *DryrunProvider) AddHostnameMapping(mapping *types.DNSMapping) error {
	log.Printf("[INFO] Dryrun - Adding mapping: %s\tA\t%s", mapping.Name, mapping.IP)
	if len(provider.Zone[mapping.Name]) == 0 {
		provider.Zone[mapping.Name] = []string{mapping.IP}
	} else {
		if stringslice.FindIndex(provider.Zone[mapping.Name], mapping.IP) == -1 {
			provider.Zone[mapping.Name] = append(provider.Zone[mapping.Name], mapping.IP)
		}
	}
	log.Printf("[INFO] Dryrun - Resulting record: %s\tA\t%s", mapping.Name, strings.Join(provider.Zone[mapping.Name], ","))
	return nil
}

func (provider *DryrunProvider) RemoveHostnameMapping(mapping *types.DNSMapping) error {
	log.Printf("[INFO] Dryrun - Removing mapping: %s\tA\t%s", mapping.Name, mapping.IP)
	record := provider.Zone[mapping.Name]
	index := stringslice.FindIndex(record, mapping.IP)
	if index == -1 {
		// Should never happen
		log.Printf("[WARN] Dryrun - Attemting to remove a non mapped IP")
		return nil
	}
	if len(record) == 1 {
		delete(provider.Zone, mapping.Name)
	} else {
		record[index] = record[len(record)-1]
		provider.Zone[mapping.Name] = record[:len(record)-1]
	}
	log.Printf("[INFO] Dryrun - Resulting record: %s\tA\t%s", mapping.Name, strings.Join(provider.Zone[mapping.Name], ","))
	return nil
}
