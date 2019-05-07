package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	t.Run("Should set default values", func(t *testing.T) {
		input := config{}
		if assert.Empty(t, input.Validate(), "Expected empty config not to have validation errors") {
			assert.NotEmpty(t, input.Provider, "Provider should have a default value")
			assert.NotEmpty(t, input.DNSContent, "DNSContent should have a default value")
			assert.NotEmpty(t, input.DockerLabel, "DockerLabel should have a default value")
			assert.NotEmpty(t, input.Store, "Store should have a default value")
			assert.NotEmpty(t, input.DataDirectory, "DataDirectory should have a default value")
		}
	})

	t.Run("Should return all errors", func(t *testing.T) {
		input := config{
			Store:    "notvalid",
			Provider: "notvalid",
		}
		errs := input.Validate()
		assert.Equal(t, 2, len(errs), "Expected validate to receive 2 errors")
	})
}

func TestValidateProvider(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should set a default value of `cloudflare`",
			input:    "",
			expected: "cloudflare",
			error:    false,
		},
		{
			name:     "Should pass on a valid input",
			input:    "cloudflare",
			expected: "cloudflare",
			error:    false,
		},
		{
			name:     "Should lowercase a valid input",
			input:    "cLoUdFlaRe",
			expected: "cloudflare",
			error:    false,
		},
		{
			name:     "Should trim excess whitespace of a valid input",
			input:    "   cloudFlare\t",
			expected: "cloudflare",
			error:    false,
		},
		{
			name:     "Should accept `dryrun` as a valid input",
			input:    "dryrun",
			expected: "dryrun",
			error:    false,
		},
		{
			name:     "Should return an error for an invalid input",
			input:    "my-dns-provider",
			expected: "",
			error:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateProvider(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateProvider` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateProvider` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateAccountName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Shoud allow an empty input",
			input:    "",
			expected: "",
			error:    false,
		},
		{
			name:     "Should pass on any non-empty input",
			input:    "foO",
			expected: "foO",
			error:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateAccountName(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateAccountName` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateAccountName` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateAccountSecret(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should allow an empty input",
			input:    "",
			expected: "",
			error:    false,
		},
		{
			name:     "Should pass on any non-empty input",
			input:    "foO",
			expected: "foO",
			error:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateAccountSecret(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateAccountSecret` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateAccountSecret` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateDNSContent(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should set a default value of `container`",
			input:    "",
			expected: "container",
			error:    false,
		},
		{
			name:     "Should pass on an input of `container`",
			input:    "container",
			expected: "container",
			error:    false,
		},
		{
			name:     "Should lowercase a valid input",
			input:    "cOnTaInEr",
			expected: "container",
			error:    false,
		},
		{
			name:     "Should trim whitespace off a valid input",
			input:    "  container\t",
			expected: "container",
			error:    false,
		},
		{
			name:     "Should pass on a v4 IP address",
			input:    "192.168.0.1",
			expected: "192.168.0.1",
			error:    false,
		},
		{
			name:     "Should reject an invalid input",
			input:    "foobar",
			expected: "",
			error:    true,
		},
		// TODO: Add an IPv6 case
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateDNSContent(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateDNSContent` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateDNSContent` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateDockerLabel(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should set a default of caddy.address",
			input:    "",
			expected: "caddy.address",
			error:    false,
		},
		{
			name:     "Should pass on any valid input",
			input:    "example.com",
			expected: "example.com",
			error:    false,
		},
		{
			name:     "Should lowercase the input",
			input:    "eXaMple.com",
			expected: "example.com",
			error:    false,
		},
		{
			name:     "Should trim whitespace of an input",
			input:    "  example.com\t",
			expected: "example.com",
			error:    false,
		},
		// TODO: Add case that rejects whitespace in the middle
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateDockerLabel(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateDockerLabel` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateDockerLabel` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateStore(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should set a default value of `memory`",
			input:    "",
			expected: "memory",
			error:    false,
		},
		{
			name:     "Should pass on an input of `memory`",
			input:    "memory",
			expected: "memory",
			error:    false,
		},
		{
			name:     "Should lowercase a valid input",
			input:    "mEmOry",
			expected: "memory",
			error:    false,
		},
		{
			name:     "Should trim whitespace off a valid input",
			input:    "  memory\t",
			expected: "memory",
			error:    false,
		},
		{
			name:     "Should pass on an input of `boltdb`",
			input:    "boltdb",
			expected: "boltdb",
			error:    false,
		},
		{
			name:     "Should reject an invalid input",
			input:    "foo",
			expected: "",
			error:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateStore(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateStore` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateStore` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestValidateDataDirectory(t *testing.T) {
	defaultDirectory, _ := os.Getwd()
	cases := []struct {
		name     string
		input    string
		expected string
		error    bool
	}{
		{
			name:     "Should set a default value",
			input:    "",
			expected: defaultDirectory,
			error:    false,
		},
		{
			name:     "Should pass on any non-empty input",
			input:    "foo",
			expected: "foo",
			error:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateDataDirectory(tc.input)
			if tc.error {
				assert.Errorf(t, err, "Expected `validateDataDirectory` with input `%s` to return an error", tc.input)
			} else {
				assert.NoErrorf(t, err, "Expected `validateDataDirectory` with input `%s` to not return an error", tc.input)
			}
			assert.Equal(t, tc.expected, output)
		})
	}
}
