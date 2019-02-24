package main

import "testing"

func TestConfigValidate(t *testing.T) {
	// Should set default values
	input := config{}
	errs := input.Validate()

	if len(errs) != 0 {
		t.Log("Expected empty config not to have validation errors")
		t.Fail()
	}
	if input.Provider == "" {
		t.Log("Provider should have a default value")
		t.Fail()
	}
	if input.DNSContent == "" {
		t.Log("DNSContent should have a default value")
		t.Fail()
	}
	if input.DockerLabel == "" {
		t.Log("DockerLabel should have a default value")
		t.Fail()
	}
	if input.Store == "" {
		t.Log("Store should have a default value")
		t.Fail()
	}

	// Should return all errors
	input = config{
		Store:    "notvalid",
		Provider: "notvalid",
	}
	errs = input.Validate()
	if len(errs) != 2 {
		t.Logf("Expected Validate to return 2 errors, recieved %d", len(errs))
		t.Fail()
	}
}

func TestValidateProvider(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "cloudflare",
			error:    false,
		},
		{
			input:    "cloudflare",
			expected: "cloudflare",
			error:    false,
		},
		{
			input:    "cLoUdFlaRe",
			expected: "cloudflare",
			error:    false,
		},
		{
			input:    "   cloudFlare\t",
			expected: "cloudflare",
			error:    false,
		},
		{
			input:    "dryrun",
			expected: "dryrun",
			error:    false,
		},
		{
			input:    "my-dns-provider",
			expected: "",
			error:    true,
		},
	}

	for _, tc := range cases {
		output, err := validateProvider(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateInput` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}

func TestValidateAccountName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "",
			error:    false,
		},
		{
			input:    "foO",
			expected: "foO",
			error:    false,
		},
	}

	for _, tc := range cases {
		output, err := validateAccountName(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateAccountName` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}

func TestValidateAccountSecret(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "",
			error:    false,
		},
		{
			input:    "foO",
			expected: "foO",
			error:    false,
		},
	}

	for _, tc := range cases {
		output, err := validateAccountSecret(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateAccountSecret` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}

func TestValidateDNSContent(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "container",
			error:    false,
		},
		{
			input:    "container",
			expected: "container",
			error:    false,
		},
		{
			input:    "cOnTaInEr",
			expected: "container",
			error:    false,
		},
		{
			input:    "  container\t",
			expected: "container",
			error:    false,
		},
		{
			input:    "192.168.0.1",
			expected: "192.168.0.1",
			error:    false,
		},
		{
			input:    "foobar",
			expected: "",
			error:    true,
		},
		// TODO: Add an IPv6 case
	}

	for _, tc := range cases {
		output, err := validateDNSContent(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateDNSContent` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}

func TestValidateDockerLabel(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "caddy.address",
			error:    false,
		},
		{
			input:    "example.com",
			expected: "example.com",
			error:    false,
		},
		{
			input:    "eXaMple.com",
			expected: "example.com",
			error:    false,
		},
		{
			input:    "  example.com\t",
			expected: "example.com",
			error:    false,
		},
	}

	for _, tc := range cases {
		output, err := validateDockerLabel(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateDockerLabel` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}

func TestValidateStore(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		error    bool
	}{
		{
			input:    "",
			expected: "memory",
			error:    false,
		},
		{
			input:    "memory",
			expected: "memory",
			error:    false,
		},
		{
			input:    "mEmOry",
			expected: "memory",
			error:    false,
		},
		{
			input:    "  memory\t",
			expected: "memory",
			error:    false,
		},
		{
			input:    "boltdb",
			expected: "boltdb",
			error:    false,
		},
		{
			input:    "foo",
			expected: "",
			error:    true,
		},
	}

	for _, tc := range cases {
		output, err := validateStore(tc.input)
		if (err != nil) != tc.error {
			t.Logf("Expected `validateStore` with input `%s` to not return an error", tc.input)
			t.Fail()
		}
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}
