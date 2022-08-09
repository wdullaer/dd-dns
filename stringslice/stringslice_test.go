package stringslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	cases := []struct {
		name        string
		inputSearch string
		inputSlice  []string
		expected    bool
	}{
		{
			name:        "Should return false for an empty slice",
			inputSlice:  []string{},
			inputSearch: "foo",
			expected:    false,
		},
		{
			name:        "Should return false if the searchString is not present",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "hello",
			expected:    false,
		},
		{
			name:        "Should return true if the searchString is present",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "foo",
			expected:    true,
		},
		{
			name:        "Should return true if the searchString is present multiple times",
			inputSlice:  []string{"foo", "bar", "foo", "baz"},
			inputSearch: "foo",
			expected:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := Contains(tc.inputSlice, tc.inputSearch)
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestFindIndex(t *testing.T) {
	cases := []struct {
		name        string
		inputSearch string
		inputSlice  []string
		expected    int
	}{
		{
			name:        "Should return -1 for an empty input slice",
			inputSlice:  []string{},
			inputSearch: "foo",
			expected:    -1,
		},
		{
			name:        "Should return -1 if the inputSearch is not present in the slice",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "hello",
			expected:    -1,
		},
		{
			name:        "Should return the index if the inputSearch is present in the slice",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "foo",
			expected:    0,
		},
		{
			name:        "Should return the first index if the inputSearch is present multiple times",
			inputSlice:  []string{"foo", "bar", "foo", "baz"},
			inputSearch: "foo",
			expected:    0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := FindIndex(tc.inputSlice, tc.inputSearch)
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestRemoveFirst(t *testing.T) {
	cases := []struct {
		name        string
		inputSlice  []string
		inputSearch string
		expected    []string
	}{
		{
			name:        "Should return the input if the inputSearch is not present in the slice",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "hello",
			expected:    []string{"foo", "bar", "baz"},
		},
		{
			name:        "Should return an empty slice if the input is an empty slice",
			inputSlice:  []string{},
			inputSearch: "hello",
			expected:    []string{},
		},
		{
			name:        "Should remove the inputSearch if it is present in the slice",
			inputSlice:  []string{"foo", "bar", "baz"},
			inputSearch: "foo",
			expected:    []string{"bar", "baz"},
		},
		{
			name:        "Should remove the first occurrence of the inputSearch if it is present multiple times",
			inputSlice:  []string{"foo", "bar", "foo", "baz"},
			inputSearch: "foo",
			expected:    []string{"bar", "foo", "baz"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := RemoveFirst(tc.inputSlice, tc.inputSearch)
			assert.ElementsMatch(t, tc.expected, output)
		})
	}
}
