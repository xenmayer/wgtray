package wg

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"wgtray/internal/config"
)

// ResolveEntries resolves domain names to IPs and normalises all entries to
// CIDR notation. Duplicate entries are suppressed.
func ResolveEntries(entries []string) []string {
	seen := make(map[string]bool)
	var out []string

	add := func(cidr string) {
		if !seen[cidr] {
			seen[cidr] = true
			out = append(out, cidr)
		}
	}

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Already CIDR?
		if _, _, err := net.ParseCIDR(entry); err == nil {
			add(entry)
			continue
		}

		// Bare IP?
		if ip := net.ParseIP(entry); ip != nil {
			if ip.To4() != nil {
				add(entry + "/32")
			} else {
				add(entry + "/128")
			}
			continue
		}

		// DNS lookup.
		ips, err := net.LookupHost(entry)
		if err != nil {
			continue
		}
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if ip.To4() != nil {
				add(ipStr + "/32")
			} else {
				add(ipStr + "/128")
			}
		}
	}

	return out
}

// GetDefaultGateway returns the current default gateway IP address.
func GetDefaultGateway() (string, error) {
	out, err := exec.Command("sh", "-c",
		"route -n get default 2>/dev/null | grep 'gateway:' | awk '{print $2}'").Output()
	if err == nil {
		if gw := strings.TrimSpace(string(out)); gw != "" {
			return gw, nil
		}
	}
	// Fallback via netstat.
	out2, err2 := exec.Command("sh", "-c",
		"netstat -rn 2>/dev/null | awk '/^default/{print $2; exit}'").Output()
	if err2 != nil {
		return "", fmt.Errorf("get default gateway: %w", err)
	}
	gw := strings.TrimSpace(string(out2))
	if gw == "" {
		return "", fmt.Errorf("could not determine default gateway")
	}
	return gw, nil
}

// BuildIncludeConfig returns a modified WireGuard config where AllowedIPs in
// every [Peer] section is replaced with the resolved entries from rules.
func BuildIncludeConfig(original string, rules config.Rules) (string, error) {
	resolved := ResolveEntries(rules.Entries)
	if len(resolved) == 0 {
		return original, nil
	}

	allowedIPs := strings.Join(resolved, ", ")
	lines := strings.Split(original, "\n")
	result := make([]string, 0, len(lines)+2)
	inPeer := false
	wroteAllowedIPs := false

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))

		switch {
		case lower == "[peer]":
			inPeer = true
			wroteAllowedIPs = false
			result = append(result, line)

		case strings.HasPrefix(lower, "[") && lower != "[peer]":
			if inPeer && !wroteAllowedIPs {
				result = append(result, "AllowedIPs = "+allowedIPs)
			}
			inPeer = false
			result = append(result, line)

		case inPeer && strings.HasPrefix(lower, "allowedips"):
			if !wroteAllowedIPs {
				result = append(result, "AllowedIPs = "+allowedIPs)
				wroteAllowedIPs = true
			}
			// Drop the original AllowedIPs line.

		default:
			result = append(result, line)
		}
	}

	// Handle file ending inside a [Peer] section.
	if inPeer && !wroteAllowedIPs {
		result = append(result, "AllowedIPs = "+allowedIPs)
	}

	return strings.Join(result, "\n"), nil
}
