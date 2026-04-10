# Architecture: Layered Architecture

## Overview

WGTray uses a layered architecture with clear horizontal package layers. Each layer depends only on the layer below it — no upward dependencies are allowed. This maps naturally onto Go's `internal/` package convention and the project's data flow: the UI layer drives everything, delegating to domain logic in `wg/`, and reading shared state from `config/`. Supporting packages (`auth/`, `notify/`, `icon/`) are leaf nodes with no internal dependencies.

This pattern was chosen because WGTray is a small, single-binary desktop app with no web server, no database, and straightforward business logic. A heavier pattern (Clean Architecture, DDD) would add unnecessary abstraction. The existing codebase already follows this structure — this document codifies it.

## Decision Rationale

- **Project type:** macOS menu bar app (systray), single binary
- **Tech stack:** Go 1.20, `fyne.io/systray`, Makefile
- **Team size:** Small (1–3)
- **Domain complexity:** Low — tunnel lifecycle, routing rules, file I/O
- **Key factor:** Existing code already has a clean layered structure; documenting it prevents drift

## Package Layers

```
┌─────────────────────────────────────────────────────────┐
│                    Layer 0: Entry Point                  │
│  main.go  — logging setup, systray.Run                   │
├─────────────────────────────────────────────────────────┤
│                    Layer 1: UI / Presentation            │
│  internal/ui/tray.go  — menu, slots, polling loop        │
├─────────────────────────────────────────────────────────┤
│                    Layer 2: Domain / Logic               │
│  internal/wg/   — tunnel lifecycle, routing rules        │
│  internal/auth/ — Touch ID, sudoers setup                │
├─────────────────────────────────────────────────────────┤
│                    Layer 3: Foundation                   │
│  internal/config/ — Config/Rules types, file I/O         │
├─────────────────────────────────────────────────────────┤
│                    Leaf Packages (no internal deps)      │
│  internal/notify/ — user notifications                   │
│  icon/            — embedded icon bytes                  │
└─────────────────────────────────────────────────────────┘
```

## Folder Structure

```
wgtray/
├── main.go                 # Entry point — Layer 0
├── go.mod / go.sum
├── Info.plist
├── Makefile
├── internal/
│   ├── config/             # Layer 3 — base types and file I/O
│   │   └── store.go
│   ├── wg/                 # Layer 2 — WireGuard domain logic
│   │   ├── manager.go
│   │   ├── rules.go
│   │   ├── admin_darwin.go
│   │   ├── admin_linux.go
│   │   ├── wgbin_darwin.go
│   │   └── wgbin_linux.go
│   ├── auth/               # Layer 2 — authentication domain
│   │   ├── touchid.go
│   │   ├── touchid_darwin.{m,h}
│   │   ├── touchid_other.go
│   │   ├── setup.go
│   │   └── setup_other.go
│   ├── notify/             # Leaf — notifications (no internal imports)
│   │   ├── notify.go
│   │   └── notify_other.go
│   └── ui/                 # Layer 1 — presentation
│       └── tray.go
└── icon/                   # Leaf — embedded assets
    └── icon.go
```

## Dependency Rules

Dependencies flow **downward only**. Inner/lower layers never import outer/upper layers.

```
main   →  ui, config
ui     →  wg, config, notify, icon
wg     →  config
auth   →  (nothing internal)
notify →  (nothing internal)
icon   →  (nothing internal)
config →  (nothing internal)
```

- ✅ `ui` may import `wg`, `config`, `notify`, `icon`
- ✅ `wg` may import `config`
- ✅ `main` may import `ui`, `config`
- ❌ `config` must NOT import `wg`, `ui`, `notify`, or `auth`
- ❌ `wg` must NOT import `ui` or `notify` (surface errors upward via return values, not side effects)
- ❌ `notify` must NOT import any other internal package
- ❌ `icon` must NOT import any other internal package

## Layer Communication

- **main → ui:** `systray.Run(ui.OnReady, ui.OnExit)` — event-driven lifecycle callbacks
- **ui → wg:** direct method calls on `*wg.Manager`; `wg.InterfaceForConfig` for external tunnel detection
- **ui → config:** `config.LoadConfigs()` on every poll tick; file helpers for add/open actions
- **ui → notify:** `notify.Info` / `notify.Error` for user-visible feedback after tunnel operations
- **wg → config:** reads `config.Config`, `config.Rules`, `config.ConfigDir()` — types and paths only
- Error propagation: errors return up the call stack; only the `ui` layer converts them into notifications

## Key Principles

1. **Single-direction data flow:** config is read at UI tick time and passed down — domain packages do not hold config caches
2. **Platform split by filename:** `_darwin.go` / `_linux.go` / `_other.go` — no `runtime.GOOS` checks in shared code
3. **Leaf packages are pure:** `notify` and `icon` have no internal dependencies and no side effects on import
4. **Manager owns tunnel state:** `wg.Manager` is the single source of truth for which tunnels *this process* started; external tunnel state is checked on-demand via `InterfaceForConfig`
5. **UI drives all state:** the 3-second polling loop in `ui` is the only place that reads config and refreshes the menu — domain packages do not push updates

## Code Examples

### Correct: error surfaces up through layers

```go
// internal/wg/manager.go (Layer 2) — returns error, no notification
func (m *Manager) Connect(cfg config.Config) error {
    if err := runAsAdmin(cmd); err != nil {
        return fmt.Errorf("connect %s: %w", cfg.Name, err)
    }
    return nil
}

// internal/ui/tray.go (Layer 1) — handles the error, notifies user
if err := mgr.Connect(*targetCfg); err != nil {
    log.Printf("wgtray: connect %s: %v", name, err)
    notify.Error("Connect failed", fmt.Sprintf("%s: %v", name, err))
}
```

### Correct: config read at the UI layer, passed down

```go
// ui/tray.go — reads config each poll tick
cfgs, err := config.LoadConfigs()
// ...
for _, cfg := range cfgs {
    connected := mgr.IsActive(cfg.Name) || wg.InterfaceForConfig(cfg.FilePath) != ""
}
```

### Incorrect: lower layer reaching up

```go
// ❌ WRONG — wg package importing notify (upward dependency)
import "wgtray/internal/notify"

func (m *Manager) Connect(cfg config.Config) error {
    if err := runAsAdmin(cmd); err != nil {
        notify.Error("Connect failed", err.Error()) // ❌ breaks layer rule
    }
}
```

### Correct: platform split

```go
// internal/wg/admin_darwin.go
func runAsAdmin(cmd string) error { /* osascript implementation */ }

// internal/wg/admin_linux.go
func runAsAdmin(cmd string) error { /* Linux implementation */ }
```

## Anti-Patterns

- ❌ **Skipping layers:** `main.go` calling `config.LoadConfigs()` directly instead of going through `ui`
- ❌ **Upward imports:** `wg` or `config` importing `notify` or `ui` to show errors or update the menu
- ❌ **Runtime OS checks:** using `if runtime.GOOS == "darwin"` in shared code instead of platform files
- ❌ **Domain side-effects:** `wg.Manager.Connect` sending notifications or updating menu items directly
- ❌ **Shared mutable state:** using package-level variables for tunnel state outside `wg.Manager`
- ❌ **Config caching in domain layer:** `wg` storing a config snapshot — always read fresh from `config.LoadConfigs()`
