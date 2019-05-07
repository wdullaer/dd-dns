package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetZoneName(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expected string
	}{
		{
			name:   "Should return the input if it already is a top level zone",
			input:  "example.com",
			expected: "example.com",
		},
		{
			name:   "Should return the top level zone for a subdomain",
			input:  "foo.example.com",
			expected: "example.com",
		},
		{
			name:   "Should return the top level for a deeply nested subdomain",
			input:  "test-domain.whatever.foo.me",
			expected: "foo.me",
		},
		{
			name:   "Should return the input for a single word",
			input:  "home",
			expected: "home",
		},
		{
			name:   "Should work with the empty string input",
			input:  "",
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := getZoneName(tc.input)
			assert.Equal(t, tc.expected, output)
		})
	}
}
