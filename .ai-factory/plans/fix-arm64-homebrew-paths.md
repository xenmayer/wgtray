# Implementation Plan: Fix ARM64 Homebrew Binary Paths

Branch: fix/arm64-homebrew-paths
Created: 2026-04-18

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

On Apple Silicon (ARM64) Macs, Homebrew installs binaries under `/opt/homebrew/bin/` instead of `/usr/local/bin/`. WGTray currently hardcodes `/usr/local/bin/wg` and `/usr/local/bin/wg-quick` in three places, causing `wg-quick: No such file or directory (127)` errors on ARM64 machines.

**Affected files:**

| File | Issue |
|------|-------|
| `internal/wg/wgbin_darwin.go` | `wgBin` and `wgQuickBin` constants hardcoded to `/usr/local/bin/` |
| `internal/auth/setup.go` | Sudoers rule only allows `/usr/local/bin/wg-quick` |

**Root cause:** The comment in `wgbin_darwin.go` claims Homebrew symlinks ARM64 binaries to `/usr/local/bin` — this is incorrect. Homebrew on ARM64 does NOT create such symlinks by default.

**Note:** `internal/wg/admin_darwin.go` already has the correct `brewPath` constant that includes both `/opt/homebrew/bin` and `/usr/local/bin` in the PATH, but this only helps shell commands — it does not fix direct binary references.

## Tasks

### Phase 1: Runtime Binary Resolution

- [x] Task 1: Convert wgBin/wgQuickBin from constants to runtime-resolved variables

  **Deliverable:** Replace the two hardcoded `const` declarations in `internal/wg/wgbin_darwin.go` with a function that probes known Homebrew paths at startup. This is the **critical fix** — once `wgQuickBin` resolves to the correct path, tunnels work on ARM64 even with the old sudoers rule (because `runAsAdmin` wraps commands in `sudo -n sh -c "..."` and `/bin/sh` is already in the sudoers allowlist).

  **Implementation details:**
  - Create a `findBinary(name string, candidates []string) string` helper that checks `os.Stat` for each candidate path and returns the first existing one
  - Candidate paths for `wg`: `["/opt/homebrew/bin/wg", "/usr/local/bin/wg"]`
  - Candidate paths for `wg-quick`: `["/opt/homebrew/bin/wg-quick", "/usr/local/bin/wg-quick"]`
  - ARM64 Homebrew path (`/opt/homebrew/bin/`) listed first — it's the more common case on modern Macs
  - If neither path exists, fall back to just the binary name (`"wg"`, `"wg-quick"`) to let PATH resolution handle it
  - Change `wgBin`/`wgQuickBin` from `const` to package-level `var` initialized via `init()` — binary paths never change at runtime, so `init()` is sufficient (no `sync.Once` or lazy accessors needed)

  **LOGGING REQUIREMENTS:**
  - Log resolved binary paths on init: `"wgtray: wg: resolved wg binary: %s"`
  - Log a WARN if no candidate path exists and falling back to bare name: `"wgtray: wg: wg binary not found at known paths, falling back to PATH lookup"`

  **Files:** `internal/wg/wgbin_darwin.go`

- [x] Task 2: Update sudoers rule to allow both Homebrew paths

  **Deliverable:** The sudoers rule in `internal/auth/setup.go` must allow `wg` and `wg-quick` from both `/usr/local/bin/` and `/opt/homebrew/bin/`.

  **Context:** The main tunnel flow (`runAsAdmin`) wraps commands in `sudo -n sh -c "..."`, so the `/bin/sh` sudoers entry already enables it. However, `ActiveInterfaces()` calls `exec.Command("sudo", "-n", wgBin, "show", "interfaces")` **directly** — not through `sh -c` — so it needs the explicit `wg` binary entry in sudoers for the sudo fallback to work. Without this, external tunnel detection via sudo fails silently.

  **Implementation details:**
  - Update the `RunFirstTimeSetup()` function's osascript command to include both paths in the sudoers rule
  - Final rule: `%admin ALL=(ALL) NOPASSWD: /usr/local/bin/wg, /opt/homebrew/bin/wg, /usr/local/bin/wg-quick, /opt/homebrew/bin/wg-quick, /sbin/route, /bin/sh`
  - Define `const sudoersVersion = 2` to track rule format changes (used by Task 3)

  **LOGGING REQUIREMENTS:**
  - Log when an outdated sudoers rule is detected and needs updating: `"wgtray: auth: sudoers rule outdated, re-installing"`

  **Files:** `internal/auth/setup.go`

