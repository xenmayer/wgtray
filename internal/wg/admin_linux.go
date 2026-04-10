//go:build linux

package wg

import (
	"fmt"
	"os/exec"
	"strings"
)

// runAsAdmin runs shellCmd with root privileges using pkexec (GUI) or sudo.
func runAsAdmin(shellCmd string) error {
	if _, err := exec.Command("pkexec", "sh", "-c", shellCmd).CombinedOutput(); err == nil {
		return nil
	}
	out, err := exec.Command("sudo", "sh", "-c", shellCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
