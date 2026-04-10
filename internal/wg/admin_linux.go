//go:build linux

package wg

import (
	"fmt"
	"os/exec"
	"strings"
)

// runAsAdmin executes shellCmd with root privileges using pkexec or sudo.
func runAsAdmin(shellCmd string) error {
	out, err := exec.Command("pkexec", "sh", "-c", shellCmd).CombinedOutput()
	if err == nil {
		return nil
	}
	out2, err2 := exec.Command("sudo", "sh", "-c", shellCmd).CombinedOutput()
	if err2 != nil {
		return fmt.Errorf("%w: %s", err2, strings.TrimSpace(string(out2)))
	}
	_ = out
	return nil
}
