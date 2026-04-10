//go:build !darwin

package auth

import "errors"

// ErrNotAvailable is returned when Touch ID is not supported on this platform.
var ErrNotAvailable = errors.New("Touch ID not available on this platform")

// Authenticate always succeeds on non-darwin platforms (no biometric auth available).
func Authenticate(_ string) (bool, error) {
	return true, nil
}
