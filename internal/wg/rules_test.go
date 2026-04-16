package wg

import "testing"

func TestIsWildcardEntry(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"*.example.com", true},
		{"*.x", true},
		{"*.", false},      // no suffix after "*."
		{"*", false},       // no dot
		{"example.com", false},
		{"*example.com", false}, // no dot after "*"
		{"**example.com", false},
	}
	for _, tc := range tests {
		if got := isWildcardEntry(tc.input); got != tc.want {
			t.Errorf("isWildcardEntry(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestWildcardBaseDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"*.example.com", "example.com"},
		{"*.x", "x"},
		{"*.sub.domain.org", "sub.domain.org"},
	}
	for _, tc := range tests {
		if got := wildcardBaseDomain(tc.input); got != tc.want {
			t.Errorf("wildcardBaseDomain(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
