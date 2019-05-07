package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
)

type config struct {
	Provider      string `json:"provider"`
	AccountName   string `json:"account-name"`
	AccountSecret string `json:"account-secret"`
	DNSContent    string `json:"dns-content"`
	DockerLabel   string `json:"docker-label"`
	Store         string `json:"store"`
	DebugLogger   bool   `json:"debug-logger"`
	DataDirectory string `json:"data-directory"`
	// TODO: Add config entry for default docker network to use when DNSContent is container
}

func (c *config) String() string {
	return fmt.Sprintf(
		"{\"provider\": \"%s\", \"account-name\": \"%s\", \"account-secret\": \"%s\", \"dns-content\": \"%s\", \"dns-label\": \"%s\", \"store\": \"%s\", \"debug-logger\": \"%t\", \"data-directory\": \"%s\"}",
		c.Provider,
		c.AccountName,
		"****",
		c.DNSContent,
		c.DockerLabel,
		c.Store,
		c.DebugLogger,
		c.DataDirectory,
	)
}
func (c *config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("provider", c.Provider)
	enc.AddString("account-name", c.AccountName)
	enc.AddString("account-secret", "****")
	enc.AddString("dns-content", c.DNSContent)
	enc.AddString("docker-label", c.DockerLabel)
	enc.AddString("store", c.Store)
	enc.AddBool("debug-logger", c.DebugLogger)
	enc.AddString("data-directory", c.DataDirectory)
	return nil
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
	if value, err := validateStore(c.Store); err != nil {
		errs = append(errs, err)
	} else {
		c.Store = value
	}
	if value, err := validateDataDirectory(c.DataDirectory); err != nil {
		errs = append(errs, err)
	} else {
		c.DataDirectory = value
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
		ip := net.ParseIP(dnsContent)
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
		return "dd-dns.hostname", nil
	}
	return dockerLabel, nil
}

// validateStore normalizes Store and checks that it is part of the list of allowable values
func validateStore(store string) (string, error) {
	switch sanitize(store) {
	case "memory":
		return "memory", nil
	case "boltdb":
		return "boltdb", nil
	case "":
		return "memory", nil
	default:
		return "", fmt.Errorf("Invalid store `%s` provided. Available store implementations: [`memory`, `boltdb`]", store)
	}
}

// validateDataDirectory sets a default, any other value is valid
func validateDataDirectory(directory string) (string, error) {
	if directory == "" {
		return os.Getwd()
	}
	// Further validation of the data directory will happen implicitly
	// when Boltdb tries to access or create its data file
	// This value should be ignored if memorydb is used
	return directory, nil
}

func sanitize(value string) string {
	return strings.Trim(strings.ToLower(value), " \t")
}
