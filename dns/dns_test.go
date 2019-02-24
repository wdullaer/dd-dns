package dns

import "testing"

func TestGetZoneName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "example.com",
			expected: "example.com",
		},
		{
			input:    "foo.example.com",
			expected: "example.com",
		},
		{
			input:    "test-domain.whatever.foo.me",
			expected: "foo.me",
		},
		{
			input:    "home",
			expected: "home",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tc := range cases {
		output := getZoneName(tc.input)
		if output != tc.expected {
			t.Logf("Expected `%s` to equal `%s`", output, tc.expected)
			t.Fail()
		}
	}
}
