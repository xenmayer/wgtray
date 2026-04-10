//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// Error shows a macOS error notification with sound.
func Error(title, message string) {
	script := fmt.Sprintf(
		`display notification %q with title %q subtitle "WG VPN" sound name "Basso"`,
		message, title,
	)
	exec.Command("osascript", "-e", script).Start() //nolint:errcheck
}

// Info shows a macOS notification (no sound).
func Info(title, message string) {
	script := fmt.Sprintf(
		`display notification %q with title %q subtitle "WG VPN"`,
		message, title,
	)
	exec.Command("osascript", "-e", script).Start() //nolint:errcheck
}
