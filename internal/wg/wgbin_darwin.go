//go:build darwin

package wg

import (
	"log"
	"os"
)

// Homebrew installs wireguard-tools to /usr/local/bin on Intel and
// /opt/homebrew/bin on Apple Silicon. Resolved at init time.
var wgBin string
var wgQuickBin string

func init() {
	wgBin = findBinary("wg", []string{
		"/opt/homebrew/bin/wg",
		"/usr/local/bin/wg",
	})
	wgQuickBin = findBinary("wg-quick", []string{
		"/opt/homebrew/bin/wg-quick",
		"/usr/local/bin/wg-quick",
	})
}

// findBinary returns the first existing path from candidates.
// Falls back to the bare name if none exist, relying on PATH lookup.
func findBinary(name string, candidates []string) string {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			log.Printf("wgtray: wg: resolved %s binary: %s", name, p)
			return p
		}
	}
	log.Printf("wgtray: wg: %s binary not found at known paths, falling back to PATH lookup", name)
	return name
}
