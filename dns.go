package main

import (
	"log"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type DNSProvider interface {
	AddHostnameMapping(hostname string, ip string) error
	RemoveHostnameMapping(hostname string, ip string) error
}

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

func (provider *CloudflareProvider) AddHostnameMapping(hostname string, ip string) error {
	zoneName := getZoneName(hostname)

	zoneID, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	records, err := provider.API.DNSRecords(zoneID, cloudflare.DNSRecord{Name: hostname, Type: "A"})
	if err != nil {
		return err
	}

	// If there is no remote record for this hostname, we need to create it
	if len(records) == 0 {
		dnsRecord := cloudflare.DNSRecord{
			Name:    hostname,
			Content: ip,
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
		if currentIPs[i] == ip {
			return nil
		}
	}
	record.Content = strings.Join(append(currentIPs, ip), ",")

	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}

func (provider *CloudflareProvider) RemoveHostnameMapping(hostname string, ip string) error {
	zoneName := getZoneName(hostname)

	zoneID, err := provider.API.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	records, err := provider.API.DNSRecords(zoneID, cloudflare.DNSRecord{Name: hostname, Type: "A"})
	if err != nil {
		return err
	}

	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if len(records) == 0 {
		log.Printf("[WARN] No records exist for hostname %s.", hostname)
		return nil
	}

	record := records[0]
	currentIPs := strings.Split(record.Content, ",")

	index := findIndex(currentIPs, ip)
	// This shouldn't happen, but it's not lethal, so log a warning and continue
	if index == -1 {
		log.Printf("[WARN] IP %s is not mapped to hostname %s.", ip, hostname)
		return nil
	}

	currentIPs[index] = currentIPs[len(currentIPs)-1]
	record.Content = strings.Join(currentIPs[:len(currentIPs)-1], ",")
	if err = provider.API.UpdateDNSRecord(zoneID, record.ID, record); err != nil {
		return err
	}

	return nil
}

func getZoneName(hostname string) string {
	parts := strings.Split(hostname, ".")
	parts = parts[len(parts)-2 : len(parts)]
	return strings.Join(parts, ".")
}

func findIndex(col []string, item string) int {
	for i := range col {
		if col[i] == item {
			return i
		}
	}
	return -1
}
