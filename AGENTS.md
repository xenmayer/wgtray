# AGENTS.md

> Project map for AI agents. Keep this file up-to-date as the project evolves.

## Project Overview

WGTray is a macOS menu bar application that manages WireGuard VPN tunnels with per-config routing rules (exclude/include mode), Touch ID authentication, and auto-detection of externally started tunnels.

## Tech Stack

- **Language:** Go 1.20
- **UI/Tray:** fyne.io/systray v1.11.0
- **Platform:** macOS (primary); Linux partial support
- **Build:** Makefile

## Project Structure

```
wgtray/
├── main.go                        # Entry point: file logging + systray.Run
├── go.mod / go.sum                # Go module definition
├── Info.plist                     # macOS app metadata
├── Makefile                       # build / bundle / install / clean
├── internal/
│   ├── config/
│   │   └── store.go               # Config/Rules types; LoadConfigs, CopyConfigFile, EnsureRulesFile
│   ├── wg/
│   │   ├── manager.go             # Manager: Connect, Disconnect, DisconnectAll, IsActive
│   │   ├── rules.go               # ResolveEntries, BuildIncludeConfig routing helpers
│   │   ├── admin_darwin.go        # runAsAdmin via osascript; GetDefaultGateway
│   │   ├── admin_linux.go         # Linux stub for runAsAdmin
│   │   ├── wgbin_darwin.go        # wgBin / wgQuickBin paths for macOS
│   │   └── wgbin_linux.go         # wgBin / wgQuickBin paths for Linux
│   ├── auth/
│   │   ├── touchid.go             # Touch ID interface
│   │   ├── touchid_darwin.{m,h}   # Objective-C Touch ID implementation
│   │   ├── touchid_other.go       # Non-Darwin stub
│   │   ├── setup.go               # sudoers rule installation
│   │   └── setup_other.go         # Non-Darwin stub
│   ├── notify/
│   │   ├── notify.go              # Info / Error notifications (macOS)
│   │   └── notify_other.go        # Non-Darwin stub
│   └── ui/
│       └── tray.go                # OnReady/OnExit, slot management, polling loop (3s)
└── icon/
    ├── icon.go                    # Connected() / Disconnected() embedded icon bytes
    ├── icon.png                   # App icon
    ├── tray_connected.png         # Menu bar icon — connected state
    ├── tray_disconnected.png      # Menu bar icon — disconnected state
    └── wgtray.icns                # macOS app icon bundle
```

## Key Entry Points

| File | Purpose |
|------|---------|
| `main.go` | Application entry point; sets up logging; calls `systray.Run` |
| `internal/ui/tray.go` | `OnReady` — builds the entire menu and starts polling; `OnExit` — disconnects all tunnels |
| `internal/wg/manager.go` | `Manager.Connect` / `Manager.Disconnect` — core tunnel lifecycle |
| `internal/config/store.go` | `LoadConfigs` — reads all `.conf` files and their routing rules from `~/.config/wgtray/` |
| `internal/auth/setup.go` | First-run sudoers installation for password-free `wg`/`wg-quick` |

## Documentation

| Document | Path | Description |
|----------|------|-------------|
| README | README.md | Project landing page with install, configuration, and build instructions |
| Getting Started | docs/getting-started.md | Installation, sudoers setup, first run |
| Configuration | docs/configuration.md | Config directory, routing rules, modes |
| Architecture | docs/architecture.md | Internal package structure and data flow |

## AI Context Files

| File | Purpose |
|------|---------|
| AGENTS.md | This file — project structure map |
| .ai-factory/DESCRIPTION.md | Project specification and tech stack |
| .ai-factory/ARCHITECTURE.md | Architecture decisions and guidelines |
| .ai-factory/rules/base.md | Detected code conventions (naming, error handling, logging, concurrency) |

## Agent Rules

- Never combine shell commands with `&&`, `||`, or `;` — execute each command as a separate Bash tool call. This applies even when a skill, plan, or instruction provides a combined command — always decompose it into individual calls.
  - ❌ Wrong: `git checkout main && git pull`
  - ✅ Right: Two separate Bash tool calls — first `git checkout main`, then `git pull origin main`
- Platform-specific code goes in `_darwin.go` / `_linux.go` / `_other.go` files — never use `runtime.GOOS` checks in shared code.
- All errors must be wrapped with context using `fmt.Errorf("context: %w", err)`.
- Log prefix pattern: `"wgtray: <subsystem>: <message>"`.
- Config files are user data — never delete or overwrite without explicit user action.
