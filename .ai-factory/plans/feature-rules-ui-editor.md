# Rules UI Editor

**Branch:** `feature/rules-ui-editor`
**Date:** 2026-04-16
**Mode:** Full

## Summary

Add a native macOS UI (via osascript AppleScript dialogs) for managing per-config routing rules directly from the tray menu. Users will be able to add, delete, and modify rule entries and switch the tunnel mode (exclude/include) through an interactive dialog — without manually editing JSON files. Saving rules while the tunnel is active will automatically reconnect it.

---

## Settings

- **Testing:** Yes — unit tests for new logic
- **Logging:** Verbose — detailed DEBUG logs
- **Docs:** Mandatory checkpoint at completion (route through /aif-docs)

---

## Roadmap Linkage

- **Milestone:** none
- **Rationale:** Skipped by user — no roadmap file exists in this project

---

## Architecture Context

Follows the layered architecture convention:

```
ui (tray.go / rules_editor_darwin.go)
  └── config (store.go)   — save rules
  └── wg (manager.go)     — disconnect + reconnect
```

Platform-specific UI code goes in `_darwin.go`; non-Darwin stub in `_other.go`.
No new external dependencies — all dialogs use `osascript` (already used in `addConfig()`).

---

## Tasks

### Phase 1 — Config layer: save rules

#### ~~Task 1~~ ✅ — Add `SaveRules` to `internal/config/store.go`

**Deliverable:** A new exported function `SaveRules(name string, rules Rules) error` that atomically writes the rules JSON file for the named config.

**Files to change:**
- `internal/config/store.go` — add `SaveRules`

**Implementation details:**
- Marshal `rules` to indented JSON (`json.MarshalIndent`)
- Write to `<ConfigDir>/<name>.rules.json` with permission `0o644`
- Re-use the existing path pattern from `EnsureRulesFile`

**Logging:**
- `log.Printf("wgtray: config: saving rules for %q to %s", name, path)` before write
- `log.Printf("wgtray: config: rules saved for %q", name)` on success
- Wrap error: `fmt.Errorf("save rules %s: %w", name, err)`

---

### Phase 2 — Rules editor UI (macOS)

#### ~~Task 2~~ ✅ — Create `internal/ui/rules_editor_darwin.go`

**Deliverable:** An `openRulesEditor(name string, mgr *wg.Manager)` function that drives a multi-step AppleScript rules editor dialog.

**Files to create:**
- `internal/ui/rules_editor_darwin.go`

**Build constraint:** `//go:build darwin` — required to avoid duplicate symbol errors with `rules_editor_other.go`.

**Dialog flow:**

1. **Load current rules** — call `config.LoadConfigs()`, find the config by name, read its `Rules`.

2. **Main editor dialog** — build a text summary showing:
   - Current mode: `EXCLUDE (blacklist)` or `INCLUDE (whitelist)`
   - Numbered list of current rules (entries), or `(no rules)` if empty
   - Use `choose from list {"Add Rule", "Delete Rule", "Change Mode", "Apply & Reconnect"}` with a `Cancel button` for action selection.
   - **Important:** macOS `display dialog` supports max 3 buttons. Do NOT use `display dialog` with 5 buttons — it will fail at runtime. Use `choose from list` for the action menu instead.

3. **Add rule** — `display dialog "Enter rule (IP, CIDR, or domain):" default answer ""`
   - Trim whitespace, reject empty input, append to `rules.Entries`
   - Return to main editor dialog

4. **Delete rule** — `choose from list` showing all entries with `Select rule to delete:`
   - On selection: remove the chosen entry from `rules.Entries`
   - Return to main editor dialog

5. **Change Mode** — `choose from list {"exclude (blacklist)", "include (whitelist)"}`
   - On selection: update `rules.Mode` to `"exclude"` or `"include"`
   - Return to main editor dialog

6. **Apply & Reconnect** —
   a. Call `config.SaveRules(name, rules)` — persist to JSON
   b. Detect active tunnel using the same pattern as `toggleTunnel`: `mgr.IsActive(name) || wg.InterfaceForConfig(cfg.FilePath) != ""`
   c. If active: call `mgr.Disconnect(name)`, reload config from disk via `config.LoadConfigs()`, then `mgr.Connect(updatedCfg)`
   d. Call `notify.Info("Rules applied", name)`
   e. Call `doRefresh()` to update menu state
   f. Close dialog

   **Critical:** Always reload config from disk after saving (not from in-memory struct) — ensures include-mode temp configs are rebuilt with new rules.

7. **Cancel** — discard in-memory changes, return without saving

**Logging:**
- `log.Printf("wgtray: ui: rules editor opened for %q", name)` on entry
- `log.Printf("wgtray: ui: add rule %q to %q", entry, name)` when adding
- `log.Printf("wgtray: ui: delete rule %q from %q", entry, name)` when deleting
- `log.Printf("wgtray: ui: mode changed to %q for %q", mode, name)` on mode change
- `log.Printf("wgtray: ui: applying rules for %q, active=%v", name, isActive)` before apply
- `log.Printf("wgtray: ui: rules editor closed without applying for %q", name)` on cancel
- Wrap all errors: `fmt.Errorf("rules editor %s: %w", name, err)`

**osascript helper:**
Add a private helper `runAppleScript(script string) (string, error)` that wraps `exec.Command("osascript", "-e", script)`. Reuse for all dialogs.

**Concurrency guard:**
Add a package-level `var editorMu sync.Mutex` (or per-slot `atomic.Bool`) to prevent opening multiple editor dialogs for the same config simultaneously. Check/acquire before entering the dialog loop, release on exit.

---

