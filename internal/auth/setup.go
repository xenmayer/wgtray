//go:build darwin

package auth

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const sudoersPath = "/etc/sudoers.d/wgtray"

// sudoersVersion tracks the format of the sudoers rule. Bump this when the
// rule content changes so that IsSetupCurrent detects outdated rules.
const sudoersVersion = "2"

// sudoersRule is the NOPASSWD rule installed into /etc/sudoers.d/wgtray.
// It covers both Intel (/usr/local/bin) and ARM64 (/opt/homebrew/bin) Homebrew paths.
const sudoersRule = `%admin ALL=(ALL) NOPASSWD: /usr/local/bin/wg, /opt/homebrew/bin/wg, /usr/local/bin/wg-quick, /opt/homebrew/bin/wg-quick, /sbin/route, /bin/sh`

// IsSetupDone reports whether the sudoers rule is already installed.
// Uses Stat (not ReadFile) — the file is root-owned and not readable.
func IsSetupDone() bool {
	_, err := os.Stat(sudoersPath)
	return err == nil
}

// IsSetupCurrent reports whether the installed sudoers rule matches the
// current version. Uses a local marker file to avoid needing sudo for the check.
func IsSetupCurrent() bool {
	markerPath := filepath.Join(configDirFn(), ".sudoers-version")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == sudoersVersion
}

// writeSudoersMarker records the current sudoers version after a successful setup.
func writeSudoersMarker() {
	markerPath := filepath.Join(configDirFn(), ".sudoers-version")
	if err := os.WriteFile(markerPath, []byte(sudoersVersion), 0o644); err != nil {
		log.Printf("wgtray: auth: failed to write sudoers version marker: %v", err)
	}
}

// configDirFn returns the wgtray config directory. Duplicated here to avoid
// importing internal/config (which would create a circular dependency
// since wg already imports config). Variable for testability.
var configDirFn = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/wgtray"
	}
	return filepath.Join(home, ".config", "wgtray")
}

// RunFirstTimeSetup installs the sudoers rule via a one-time password prompt.
// Uses echo (not printf) — otherwise % in %admin is interpreted by the shell.
func RunFirstTimeSetup() error {
	script := fmt.Sprintf(`do shell script "echo '%s' > /etc/sudoers.d/wgtray && chmod 440 /etc/sudoers.d/wgtray" with administrator privileges`, sudoersRule)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("setup failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	writeSudoersMarker()
	log.Printf("wgtray: auth: sudoers rule installed (version %s)", sudoersVersion)
	return nil
}
