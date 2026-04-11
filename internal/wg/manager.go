package wg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"wgtray/internal/config"
)

// TunnelState holds the runtime state of a managed WireGuard tunnel.
type TunnelState struct {
	ConfigPath string
	TempPath   string // non-empty when a modified config was written (include mode)
	Rules      config.Rules
	Gateway    string // default gateway captured at connect time (exclude mode)
}

// Manager manages the lifecycle of WireGuard tunnels.
type Manager struct {
	mu     sync.RWMutex
	active map[string]*TunnelState // interface name → state
}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{active: make(map[string]*TunnelState)}
}

// ActiveInterfaces returns the names of currently active WireGuard interfaces.
// It tries without sudo first; falls back to sudo -n (non-interactive).
func ActiveInterfaces() ([]string, error) {
	out, err := exec.Command(wgBin, "show", "interfaces").Output()
	if err != nil {
		out, err = exec.Command("sudo", "-n", wgBin, "show", "interfaces").Output()
		if err != nil {
			return nil, nil
		}
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Fields(raw), nil
}

// IsConnected reports whether the named WireGuard interface is currently up.
// NOTE: on macOS wg show interfaces returns utun names (utun4), not config names.
// Use IsConfigConnected or Manager.IsActive for reliable checks.
func IsConnected(name string) bool {
	ifaces, _ := ActiveInterfaces()
	for _, iface := range ifaces {
		if iface == name {
			return true
		}
	}
	return false
}

// IsActive reports whether this Manager instance started the named tunnel.
func (m *Manager) IsActive(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.active[name]
	return ok
}

// InterfaceForConfig returns the running WireGuard interface name for the
// given config file path (matched by peer public key), or "" if not found.
// Uses sudo -n sh to read interface details as root.
func InterfaceForConfig(cfgPath string) string {
	peerKey := extractPeerPublicKey(cfgPath)
	if peerKey == "" {
		return ""
	}
	ifaces, _ := ActiveInterfaces()
	for _, iface := range ifaces {
		cmd := fmt.Sprintf("%s show %s peers 2>/dev/null", wgBin, iface)
		out, err := exec.Command("sudo", "-n", "sh", "-c", cmd).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Fields(string(out)) {
			if line == peerKey {
				return iface
			}
		}
	}
	return ""
}

// extractPeerPublicKey parses the [Peer] PublicKey from a wg config file.
func extractPeerPublicKey(cfgPath string) string {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	inPeer := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.EqualFold(line, "[Peer]") {
			inPeer = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPeer = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(line), "publickey") {
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// Connect brings up the WireGuard tunnel for cfg.
func (m *Manager) Connect(cfg config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.active[cfg.Name]; exists {
		return fmt.Errorf("tunnel %q is already active", cfg.Name)
	}

	state := &TunnelState{
		ConfigPath: cfg.FilePath,
		Rules:      cfg.Rules,
	}

	upConfigPath := cfg.FilePath

	// Include mode: rewrite AllowedIPs to only the listed entries.
	if cfg.Rules.Mode == "include" && len(cfg.Rules.Entries) > 0 {
		data, err := os.ReadFile(cfg.FilePath)
		if err != nil {
			return fmt.Errorf("read config %s: %w", cfg.FilePath, err)
		}
		modified, err := BuildIncludeConfig(string(data), cfg.Rules)
		if err != nil {
			return fmt.Errorf("build include config: %w", err)
		}
		// Use tmp/<name>.conf so wg-quick derives the correct interface name.
		tmpDir := filepath.Join(config.ConfigDir(), "tmp")
		if err := os.MkdirAll(tmpDir, 0o700); err != nil {
			return fmt.Errorf("create tmp dir: %w", err)
		}
		tmpPath := filepath.Join(tmpDir, cfg.Name+".conf")
		if err := os.WriteFile(tmpPath, []byte(modified), 0o600); err != nil {
			return fmt.Errorf("write temp config: %w", err)
		}
		upConfigPath = tmpPath
		state.TempPath = tmpPath
	}

	// Exclude mode: capture the default gateway before the tunnel changes routing.
	if cfg.Rules.Mode == "exclude" && len(cfg.Rules.Entries) > 0 {
		gw, err := GetDefaultGateway()
		if err != nil {
			log.Printf("wgtray: gateway lookup: %v — exclude routes will not be added", err)
		} else {
			state.Gateway = gw
		}
	}

	// Check if tunnel is already running (started externally).
	if iface := InterfaceForConfig(cfg.FilePath); iface != "" {
		log.Printf("wgtray: tunnel %q already up as %s (external), tracking it", cfg.Name, iface)
		m.active[cfg.Name] = state
		return nil
	}

	// Bring up the tunnel first (separate from route commands so a route
	// failure does not leave the tunnel up but untracked).
	upCmd := wgQuickBin + " up " + shellQuote(upConfigPath)

	if err := runAsAdmin(upCmd); err != nil {
		// Tunnel might already be running (e.g. started externally and not
		// detected because sudo -n was unavailable for wg show).
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("wgtray: tunnel %q already exists, disconnecting first", cfg.Name)
			downCmd := wgQuickBin + " down " + shellQuote(upConfigPath)
			if downErr := runAsAdmin(downCmd); downErr != nil {
				if state.TempPath != "" {
					os.Remove(state.TempPath)
				}
				return fmt.Errorf("connect %s: down existing tunnel: %w", cfg.Name, downErr)
			}
			if retryErr := runAsAdmin(upCmd); retryErr != nil {
				if state.TempPath != "" {
					os.Remove(state.TempPath)
				}
				return fmt.Errorf("connect %s: %w", cfg.Name, retryErr)
			}
		} else {
			if state.TempPath != "" {
				os.Remove(state.TempPath)
			}
			return fmt.Errorf("connect %s: %w", cfg.Name, err)
		}
	}

	// Add exclude-mode routes separately; failures are logged but non-fatal
	// (the tunnel is already up and must remain tracked).
	if cfg.Rules.Mode == "exclude" && state.Gateway != "" {
		routeCmds := buildRouteAddCmds(ResolveEntries(cfg.Rules.Entries), state.Gateway)
		if len(routeCmds) > 0 {
			// Join with "; " so one failing route doesn't abort the rest.
			if err := runAsAdmin(strings.Join(routeCmds, "; ")); err != nil {
				log.Printf("wgtray: exclude routes for %s (non-fatal): %v", cfg.Name, err)
			}
		}
	}

	m.active[cfg.Name] = state
	return nil
}

// Disconnect tears down the named WireGuard tunnel.
func (m *Manager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, managed := m.active[name]
	if !managed {
		// Tunnel wasn't started by us; try the default config path.
		state = &TunnelState{
			ConfigPath: filepath.Join(config.ConfigDir(), name+".conf"),
		}
	}

	// wg-quick down needs the same config that was used for up.
	downConfigPath := state.ConfigPath
	if state.TempPath != "" {
		downConfigPath = state.TempPath
	}

	// Remove exclude-mode routes first (non-fatal).
	if state.Rules.Mode == "exclude" && len(state.Rules.Entries) > 0 {
		routeCmds := buildRouteDeleteCmds(ResolveEntries(state.Rules.Entries))
		if len(routeCmds) > 0 {
			if err := runAsAdmin(strings.Join(routeCmds, "; ")); err != nil {
				log.Printf("wgtray: delete routes for %s (non-fatal): %v", name, err)
			}
		}
	}

	downCmd := wgQuickBin + " down " + shellQuote(downConfigPath)
	if err := runAsAdmin(downCmd); err != nil {
		return fmt.Errorf("disconnect %s: %w", name, err)
	}

	if managed && state.TempPath != "" {
		os.Remove(state.TempPath)
	}
	delete(m.active, name)
	return nil
}

// DisconnectAll disconnects every tunnel this Manager brought up.
func (m *Manager) DisconnectAll() {
	m.mu.RLock()
	names := make([]string, 0, len(m.active))
	for name := range m.active {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.Disconnect(name); err != nil {
			log.Printf("wgtray: disconnect %s on exit: %v", name, err)
		}
	}
}

// shellQuote wraps s in single quotes suitable for embedding in a shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// isIPv6CIDR returns true if cidr is an IPv6 address block.
func isIPv6CIDR(cidr string) bool {
	return strings.Contains(cidr, ":")
}

// buildRouteAddCmds returns shell commands to add routes via gateway.
// Uses "route add -inet6" for IPv6 and "route add -net" for IPv4.
// Each command is suffixed with "|| true" so failures don't abort the chain.
func buildRouteAddCmds(cidrs []string, gateway string) []string {
	cmds := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if isIPv6CIDR(cidr) {
			cmds = append(cmds, fmt.Sprintf("route add -inet6 %s %s || true", cidr, gateway))
		} else {
			cmds = append(cmds, fmt.Sprintf("route add -net %s %s || true", cidr, gateway))
		}
	}
	return cmds
}

// buildRouteDeleteCmds returns shell commands to delete routes.
// Each command is suffixed with "|| true" so failures don't abort the chain.
func buildRouteDeleteCmds(cidrs []string) []string {
	cmds := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if isIPv6CIDR(cidr) {
			cmds = append(cmds, "route delete -inet6 "+cidr+" || true")
		} else {
			cmds = append(cmds, "route delete -net "+cidr+" || true")
		}
	}
	return cmds
}
