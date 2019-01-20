package main

import (
	"fmt"
	"net"
	"strings"
)

type config struct {
	Provider      string
	AccountName   string
	AccountSecret string
	DNSContent    string
	DockerLabel   string
}

func (c *config) Validate() (*config, error) {
	// TODO: Implement validation
	return c, nil
}

func validateProvider(provider string) (string, error) {
	switch sanitize(provider) {
	case "":
		return "cloudflare", nil
	case "cloudflare":
		return "cloudflare", nil
	default:
		return "", fmt.Errorf("Invalid provider `%s` specified. Available providers: [`cloudflare`]", provider)
	}
}

func validateAccountName(accountName string) (string, error) {
	return accountName, nil
}

func validateAccountSecret(accountSecret string) (string, error) {
	return accountSecret, nil
}

func validateDNSContent(dnsContent string) (string, error) {
	dnsContent = sanitize(dnsContent)
	switch dnsContent {
	case "":
		return "host", nil
	case "container":
		return "container", nil
	default:
		ip := net.IP(dnsContent)
		if ip == nil {
			return "", fmt.Errorf("Invalid dns-content specified. %s must be a valid IPv4 address or one of [`host`, `container`]")
		}
		ip = ip.To4()
		// TODO: remove this check when we add IPv6 support. We might want to split this config variable in 2 when we do (MODE and actual IP)
		if ip == nil {
			return "", fmt.Errorf("Invalid dns-content specified. %s must be a valid IPv4 address or one of [`host`, `container`]")
		}
		return ip.String(), nil
	}
}

func validateDockerLabel(dockerLabel string) (string, error) {
	dockerLabel = sanitize(dockerLabel)
	if dockerLabel == "" {
		return "caddy.address", nil
	}
	return dockerLabel, nil
}

func sanitize(value string) string {
	return strings.Trim(strings.ToLower(value), " \t")
}
