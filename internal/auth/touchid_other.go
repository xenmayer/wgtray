//go:build !darwin

package auth

import "errors"

var ErrNotAvailable = errors.New("Touch ID not available on this device")

// Authenticate is a stub for non-darwin platforms; always returns true.
func Authenticate(reason string) (bool, error) {
	return true, nil
}
