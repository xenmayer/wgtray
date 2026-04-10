//go:build !darwin

package notify

import "log"

// Error logs an error notification (stub on non-darwin platforms).
func Error(title, message string) {
	log.Printf("error: %s: %s", title, message)
}

// Info logs an info notification (stub on non-darwin platforms).
func Info(title, message string) {
	log.Printf("info: %s: %s", title, message)
}
