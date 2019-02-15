package dns

import (
	"log"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/wdullaer/docker-dns-updater/stringslice"
	"github.com/wdullaer/docker-dns-updater/types"
)

type CloudflareProvider struct {
	API *cloudflare.API
	// TODO: Maybe add private variable with local cache of remote table
}

const cloudflareEndpoint = "https://api.cloudflare.com/client/v4/"

func NewCloudflareProvider(email string, token string) (*CloudflareProvider, error) {
	api, err := cloudflare.New(token, email)
	if err != nil {
		return nil, err
	}
	return &CloudflareProvider{API: api}, nil
}

func (provider *CloudflareProvider) AddHostnameMapping(mapping *types.DNSMapping) error {
	zoneName := getZoneName(mapping.Name)

	zoneID, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	records, err := provider.API.DNSRecords(zoneID, cloudflare.DNSRecord{Name: mapping.Name, Type: "A"})
	if err != nil {
		return err
	}

	// If there is no remote record for this hostname, we need to create it
	if len(records) == 0 {
		dnsRecord := cloudflare.DNSRecord{
			Name:    mapping.Name,
			Content: mapping.IP,
			Type:    "A",
		}
		if _, err = provider.API.CreateDNSRecord(zoneID, dnsRecord); err != nil {
			return err
		}
		return nil
	}

	// There should only be one entry for the hostname, we should update it
	record := records[0]
	currentIPs := strings.Split(record.Content, ",")

	for i := range currentIPs {
		if currentIPs[i] == mapping.IP {
			return nil
		}
	}
	record.Content = strings.Join(append(currentIPs, mapping.IP), ",")

	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}

func (provider *CloudflareProvider) RemoveHostnameMapping(mapping *types.DNSMapping) error {
	zoneName := getZoneName(mapping.Name)

	zoneID, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	records, err := provider.API.DNSRecords(zoneID, cloudflare.DNSRecord{Name: mapping.Name, Type: "A"})
	if err != nil {
		return err
	}

	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if len(records) == 0 {
		log.Printf("[WARN] No records exist for hostname %s.", mapping.Name)
		return nil
	}

	record := records[0]
	currentIPs := strings.Split(record.Content, ",")

	index := stringslice.FindIndex(currentIPs, mapping.IP)
	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if index == -1 {
		log.Printf("[WARN] IP %s is not mapped to hostname %s.", mapping.IP, mapping.Name)
		return nil
	}

	currentIPs[index] = currentIPs[len(currentIPs)-1]
	record.Content = strings.Join(currentIPs[:len(currentIPs)-1], ",")
	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}
