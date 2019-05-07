package dns

import (
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

func TestHasRecordForIP(t *testing.T) {
	cases := []struct {
		name       string
		inputSlice []cloudflare.DNSRecord
		inputIP    string
		expected     bool
	}{
		{
			name:       "Should return false for an empty slice input",
			inputSlice: []cloudflare.DNSRecord{},
			inputIP:    "127.0.0.1",
			expected:     false,
		},
		{
			name: "Should return false if the IP is not part of any DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.2",
			expected:  false,
		},
		{
			name: "Should return true if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			expected:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := hasRecordForIP(tc.inputSlice, tc.inputIP)
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestFindIndex(t *testing.T) {
	cases := []struct {
		name       string
		inputSlice []cloudflare.DNSRecord
		inputIP    string
		expected     int
	}{
		{
			name:       "Should return -1 for an empty slice input",
			inputSlice: []cloudflare.DNSRecord{},
			inputIP:    "127.0.0.1",
			expected:     -1,
		},
		{
			name: "Should return -1 if the IP is not part of any DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.2",
			expected:  -1,
		},
		{
			name: "Should return the index if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			expected:  1,
		},
		{
			name: "Should return the first index if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			expected:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := findRecordIndex(tc.inputSlice, tc.inputIP)
			assert.Equal(t, tc.expected, output)
		})
	}
}
