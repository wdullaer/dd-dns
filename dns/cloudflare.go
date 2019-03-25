package dns

import (
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/wdullaer/dd-dns/stringslice"
	"github.com/wdullaer/dd-dns/types"
	"go.uber.org/zap"
)

// CloudflareProvider implements the DNSProvider interface for Cloudflare
type CloudflareProvider struct {
	API    *cloudflare.API
	logger *zap.SugaredLogger
}

const cloudflareEndpoint = "https://api.cloudflare.com/client/v4/"

// NewCloudflareProvider generates a CloudflareProvider using the given credentials
func NewCloudflareProvider(email string, token string, logger *zap.SugaredLogger) (*CloudflareProvider, error) {
	api, err := cloudflare.New(token, email)
	if err != nil {
		return nil, err
	}
	return &CloudflareProvider{API: api, logger: logger.Named("cloudflare-dns")}, nil
}

// AddHostnameMapping adds the given DNSMapping to an A record
// In case an A record already exists, it will append the mapping, trying to keep the current information intact
// It will not modify any records that are not A records.
func (provider *CloudflareProvider) AddHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Adding mapping to DNS", "mapping", mapping)
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
			Content: mapping.IP.String(),
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
		if currentIPs[i] == mapping.IP.String() {
			return nil
		}
	}
	record.Content = strings.Join(append(currentIPs, mapping.IP.String()), ",")

	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}

// RemoveHostnameMapping will remove the given DNSMapping from an A record
// In case no A record or no mapping exists, the call will succeed, given that the required has already been achieved
// It will not modify any records that are not A records
func (provider *CloudflareProvider) RemoveHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Removing mapping from DNS", "mapping", mapping)
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
		provider.logger.Warnw("No records exist for hostname", "hostname", mapping.Name)
		return nil
	}

	record := records[0]
	currentIPs := strings.Split(record.Content, ",")

	index := stringslice.FindIndex(currentIPs, mapping.IP.String())
	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if index == -1 {
		provider.logger.Warnw("IP is not mapped to hostname ", "mapping", mapping)
		return nil
	}

	currentIPs[index] = currentIPs[len(currentIPs)-1]
	record.Content = strings.Join(currentIPs[:len(currentIPs)-1], ",")
	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}
