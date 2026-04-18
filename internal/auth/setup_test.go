//go:build darwin

package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSudoersRule_ContainsBothHomebrewPaths(t *testing.T) {
	required := []string{
		"/opt/homebrew/bin/wg",
		"/opt/homebrew/bin/wg-quick",
		"/usr/local/bin/wg",
		"/usr/local/bin/wg-quick",
		"/sbin/route",
		"/bin/sh",
	}
	for _, path := range required {
		if !strings.Contains(sudoersRule, path) {
			t.Errorf("sudoersRule missing %q", path)
		}
	}
}

func TestSudoersRule_ValidFormat(t *testing.T) {
	if !strings.HasPrefix(sudoersRule, "%admin ALL=(ALL) NOPASSWD:") {
		t.Errorf("sudoersRule has unexpected format: %q", sudoersRule)
	}
}

func TestIsSetupCurrent_MatchingVersion(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".sudoers-version")
	if err := os.WriteFile(marker, []byte(sudoersVersion), 0o644); err != nil {
		t.Fatal(err)
	}

	// Temporarily override configDir for the test.
	origFn := configDirFn
	configDirFn = func() string { return dir }
	defer func() { configDirFn = origFn }()

	if !IsSetupCurrent() {
		t.Error("IsSetupCurrent returned false for matching version")
	}
}

func TestIsSetupCurrent_MissingMarker(t *testing.T) {
	dir := t.TempDir()

	origFn := configDirFn
	configDirFn = func() string { return dir }
	defer func() { configDirFn = origFn }()

	if IsSetupCurrent() {
		t.Error("IsSetupCurrent returned true when marker file is missing")
	}
}

func TestIsSetupCurrent_OldVersion(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".sudoers-version")
	if err := os.WriteFile(marker, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	origFn := configDirFn
	configDirFn = func() string { return dir }
	defer func() { configDirFn = origFn }()

	if IsSetupCurrent() {
		t.Error("IsSetupCurrent returned true for old version")
	}
}
