//go:build darwin

package wg

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"wgtray/internal/auth"
)

// runAsAdmin executes shellCmd with root privileges.
// It requests Touch ID first; on success uses sudo (NOPASSWD).
// Falls back to osascript password dialog if sudo is not configured or Touch ID is unavailable.
func runAsAdmin(shellCmd string) error {
	ok, err := auth.Authenticate("WG VPN requires administrator privileges")
	if err != nil {
		return runAsAdminOsascript(shellCmd)
	}
	if !ok {
		return fmt.Errorf("authentication cancelled")
	}
	out, sudoErr := exec.Command("sudo", "-n", "sh", "-c", shellCmd).CombinedOutput()
	if sudoErr == nil {
		return nil
	}
	log.Printf("wgtray: sudo -n failed (%v), falling back to osascript", sudoErr)
	_ = out
	return runAsAdminOsascript(shellCmd)
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