- [x] Task 3: Detect and upgrade outdated sudoers rules

  **Deliverable:** On startup, detect if the sudoers rule is outdated and trigger a one-time re-install with the updated rule.

  **Implementation details:**
  - Use a **local version marker file** at `~/.config/wgtray/.sudoers-version` instead of reading the sudoers file via `sudo -n cat` (avoids chicken-and-egg sudo dependency)
  - Add `IsSetupCurrent() bool` that reads the marker file and compares its content to `sudoersVersion` (defined in Task 2)
  - After successful `RunFirstTimeSetup()`, write the current `sudoersVersion` to the marker file
  - **CRITICAL:** Add a proactive startup check — `IsSetupDone()` is currently **never called anywhere in the codebase** (setup is purely reactive via `runAsAdmin` fallback). Add a call in `ui.OnReady()` before the first `doRefresh()`: if `IsSetupDone() && !IsSetupCurrent()`, call `RunFirstTimeSetup()` and log the result. Without this integration point, the upgrade detection would be dead code.
  - If the rule exists but is outdated, `RunFirstTimeSetup()` will overwrite it (the function already writes with `>`, not `>>`)

  **LOGGING REQUIREMENTS:**
  - Log when sudoers rule is current: `"wgtray: auth: sudoers rule is up to date"`
  - Log when sudoers rule exists but is outdated: `"wgtray: auth: sudoers rule exists but missing ARM64 paths, will re-install"`
  - Log result of proactive setup: `"wgtray: auth: sudoers rule upgraded successfully"` or `"wgtray: auth: sudoers upgrade failed: %v"`

  **Files:** `internal/auth/setup.go`, `internal/ui/tray.go`

### Phase 2: Testing

- [x] Task 4: Add tests for binary path resolution (depends on 1)

  **Deliverable:** Unit tests for the `findBinary` helper and the resolved paths.

  **Implementation details:**
  - Test that `findBinary` returns the first existing path from candidates
  - Test that `findBinary` returns the fallback bare name when no candidate exists
  - Test with a temp directory containing a fake binary to simulate both paths
  - Use build tags to ensure tests only run on darwin

  **LOGGING REQUIREMENTS:**
  - Tests should verify that log output contains expected resolution messages

  **Files:** `internal/wg/wgbin_darwin_test.go`

- [x] Task 5: Add tests for sudoers rule content (depends on 2, 3)

  **Deliverable:** Unit tests verifying the sudoers rule string contains both Homebrew paths.

  **Implementation details:**
  - Test that the generated sudoers rule includes `/opt/homebrew/bin/wg-quick`
  - Test that the generated sudoers rule includes `/usr/local/bin/wg-quick`
  - Test `IsSetupCurrent` logic against sample sudoers file content

  **Files:** `internal/auth/setup_test.go`

### Phase 3: Documentation

- [x] Task 6: Update docs and README (depends on 1, 2)

  **Deliverable:** Update documentation to reflect ARM64 support and the updated sudoers rule.

  **Implementation details:**
  - Update `docs/getting-started.md` to mention both Homebrew paths
  - Add a note about users needing to re-authenticate once if upgrading from an older version (sudoers rule update)
  - Update `README.md` if it references binary paths
  - Add CHANGELOG entry for this fix

  **Files:** `docs/getting-started.md`, `README.md`, `CHANGELOG.md`

## Commit Plan
- **Commit 1** (after tasks 1-3): `fix: resolve wg/wg-quick binary paths for ARM64 Homebrew`
- **Commit 2** (after tasks 4-5): `test: add tests for binary path resolution and sudoers rule`
- **Commit 3** (after task 6): `docs: document ARM64 Homebrew support and sudoers upgrade`
