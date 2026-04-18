//go:build darwin

package wg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBinary_FirstCandidate(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "wg-first")
	second := filepath.Join(dir, "wg-second")

	if err := os.WriteFile(first, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := findBinary("wg", []string{first, second})
	if got != first {
		t.Errorf("findBinary returned %q, want %q", got, first)
	}
}

func TestFindBinary_SecondCandidate(t *testing.T) {
	dir := t.TempDir()
	second := filepath.Join(dir, "wg-second")

	if err := os.WriteFile(second, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := findBinary("wg", []string{filepath.Join(dir, "wg-missing"), second})
	if got != second {
		t.Errorf("findBinary returned %q, want %q", got, second)
	}
}

func TestFindBinary_FallbackToBareName(t *testing.T) {
	got := findBinary("wg-test", []string{"/nonexistent/path1", "/nonexistent/path2"})
	if got != "wg-test" {
		t.Errorf("findBinary returned %q, want %q", got, "wg-test")
	}
}

func TestResolvedPaths_NonEmpty(t *testing.T) {
	if wgBin == "" {
		t.Error("wgBin is empty after init")
	}
	if wgQuickBin == "" {
		t.Error("wgQuickBin is empty after init")
	}
}
