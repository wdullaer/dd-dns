package stringslice

import "testing"

func TestContains(t *testing.T) {
	input := []string{"foo", "bar", "baz"}

	searchString := "Hello"
	if Contains(input, searchString) {
		t.Logf("`%s` should not contain `%s`", input, searchString)
		t.Fail()
	}

	searchString = "bar"
	if !Contains(input, searchString) {
		t.Logf("`%s` should contain `%s`", input, searchString)
		t.Fail()
	}
}

func TestFindIndex(t *testing.T) {
	input := []string{"foo", "bar", "baz"}

	searchString := "Hello"
	if FindIndex(input, searchString) != -1 {
		t.Logf("`%s` should have index -1 in `%s`", searchString, input)
		t.Fail()
	}

	searchString = "bar"
	if FindIndex(input, searchString) != 1 {
		t.Logf("`%s` should have index 1 in `%s`", searchString, input)
		t.Fail()
	}
}

func TestRemoveFirst(t *testing.T) {
	input := []string{"foo", "bar", "baz"}

	removeString := "Hello"
	output := RemoveFirst(input, removeString)
	if len(output) != len(input) {
		t.Logf("`%s` should have same length as `%s` when trying to remove `%s`", output, input, removeString)
		t.Fail()
	}

	removeString = "foo"
	output = RemoveFirst(input, removeString)
	expected := []string{"baz", "bar"}
	if !equal(output, expected) {
		t.Logf("`%s` should  be equal to `%s`", output, expected)
		t.Fail()
	}
}

func equal(a, b []string) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
