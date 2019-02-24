package types

import (
	"net"
	"testing"
)

func TestGetKey(t *testing.T) {
	// TODO: find a generative testing framework for go, we want to test properties here
	cases := []struct {
		input1 DNSMapping
		input2 DNSMapping
		equals bool
	}{
		{
			// Should return equal key for equal (hostname, IP) pairs
			input1: DNSMapping{Name: "foo", IP: net.ParseIP("192.168.0.1")},
			input2: DNSMapping{Name: "foo", IP: net.ParseIP("192.168.0.1")},
			equals: true,
		},
		{
			// Should return different key for different (hostname, IP) pairs
			input1: DNSMapping{Name: "foo", IP: net.ParseIP("192.168.0.1")},
			input2: DNSMapping{Name: "bar", IP: net.ParseIP("192.168.0.1")},
			equals: false,
		},
		{
			// Should ignore ContainerID length
			input1: DNSMapping{Name: "foo", IP: net.ParseIP("192.168.0.1"), ContainerID: "foo"},
			input2: DNSMapping{Name: "foo", IP: net.ParseIP("192.168.0.1"), ContainerID: "bar"},
			equals: true,
		},
	}

	for _, tc := range cases {
		key1 := tc.input1.GetKey()
		key2 := tc.input2.GetKey()
		if (string(key1) == string(key2)) != tc.equals {
			t.Logf("Expected `%s` t` equal `%s`", key1, key2)
			t.Fail()
		}
	}
}
