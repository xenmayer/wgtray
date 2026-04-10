//go:build darwin

package wg

// Homebrew installs wireguard-tools to /usr/local/bin on Intel and
// /opt/homebrew/bin on Apple Silicon (symlinked to /usr/local/bin by default).
const wgBin      = "/usr/local/bin/wg"
const wgQuickBin = "/usr/local/bin/wg-quick"
