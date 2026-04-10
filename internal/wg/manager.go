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
	out, err := exec.Command("wg", "show", "interfaces").Output()
	if err != nil {
		out, err = exec.Command("sudo", "-n", "wg", "show", "interfaces").Output()
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

	// Build the shell command(s) to run with admin privileges.
	cmds := []string{wgQuickBin + " up " + shellQuote(upConfigPath)}
	if cfg.Rules.Mode == "exclude" && state.Gateway != "" {
		for _, cidr := range ResolveEntries(cfg.Rules.Entries) {
			cmds = append(cmds, fmt.Sprintf("route add -net %s %s", cidr, state.Gateway))
		}
	}

	if err := runAsAdmin(strings.Join(cmds, " && ")); err != nil {
		if state.TempPath != "" {
			os.Remove(state.TempPath)
		}
		return fmt.Errorf("connect %s: %w", cfg.Name, err)
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

	cmds := []string{wgQuickBin + " down " + shellQuote(downConfigPath)}
	if state.Rules.Mode == "exclude" && len(state.Rules.Entries) > 0 {
		for _, cidr := range ResolveEntries(state.Rules.Entries) {
			cmds = append(cmds, "route delete -net "+cidr)
		}
	}

	if err := runAsAdmin(strings.Join(cmds, " && ")); err != nil {
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
