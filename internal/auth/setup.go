//go:build darwin

package auth

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)


const sudoersPath = "/etc/sudoers.d/wgtray"

// IsSetupDone reports whether the sudoers rule is already installed.
// Uses Stat (not ReadFile) — the file is root-owned and not readable.
func IsSetupDone() bool {
	_, err := os.Stat(sudoersPath)
	return err == nil
}

// RunFirstTimeSetup installs the sudoers rule via a one-time password prompt.
// Uses echo (not printf) — otherwise % in %admin is interpreted by the shell.
func RunFirstTimeSetup() error {
	script := `do shell script "echo '%admin ALL=(ALL) NOPASSWD: /usr/local/bin/wg-quick, /sbin/route, /bin/sh' > /etc/sudoers.d/wgtray && chmod 440 /etc/sudoers.d/wgtray" with administrator privileges`
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("setup failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
