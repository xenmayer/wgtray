[← Configuration](configuration.md) · [Back to README](../README.md)

# Architecture

WGTray's internal package structure, data flow, and design decisions.

## Package Layers

WGTray uses a layered architecture — dependencies flow downward only:

```
┌─────────────────────────────────────────┐
│  main.go          Entry point            │
├─────────────────────────────────────────┤
│  internal/ui/     Presentation layer     │
│  (tray menu, polling loop)               │
├─────────────────────────────────────────┤
│  internal/wg/     Domain layer           │  internal/auth/
│  (tunnel lifecycle, routing rules)       │  (Touch ID, sudoers)
├─────────────────────────────────────────┤
│  internal/config/ Foundation layer       │
│  (Config/Rules types, file I/O)          │
├─────────────────────────────────────────┤
│  internal/notify/    icon/               │
│  (leaf packages — no internal deps)      │
└─────────────────────────────────────────┘
```

## Package Overview

| Package | Responsibility | Imports |
|---------|---------------|---------|
| `main` | Logging setup, `systray.Run` | `ui`, `config` |
| `internal/ui` | Tray menu, click handlers, 3s polling loop | `wg`, `config`, `notify`, `icon` |
| `internal/wg` | Tunnel connect/disconnect, routing rules, external tunnel detection | `config` |
| `internal/auth` | Touch ID auth, sudoers rule installation | — |
| `internal/config` | `Config`/`Rules` types, `LoadConfigs`, file helpers | — |
| `internal/notify` | macOS user notifications (`Info`, `Error`) | — |
| `icon` | Embedded tray icon bytes (`Connected`, `Disconnected`) | — |

**Dependency rule:** lower-layer packages never import upper-layer packages. `config` and `notify` never import `wg` or `ui`.

## Application Lifecycle

```
systray.Run(ui.OnReady, ui.OnExit)
     │
     ▼
ui.OnReady()
  ├─ wg.NewManager()
  ├─ pre-allocate 20 menu slots (hidden)
  ├─ add static items (Add Config, Open Dir, Logs, Quit)
  ├─ spawn goroutine per slot (watchSlot)
  ├─ spawn goroutine for static items (watchStaticItems)
  ├─ doRefresh()          ← immediate first render
  └─ ticker every 3s → doRefresh()

ui.OnExit()
  └─ mgr.DisconnectAll()
```

## Data Flow: Connect

```
User clicks menu item
  → toggleTunnel(name)
    → config.LoadConfigs()                  # read .conf + .rules.json from disk
    → wg.InterfaceForConfig(cfg.FilePath)   # check if already up externally
    → mgr.Connect(cfg)
        ├─ [include mode] BuildIncludeConfig() → write tmp/<name>.conf
        ├─ [exclude mode] GetDefaultGateway()
        ├─ runAsAdmin("wg-quick up <path>")
        └─ [exclude mode] runAsAdmin("route add -net <cidr> <gw>")
  → doRefresh()                             # update menu and icon
  → notify.Info("Connected", name)
```

## External Tunnel Detection

WGTray auto-detects tunnels started by another tool (e.g. the `wg-quick` CLI):

1. Parse `[Peer] PublicKey` from the `.conf` file.
2. Run `wg show <iface> peers` for each active WireGuard interface.
3. If the key matches → the tunnel is considered connected.

This runs on every 3-second poll tick, so the menu reflects external state changes within 3 seconds.

## Platform Split

macOS is the primary platform. Platform-specific code is split by filename — no `runtime.GOOS` checks:

| File suffix | Platform |
|-------------|----------|
| `_darwin.go` | macOS |
| `_linux.go` | Linux |
| `_other.go` | All others |

Each pair implements the same function signatures so the project compiles on all platforms.

**macOS-only features:**
- Touch ID authentication (`auth/touchid_darwin.m`)
- osascript admin execution (`wg/admin_darwin.go`)
- `route add/delete` for exclude-mode rules

## Tunnel State

`wg.Manager` is the single source of truth for tunnels **this process started**:

```go
type Manager struct {
    mu     sync.RWMutex
    active map[string]*TunnelState  // interface name → state
}
```

External tunnels (started outside WGTray) are detected on-demand via `InterfaceForConfig` — they are not stored in `Manager.active`. The UI reads both sources (`mgr.IsActive(name) || wg.InterfaceForConfig(path) != ""`) to determine displayed status.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| 3-second polling instead of filesystem watch | Avoids platform-specific file watchers; simple and reliable |
| 20 pre-allocated menu slots | Systray on macOS cannot reliably add items after startup |
| Errors returned, not pushed | `wg` and `config` packages never call `notify` — errors surface through the UI layer only |
| Temp configs for include mode | `wg-quick` derives the interface name from the filename; temp file must match original name |
| Sudoers rule on first run | Avoids per-operation password dialogs; scoped to specific binaries only |

## See Also

- [Getting Started](getting-started.md) — installation and first run
- [Configuration](configuration.md) — routing rules reference
