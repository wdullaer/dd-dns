package dns

import (
	"context"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/wdullaer/dd-dns/types"
	"go.uber.org/zap"
)

// CloudflareProvider implements the DNSProvider interface for Cloudflare
type CloudflareProvider struct {
	API    *cloudflare.API
	logger *zap.SugaredLogger
}

// NewCloudflareProvider generates a CloudflareProvider using the given credentials
func NewCloudflareProvider(email string, token string, logger *zap.SugaredLogger) (*CloudflareProvider, error) {
	api, err := cloudflare.New(token, email)
	if err != nil {
		return nil, err
	}
	return &CloudflareProvider{API: api, logger: logger.Named("cloudflare-dns")}, nil
}

// AddHostnameMapping adds the given DNSMapping as an A record
// In case an A record already exists, it will succeed, since the desired state has already been obtained
// It will not modify any records that are not A records.
func (provider *CloudflareProvider) AddHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Adding mapping to DNS", "mapping", mapping)
	zoneName := getZoneName(mapping.Name)

	zoneIDString, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	zoneID := cloudflare.ZoneIdentifier(zoneIDString)
	records, _, err := provider.API.ListDNSRecords(
		context.TODO(),
		zoneID,
		cloudflare.ListDNSRecordsParams{Type: "A", Name: mapping.Name},
	)
	if err != nil {
		return err
	}

	// If there is no remote record for this hostname, we need to create it
	if !hasRecordForIP(records, mapping.IP.String()) {
		dnsRecord := cloudflare.CreateDNSRecordParams{
			Name:    mapping.Name,
			Content: mapping.IP.String(),
			Type:    "A",
		}
		if _, err = provider.API.CreateDNSRecord(
			context.TODO(),
			zoneID,
			dnsRecord,
		); err != nil {
			return err
		}
		return nil
	}

	provider.logger.Warnw("Record already exists for DNSMapping", "dnsMapping", mapping)

	return nil
}

// RemoveHostnameMapping will remove the given DNSMapping from an A record
// In case no A record or no mapping exists, the call will succeed, given that the required has already been achieved
// It will not modify any records that are not A records
func (provider *CloudflareProvider) RemoveHostnameMapping(mapping *types.DNSMapping) error {
	provider.logger.Infow("Removing mapping from DNS", "mapping", mapping)
	zoneName := getZoneName(mapping.Name)

	zoneIDString, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	zoneID := cloudflare.ZoneIdentifier(zoneIDString)
	records, _, err := provider.API.ListDNSRecords(
		context.TODO(),
		zoneID,
		cloudflare.ListDNSRecordsParams{Name: mapping.Name, Type: "A"},
	)
	if err != nil {
		return err
	}

	index := findRecordIndex(records, mapping.IP.String())
	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if index == -1 {
		provider.logger.Warnw("IP is not mapped to hostname ", "mapping", mapping)
		return nil
	}

	return provider.API.DeleteDNSRecord(context.TODO(), zoneID, records[index].ID)
}

// hasRecordForIP returns true if there is at least 1 DNSRecord with the given
// IP as Content in the input slice
func hasRecordForIP(col []cloudflare.DNSRecord, ip string) bool {
	for i := range col {
		if col[i].Content == ip {
			return true
		}
	}
	return false
}

// findIndex returns the index of the first DNSRecord that has a given IP
// Returns -1 if no DNSRecord with the given IP is present in the slice
func findRecordIndex(col []cloudflare.DNSRecord, ip string) int {
	for i := range col {
		if col[i].Content == ip {
			return i
		}
	}
	return -1
}
