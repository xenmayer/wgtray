//go:build !darwin

package notify

import "log"

// Error logs an error notification (no desktop notification on non-darwin).
func Error(title, message string) {
	log.Printf("wgtray [error] %s: %s", title, message)
}

// Info logs an info notification (no desktop notification on non-darwin).
func Info(title, message string) {
	log.Printf("wgtray [info] %s: %s", title, message)
}
