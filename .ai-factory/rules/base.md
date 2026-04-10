# Project Base Rules

> Auto-detected conventions from codebase analysis. Edit as needed.

## Naming Conventions

- **Files:** `snake_case.go`, platform variants use `_darwin.go` / `_linux.go` / `_other.go` suffixes
- **Packages:** lowercase, single word (`config`, `wg`, `ui`, `notify`, `auth`, `icon`)
- **Types/Structs:** `PascalCase` (e.g. `TunnelState`, `Manager`, `Config`, `Rules`)
- **Functions/Methods:** `PascalCase` for exported, `camelCase` for unexported (e.g. `NewManager`, `loadRulesFile`, `shellQuote`)
- **Variables:** `camelCase` (e.g. `cfgPath`, `upConfigPath`, `anyConnected`)
- **Constants:** `camelCase` for unexported (e.g. `maxSlots`)

## Module Structure

- `main.go` — entry point only; no business logic
- `internal/config/` — data types and file I/O; no OS-level side effects
- `internal/wg/` — WireGuard tunnel management; system commands; platform-split files
- `internal/auth/` — authentication; platform-split files (`_darwin`, `_other`)
- `internal/notify/` — user notifications; platform-split files
- `internal/ui/` — systray UI layer; depends on config and wg
- `icon/` — embedded assets only

## Error Handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Log errors with `log.Printf("wgtray: <subsystem>: %v", err)` before returning or surfacing
- Non-fatal errors surfaced to user via `notify.Error(title, msg)` — never crash
- `//nolint:errcheck` used only for fire-and-forget OS calls (e.g. `exec.Command.Start()`)

## Logging

- Standard library `log` package; file-based output to `~/.config/wgtray/wgtray.log`
- Log prefix pattern: `"wgtray: <subsystem>: <message>"` (e.g. `"wgtray: connect %s: %v"`)
- Log level: all logs go to the same file; no structured logging or log levels

## Concurrency

- `sync.RWMutex` on the `Manager` struct for tunnel state; RLock for reads, Lock for writes
- `sync.Mutex` on `slot` structs in the UI for per-slot name access
- Goroutines spawned for: slot click watchers, static menu item watcher, polling ticker
- Systray UI methods (`SetTitle`, `Show`, `Hide`, `SetIcon`, `SetTooltip`) called outside locks

## Platform Targeting

- macOS is the primary target; Linux support is partial (stub files with `_other.go` / `_linux.go`)
- Use Go build tags or file name suffixes for platform-specific code, not runtime `runtime.GOOS` checks
- macOS-only features: Touch ID, osascript admin execution, `route add/delete`, `.icns` icons

## Build

- `make build` — produces `./wgtray` binary (`go build -ldflags="-s -w"`)
- `make bundle` — produces `WGTray.app` macOS app bundle
- `make install` — copies `WGTray.app` to `/Applications`
- `make clean` — removes binary and app bundle
