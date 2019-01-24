package dns

import (
	"log"
	"strings"
)

type DryrunProvider struct {
	Zone map[string][]string
}

func NewDryrunProvider() (*DryrunProvider, error) {
	return &DryrunProvider{Zone: map[string][]string{}}, nil
}

func (provider *DryrunProvider) AddHostnameMapping(hostname string, ip string) error {
	log.Printf("[INFO] Dryrun - Adding mapping: %s\tA\t%s", hostname, ip)
	if len(provider.Zone[hostname]) == 0 {
		provider.Zone[hostname] = []string{ip}
	} else {
		if findIndex(provider.Zone[hostname], ip) == -1 {
			provider.Zone[hostname] = append(provider.Zone[hostname], ip)
		}
	}
	log.Printf("[INFO] Dryrun - Resulting record: %s\tA\t%s", hostname, strings.Join(provider.Zone[hostname], ","))
	return nil
}

func (provider *DryrunProvider) RemoveHostnameMapping(hostname string, ip string) error {
	log.Printf("[INFO] Dryrun - Removing mapping: %s\tA\t%s", hostname, ip)
	mapping := provider.Zone[hostname]
	index := findIndex(mapping, ip)
	if index == -1 {
		// Should never happen
		log.Printf("[WARN] Dryrun - Attemting to remove a non mapped IP")
		return nil
	}
	if len(mapping) == 1 {
		delete(provider.Zone, hostname)
	} else {
		mapping[index] = mapping[len(mapping)-1]
		provider.Zone[hostname] = mapping[:len(mapping)-1]
	}
	log.Printf("[INFO] Dryrun - Resulting record: %s\tA\t%s", hostname, strings.Join(provider.Zone[hostname], ","))
	return nil
}
