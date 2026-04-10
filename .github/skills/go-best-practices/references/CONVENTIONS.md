# Go Conventions — Extended Reference

## Error Wrapping Patterns

### Multi-step operations
Wrap at each boundary so the error chain describes the full path:

```go
func (m *Manager) Connect(cfg config.Config) error {
    data, err := os.ReadFile(cfg.FilePath)
    if err != nil {
        return fmt.Errorf("read config %s: %w", cfg.FilePath, err)
    }
    modified, err := BuildIncludeConfig(string(data), cfg.Rules)
    if err != nil {
        return fmt.Errorf("build include config: %w", err)
    }
    // ...
}
```

### Cleanup on error
Always clean up resources on the error path:

```go
if err := runAsAdmin(cmd); err != nil {
    if state.TempPath != "" {
        os.Remove(state.TempPath)  // cleanup temp file
    }
    return fmt.Errorf("connect %s: %w", cfg.Name, err)
}
```

---

## Concurrency — Extended Patterns

### Read-heavy map with RWMutex

```go
// Snapshot keys under read lock, then process outside the lock
func (m *Manager) DisconnectAll() {
    m.mu.RLock()
    names := make([]string, 0, len(m.active))
    for name := range m.active {
        names = append(names, name)
    }
    m.mu.RUnlock()

    for _, name := range names {
        m.Disconnect(name)  // Disconnect takes its own write lock
    }
}
```

### Per-item lock in a goroutine
Read the item's stable identifier under lock, then act outside the lock:

```go
func watchSlot(s *slot) {
    for {
        select {
        case <-s.mainItem.ClickedCh:
            s.mu.Lock()
            name := s.name  // copy name
            s.mu.Unlock()
            if name == "" {
                continue
            }
            toggleTunnel(name)  // called outside lock
        }
    }
}
```

---

## Admin Execution (macOS)

Admin commands are run via `osascript` with AppleScript's `do shell script ... with administrator privileges`.
After sudoers installation, `sudo -n` is used instead to avoid the password dialog.

Multiple shell commands are joined with ` && `:

```go
cmds := []string{wgQuickBin + " up " + shellQuote(upConfigPath)}
for _, cidr := range ResolveEntries(cfg.Rules.Entries) {
    cmds = append(cmds, fmt.Sprintf("route add -net %s %s", cidr, state.Gateway))
}
runAsAdmin(strings.Join(cmds, " && "))
```

`shellQuote` wraps strings in single quotes safe for shell embedding:

```go
func shellQuote(s string) string {
    return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
```

---

## Routing Rules

Two routing modes:

| Mode | Mechanism |
|------|-----------|
| `exclude` | Capture default gateway before connect; add `route add -net <cidr> <gw>` for each entry after `wg-quick up` |
| `include` | Rewrite `AllowedIPs` in a temp config (`tmp/<name>.conf`) so only listed CIDRs go through VPN |

Temp configs live at `~/.config/wgtray/tmp/<name>.conf` and are removed on disconnect.

---

## External Tunnel Detection

WGTray detects tunnels it did not start by matching the peer public key:

1. Parse `[Peer] PublicKey = <key>` from the `.conf` file
2. Run `wg show <iface> peers` for each active interface
3. If the key appears → the tunnel is running as that interface

```go
func InterfaceForConfig(cfgPath string) string {
    peerKey := extractPeerPublicKey(cfgPath)
    if peerKey == "" {
        return ""
    }
    ifaces, _ := ActiveInterfaces()
    for _, iface := range ifaces {
        // sudo -n sh -c "wg show <iface> peers"
        if peerKeyFound(iface, peerKey) {
            return iface
        }
    }
    return ""
}
```

---

## Embedded Icons

Icons are embedded as byte slices in `icon/icon.go` and exposed as functions:

```go
func Connected() []byte    { return trayConnected }
func Disconnected() []byte { return trayDisconnected }
```

Use `icon.Connected()` and `icon.Disconnected()` in `ui/tray.go` — never access icon files directly at runtime.
