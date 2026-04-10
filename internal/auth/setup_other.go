//go:build !darwin

package auth

// IsSetupDone always returns true on non-darwin platforms.
func IsSetupDone() bool {
	return true
}

// RunFirstTimeSetup is a no-op on non-darwin platforms.
func RunFirstTimeSetup() error {
	return nil
}