#### ~~Task 3~~ ✅ — Create `internal/ui/rules_editor_other.go`

**Deliverable:** A stub `openRulesEditor(name string, mgr *wg.Manager)` for non-Darwin platforms that logs the rules file path.

**Files to create:**
- `internal/ui/rules_editor_other.go`

**Implementation details:**
- Build constraint: `//go:build !darwin`
- Call `config.EnsureRulesFile(name)` to get path
- Log: `log.Printf("wgtray: ui: rules editor not available on this platform, edit manually: %s", path)`
- Do NOT use `exec.Command("open", "-e", ...)` — `open -e` is macOS-only and will fail on Linux. Linux support is partial; a simple log message is the correct fallback.

---

### Phase 3 — Wire into tray

#### ~~Task 4~~ ✅ — Update `internal/ui/tray.go`

**Deliverable:** Replace the call to `openRulesFile(name)` with `openRulesEditor(name, mgr)` in `watchSlot`.

**Files to change:**
- `internal/ui/tray.go`

**Implementation details:**
- In `watchSlot`, find the `case <-s.editItem.ClickedCh:` block
- Replace `openRulesFile(name)` → `go openRulesEditor(name, mgr)` (must run in a separate goroutine so the slot's `select` loop remains responsive to `mainItem.ClickedCh` toggle clicks while the dialog is open)
- Remove the now-unused `openRulesFile` function (if nothing else references it)
- Update submenu tooltip: `"Edit routing rules for this config"` → `"Open rules editor for this config"`

**Logging:**
- `log.Printf("wgtray: ui: edit rules clicked for %q", name)` before calling `openRulesEditor`

---

### Phase 4 — Tests

#### ~~Task 5~~ ✅ — Unit tests for `config.SaveRules`

**Deliverable:** Test file `internal/config/store_test.go` covering the new `SaveRules` function.

**Files to create:**
- `internal/config/store_test.go`

**Test cases:**
- `TestSaveRules_creates_file` — SaveRules on a fresh dir writes a valid JSON file
- `TestSaveRules_overwrites_existing` — SaveRules on an existing file replaces content
- `TestSaveRules_roundtrip` — SaveRules then loadRulesFile returns identical struct
- `TestSaveRules_invalid_dir` — SaveRules to a bad path returns wrapped error

**Logging:** Use `t.Logf` for test diagnostics.

---

#### ~~Task 6~~ ✅ — Unit tests for AppleScript helpers

**Deliverable:** Test file `internal/ui/rules_editor_test.go` covering pure logic functions (non-osascript parts).

**Files to create:**
- `internal/ui/rules_editor_test.go`

**Test cases (no osascript invocations — test pure logic):**
- `TestFormatRulesList_empty` — empty entries formats as `(no rules)`
- `TestFormatRulesList_multiple` — list of entries formats as numbered lines
- `TestParseMode_exclude` / `_include` — mode string ↔ display string conversion

**Note:** osascript dialog loops are not unit-testable; extract formatting/parsing helpers as unexported functions and test those.

---

### Phase 5 — Docs

#### ~~Task 7~~ ✅ — Update documentation

**Deliverable:** Update `docs/configuration.md` to describe the new UI rules editor.

**Files to change:**
- `docs/configuration.md`

**Content to add:**
- Section: `## Rules Editor UI`
- Explain how to open: `Edit Rules…` in the tray menu per config
- Describe dialog flow: Add / Delete / Change Mode / Apply & Reconnect
- Describe mode semantics: exclude = blacklist, include = whitelist
- Note that Apply & Reconnect automatically re-establishes an active tunnel

---

## Commit Plan

### Checkpoint 1 — after Tasks 1–2 (config + editor core)
```
feat(config): add SaveRules function for persisting routing rules
feat(ui): add osascript rules editor dialog (darwin)
```

### Checkpoint 2 — after Tasks 3–4 (stub + tray wire-up)
```
feat(ui): add rules editor stub for non-darwin platforms
feat(ui): wire rules editor into tray slot click handler
```

### Checkpoint 3 — after Tasks 5–6 (tests)
```
test(config): unit tests for SaveRules
test(ui): unit tests for rules editor formatting helpers
```

### Checkpoint 4 — after Task 7 (docs)
```
docs: document rules editor UI in configuration.md
```

---

## Implementation Notes

- **Reconnect logic:** When applying rules and the tunnel is active, load fresh config via `config.LoadConfigs()` after saving so the `Config` struct passed to `mgr.Connect` has up-to-date rules. Detect active tunnels using `mgr.IsActive(name) || wg.InterfaceForConfig(cfg.FilePath) != ""` — same pattern as `toggleTunnel` — to handle externally-started tunnels.
- **Dialog loop:** The main editor runs in a `for` loop until user selects Apply or Cancel — use a `bool applied` flag to exit cleanly.
- **osascript limits:** macOS `display dialog` supports max 3 buttons — use `choose from list` for action menus with 4+ options.
- **osascript escaping:** Escape double quotes in dynamic content (rule entries) before embedding in AppleScript strings using `strings.ReplaceAll(s, `"`, `\\"`)`. Also escape backslashes.
- **Goroutine + guard:** `openRulesEditor` must be launched with `go` from `watchSlot` so the slot's select loop stays responsive. Use a `sync.Mutex` or `atomic.Bool` guard to prevent duplicate editor dialogs for the same config.
- **Thread safety:** `openRulesEditor` runs in its own goroutine; `doRefresh()` is safe to call directly (already called from multiple goroutines in the polling loop and slot handlers).
- **Config reload:** After saving rules, always reload the config from disk (not from the in-memory struct) before reconnecting — ensures temp configs from include mode are rebuilt correctly.
