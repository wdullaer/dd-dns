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
		output     bool
	}{
		{
			name:       "Should return false for an empty slice input",
			inputSlice: []cloudflare.DNSRecord{},
			inputIP:    "127.0.0.1",
			output:     false,
		},
		{
			name:       "Should return false for an empty IP input",
			inputSlice: []cloudflare.DNSRecord{cloudflare.DNSRecord{Content: "127.0.0.1"}},
			inputIP:    "",
			output:     false,
		},
		{
			name: "Should return false if the IP is not part of any DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.2",
			output:  false,
		},
		{
			name: "Should return true if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			output:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := hasRecordForIP(tc.inputSlice, tc.inputIP)
			assert.Equal(t, tc.output, output)
		})
	}
}

func TestFindIndex(t *testing.T) {
	cases := []struct {
		name       string
		inputSlice []cloudflare.DNSRecord
		inputIP    string
		output     int
	}{
		{
			name:       "Should return -1 for an empty slice input",
			inputSlice: []cloudflare.DNSRecord{},
			inputIP:    "127.0.0.1",
			output:     -1,
		},
		{
			name:       "Should return -1 for an empty IP input",
			inputSlice: []cloudflare.DNSRecord{cloudflare.DNSRecord{Content: "127.0.0.1"}},
			inputIP:    "",
			output:     -1,
		},
		{
			name: "Should return -1 if the IP is not part of any DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.2",
			output:  -1,
		},
		{
			name: "Should return the index if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			output:  1,
		},
		{
			name: "Should return the first index if the IP is part of one of the DNSRecords",
			inputSlice: []cloudflare.DNSRecord{
				cloudflare.DNSRecord{Content: "127.0.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
				cloudflare.DNSRecord{Content: "192.168.0.1"},
			},
			inputIP: "192.168.0.1",
			output:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := findRecordIndex(tc.inputSlice, tc.inputIP)
			assert.Equal(t, tc.output, output)
		})
	}
}
