---
name: go-best-practices
description: >-
  Go idioms, code conventions, and best practices tailored for the WGTray codebase.
  Covers error handling, concurrency (sync.RWMutex, goroutines), platform-specific
  file split patterns, systray UI patterns, and macOS admin execution. Use when
  writing, reviewing, or refactoring any Go code in this project.
argument-hint: "[topic: errors|concurrency|platform|ui|admin]"
allowed-tools: Read Grep Glob
metadata:
  author: aif-skill-generator
  version: "1.0"
  category: go
---

# Go Best Practices — WGTray

Reference conventions detected from the codebase. All code in this project MUST follow these patterns.

See [references/CONVENTIONS.md](references/CONVENTIONS.md) for extended examples.

---

## Error Handling

Always wrap errors with context using `%w` for unwrapping support:

```go
// ✅ Correct
if err != nil {
    return fmt.Errorf("connect %s: %w", cfg.Name, err)
}

// ❌ Wrong — no context
if err != nil {
    return err
}
```

Log errors before surfacing to the user. Use the `wgtray: <subsystem>: <message>` prefix:

```go
log.Printf("wgtray: load configs: %v", err)
notify.Error("Config error", err.Error())
```

Use `//nolint:errcheck` only for fire-and-forget OS calls where failure is intentionally ignored:

```go
exec.Command("open", config.ConfigDir()).Start() //nolint:errcheck
```

---

## Concurrency

Use `sync.RWMutex` for shared state with mixed read/write access:

```go
type Manager struct {
    mu     sync.RWMutex
    active map[string]*TunnelState
}

// Read: RLock/RUnlock
func (m *Manager) IsActive(name string) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    _, ok := m.active[name]
    return ok
}

// Write: Lock/Unlock
func (m *Manager) Connect(cfg config.Config) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ...
}
```

Rules:
- Always `defer` unlock immediately after locking — never unlock manually
- Never call systray UI methods (`SetTitle`, `Show`, `Hide`, `SetIcon`) while holding a lock
- Use per-item mutexes (e.g. `slot.mu`) to protect individual items, not the whole collection
- Goroutines spawned in `OnReady` run for the app lifetime — no cancellation needed

---

## Platform-Specific Code

Split platform code using Go's filename convention — **never** use `runtime.GOOS` in shared code:

| Suffix | Target |
|--------|--------|
| `_darwin.go` | macOS only |
| `_linux.go` | Linux only |
| `_other.go` | All other platforms |

Each file in a platform-specific pair must implement the same function signatures so shared code compiles on all platforms.

```
wg/
  admin_darwin.go   // runAsAdmin, GetDefaultGateway — real implementation
  admin_linux.go    // runAsAdmin, GetDefaultGateway — Linux stub
  wgbin_darwin.go   // wgBin, wgQuickBin constants
  wgbin_linux.go    // wgBin, wgQuickBin constants
```

---

## Package & Naming Conventions

- Packages: lowercase, single word (`config`, `wg`, `ui`, `notify`, `auth`, `icon`)
- Exported types/functions: `PascalCase` (`TunnelState`, `NewManager`, `LoadConfigs`)
- Unexported functions/vars: `camelCase` (`loadRulesFile`, `shellQuote`, `maxSlots`)
- File names: `snake_case.go` with optional platform suffix

---

## Internal Package Structure

Follow the layering rule — dependencies only go one way:

```
main → ui → wg, config, notify, icon
           → config
         wg → config
       auth → (none)
     notify → (none)
       icon → (none)
```

`internal/config` is the base layer — it must not import any other internal package.

---

## Systray UI Patterns

Pre-allocate a fixed number of menu slots (`maxSlots = 20`) in `OnReady`. Show/hide slots as configs change. Do NOT add new menu items after startup.

```go
// OnReady pattern
for i := 0; i < maxSlots; i++ {
    s := &slot{}
    s.mainItem = systray.AddMenuItem("", "")
    s.mainItem.Hide()
    slots[i] = s
    go watchSlot(s)   // goroutine per slot, lifetime = app
}
```

Polling loop (3s) drives all UI state updates — never push updates from wg/config code directly:

```go
go func() {
    t := time.NewTicker(3 * time.Second)
    defer t.Stop()
    for range t.C {
        doRefresh()
    }
}()
```

---

## Logging

Use the standard `log` package only. Output goes to `~/.config/wgtray/wgtray.log`.

```go
// Setup (main.go)
log.SetOutput(f)
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Usage (any package)
log.Printf("wgtray: connect %s: %v", name, err)
log.Println("wgtray: exited")
```

No structured logging, no log levels, no external logging libraries.

---

## Config File Permissions

Config files contain WireGuard keys — always use restrictive permissions:

```go
os.WriteFile(path, data, 0o600)   // configs: owner read/write only
os.MkdirAll(dir, 0o700)           // tmp dir: owner only
os.WriteFile(rulesPath, data, 0o644) // rules JSON: world-readable
```

---

## Build

```bash
make build    # go build -ldflags="-s -w" -o wgtray .
make bundle   # produces WGTray.app
make clean    # removes binary and app bundle
```

Run `go vet ./...` before committing. No test suite currently exists.
