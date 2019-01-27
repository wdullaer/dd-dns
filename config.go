package main

import (
	"fmt"
	"net"
	"strings"
)

type config struct {
	Provider      string `json:"provider"`
	AccountName   string `json:"account-name"`
	AccountSecret string `json:"account-secret"`
	DNSContent    string `json:"dns-content"`
	DockerLabel   string `json:"docker-label"`
	// TODO: Add config entry for default docker network to use when DNSContent is container
}

func (c *config) String() string {
	return fmt.Sprintf(
		"{\"provider\": \"%s\", \"account-name\": \"%s\", \"account-secret\": \"%s\", \"dns-content\": \"%s\", \"dns-label\": \"%s\"}",
		c.Provider,
		c.AccountName,
		"****",
		c.DNSContent,
		c.DockerLabel,
	)
}

// Validate checks each Property of config and provides default values
func (c *config) Validate() []error {
	var errs []error
	if value, err := validateProvider(c.Provider); err != nil {
		errs = append(errs, err)
	} else {
		c.Provider = value
	}
	if value, err := validateAccountName(c.AccountName); err != nil {
		errs = append(errs, err)
	} else {
		c.AccountName = value
	}
	if value, err := validateAccountSecret(c.AccountSecret); err != nil {
		errs = append(errs, err)
	} else {
		c.AccountSecret = value
	}
	if value, err := validateDNSContent(c.DNSContent); err != nil {
		errs = append(errs, err)
	} else {
		c.DNSContent = value
	}
	if value, err := validateDockerLabel(c.DockerLabel); err != nil {
		errs = append(errs, err)
	} else {
		c.DockerLabel = value
	}
	return errs
}

// validateProvider normalizes Provider and checks that it is part of the list of allowable values
func validateProvider(provider string) (string, error) {
	switch sanitize(provider) {
	case "":
		return "cloudflare", nil
	case "cloudflare":
		return "cloudflare", nil
	case "dryrun":
		return "dryrun", nil
	default:
		return "", fmt.Errorf("Invalid provider `%s` specified. Available providers: [`cloudflare`, `dryrun`]", provider)
	}
}

// validateAccountName is a noop, any string passes
func validateAccountName(accountName string) (string, error) {
	return accountName, nil
}

// validateAccountSecret is a noop, any string passes
func validateAccountSecret(accountSecret string) (string, error) {
	return accountSecret, nil
}

// validateDNSContent normalizes DNSContent and checks if it's an IPv4 or part of a list of allowable values
func validateDNSContent(dnsContent string) (string, error) {
	dnsContent = sanitize(dnsContent)
	switch dnsContent {
	case "":
		return "container", nil
	case "container":
		return "container", nil
	default:
		ip := net.IP(dnsContent)
		if ip == nil {
			return "", fmt.Errorf("Invalid dns-content specified. `%s` must be a valid IPv4 address or one of `container`", dnsContent)
		}
		ip = ip.To4()
		// TODO: remove this check when we add IPv6 support. We might want to split this config variable in 2 when we do (MODE and actual IP)
		if ip == nil {
			return "", fmt.Errorf("Invalid dns-content specified. `%s` must be a valid IPv4 address or one of `container`", dnsContent)
		}
		return ip.String(), nil
	}
}

// validateDockerLabel sets a default, any string is valid
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
