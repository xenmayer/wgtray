//go:build darwin

package wg

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"wgtray/internal/auth"
)

// brewPath includes Homebrew directories (both Apple Silicon and Intel) so that
// wg-quick and its dependencies (bash 4+, wg, etc.) are found when the app is
// launched via Launch Services, which provides only a minimal PATH.
const brewPath = "/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin"

// withPath prepends a PATH export so that shell commands work correctly
// regardless of the environment they are launched from.
func withPath(cmd string) string {
	return "export PATH=" + brewPath + "; " + cmd
}

// runAsAdmin executes shellCmd with root privileges.
// Tries sudo -n first (works silently if the NOPASSWD sudoers rule is installed).
// If that fails, installs the sudoers rule with a single password prompt and retries.
// Falls back to an osascript password dialog as a last resort.
func runAsAdmin(shellCmd string) error {
	patched := withPath(shellCmd)

	if out, err := exec.Command("sudo", "-n", "sh", "-c", patched).CombinedOutput(); err == nil {
		return nil
	} else {
		log.Printf("wgtray: sudo -n failed (%v: %s), attempting setup", err, strings.TrimSpace(string(out)))
	}

	// Sudoers rule missing or outdated — install it (one password prompt).
	if setupErr := auth.RunFirstTimeSetup(); setupErr != nil {
		log.Printf("wgtray: setup failed (%v), falling back to osascript", setupErr)
		return runAsAdminOsascript(patched)
	}

	// Retry after successful setup.
	if out, err := exec.Command("sudo", "-n", "sh", "-c", patched).CombinedOutput(); err != nil {
		log.Printf("wgtray: sudo -n still failed after setup (%v: %s), falling back to osascript", err, strings.TrimSpace(string(out)))
		return runAsAdminOsascript(patched)
	}
	return nil
}

// runAsAdminOsascript requests the password via the native macOS dialog.
func runAsAdminOsascript(shellCmd string) error {
	escaped := strings.ReplaceAll(shellCmd, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, escaped)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
