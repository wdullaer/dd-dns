package stringslice

// Contains returns true if the given string is present at least once in the slice
func Contains(col []string, item string) bool {
	for i := range col {
		if col[i] == item {
			return true
		}
	}
	return false
}

// RemoveFirst returns a slice with the string removed
// If the string is present multiple times, only the first will be removed
// If the string is not present in the slice the original slice is returned
// Mutates the slice in place, does not copy
func RemoveFirst(col []string, item string) []string {
	for i := range col {
		if col[i] == item {
			col[i] = col[len(col)-1]
			return col[:len(col)-1]
		}
	}
	return col
}

// FindIndex returns the index of a particular string
// Returns -1 if the string is not present in the slice
func FindIndex(col []string, item string) int {
	for i := range col {
		if col[i] == item {
			return i
		}
	}
	return -1
}
