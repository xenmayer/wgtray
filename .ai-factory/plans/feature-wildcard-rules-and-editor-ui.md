# Feature: Wildcard Domain Support + Rules Editor UI Redesign

**Branch:** `feature/wildcard-rules-and-editor-ui`
**Created:** 2026-04-16
**Type:** Enhancement

---

## Settings

- **Testing:** Yes — unit tests for wildcard helpers and updated editor helpers
- **Logging:** Verbose — `log.Printf("wgtray: <subsystem>: ...")` at every significant step
- **Docs:** Mandatory — `docs/configuration.md` must be updated as a hard checkpoint

---

## Description

Two related improvements to the routing-rules system:

1. **Wildcard domain support** — Users can enter `*.domain.com` as an entry to express intent
   to route all traffic for a domain and its subdomains. At connect time, `ResolveEntries`
   strips the `*.` prefix and resolves the base domain via DNS, returning its IPs. This is the
   best-effort approach since DNS provides no wildcard enumeration; the resolved IPs still cover
   the root domain and often CDN-shared subdomains.

2. **Redesigned Rules Editor UI** — Replace the two-step "pick action → separate dialog" flow
   with a single list-centric screen where all rules are visible at once. Users click a rule
   entry to edit or delete it inline (pre-filled `display dialog`). Action items (Add, Change
   Mode, Apply) appear as clearly labeled rows at the top of the list. This mirrors the
   interaction model of iOS/macOS Settings lists and table-based data editors.

---

## Affected Files

| File | Change |
|------|--------|
| `internal/wg/rules.go` | Add `isWildcardEntry`, `wildcardBaseDomain`; update `ResolveEntries` |
| `internal/ui/rules_editor_darwin.go` | Full UX redesign; new helpers; remove old action helpers |
| `internal/wg/rules_test.go` | **New** — tests for wildcard helpers (no build constraint — pure Go) |
| `internal/ui/rules_editor_test.go` | Add `//go:build darwin`; update tests for new helpers; remove dead tests |
| `docs/configuration.md` | Wildcard syntax docs + new editor UX description |

---

## Tasks

### Phase 1 — Wildcard Domain Resolution

#### ~~Task 1: Add wildcard support to `internal/wg/rules.go`~~ ✅

**Deliverable:** `ResolveEntries` handles `*.domain.com` entries correctly.

**Files:** `internal/wg/rules.go`

**Implementation:**

1. Add unexported helper `isWildcardEntry(s string) bool`:
   - Returns `true` when `s` has the `*.` prefix followed by at least one character
   - Example: `isWildcardEntry("*.example.com")` → `true`, `isWildcardEntry("example.com")` → `false`

2. Add unexported helper `wildcardBaseDomain(s string) string`:
   - Strips the leading `*.` from a wildcard entry
   - Example: `wildcardBaseDomain("*.example.com")` → `"example.com"`

3. In `ResolveEntries`, **after** the bare-IP branch and **before** the DNS lookup branch,
   add the wildcard check. A `*.example.com` entry won't match `ParseCIDR` or `ParseIP`,
   so placing it here avoids unnecessary calls and correctly falls through to DNS:
   ```go
   // Wildcard domain? Strip prefix and resolve base domain.
   if isWildcardEntry(e) {
       base := wildcardBaseDomain(e)
       log.Printf("wgtray: wg: wildcard entry %q → resolving base domain %q", e, base)
       e = base   // fall through to DNS lookup below
   }
   ```

**Logging requirements:**
- `log.Printf("wgtray: wg: wildcard entry %q → resolving base domain %q", entry, base)` — before DNS lookup
- Existing DNS error/success logging in `ResolveEntries` covers the rest

**No changes** to config types — wildcards are stored as plain strings in `rules.Entries`.

---

### Phase 2 — Rules Editor UI Redesign

#### ~~Task 2: Redesign `internal/ui/rules_editor_darwin.go`~~ ✅

**Deliverable:** A list-centric rules editor where all rules are visible in one `choose from list`
screen. Clicking a rule entry shows an Edit | Delete submenu. Action items (Add, Change Mode,
Apply) are clearly labeled rows at the top.

**Files:** `internal/ui/rules_editor_darwin.go`

**New UI flow:**

```
┌─────────────────────────────────────────┐
│  Rules Editor — <name>                  │
│                                         │
│  [ Add New Rule ]                       │
│  [ Change Mode: EXCLUDE (blacklist) ]   │
│  [ Apply & Reconnect ]                  │
│  ─────────────────────────────          │
│    1.  10.0.0.0/8                       │
│    2.  *.example.com  「wildcard」       │
│    3.  192.168.1.5                      │
│                         [Select] [Done] │
└─────────────────────────────────────────┘
```

When the user clicks a numbered rule entry, a second `choose from list` appears:
```
Action for "*.example.com":  Edit | Delete | Cancel
```

- **Edit** → `display dialog` pre-filled with the current entry value
  (`default answer "<current-entry>"`)
- **Delete** → removes the entry immediately (the Edit | Delete submenu already serves as confirmation — no extra alert needed)
- Entry is validated after editing via `isValidEntry` (see below)
- **Add Rule prompt** must mention wildcard syntax: `"Enter rule (IP, CIDR, domain, or *.domain):"`

**New helpers to implement:**

| Function | Signature | Purpose |
|----------|-----------|---------|
| `buildListItems` | `(rules config.Rules) []string` | Builds the full `choose from list` items slice: action rows + separator + numbered rule entries |
| `itemAction` | `(item string) string` | Returns one of `"add"`, `"mode"`, `"apply"`, `"separator"`, `"rule"` based on item prefix |
| `ruleIndexFromItem` | `(item string, entries []string) int` | Parses the `"  N.  <entry>"` format; returns 0-based index or -1 if not found |
| `editRuleEntry` | `(name, entry string) (string, bool)` | Shows a pre-filled `display dialog`; returns new value and ok=true, or old value and ok=false on cancel |
| `isValidEntry` | `(s string) bool` | Returns true for valid IPs, CIDRs, domains, and `*.domain` wildcard patterns; rejects clearly invalid input |

**Item format constants** in the same file:
```go
const (
    listItemAdd       = "[ Add New Rule ]"
    listItemSeparator = "────────────────────────────────"
    // mode item is formatted dynamically: "[ Change Mode: <formatted mode> ]"
    // apply item is constant: "[ Apply & Reconnect ]"
)
```

**Wildcard visual indicator:** Rule entries with `*.` prefix are displayed as:
```
  2.  *.example.com  「wildcard」
```

**Remove these functions** (absorbed into new flow):
- `addRule`
- `deleteRule`
- `changeMode`

**Keep these functions unchanged:**
- `applyRules`
- `runAppleScript`
- `applescriptEscape`
- `applescriptQuote`
- `applescriptQuoteItem`
- `extractTextReturned`
- `showAlert`
- `formatMode`
- `formatRulesSummary` (kept — still shown above the list and in apply confirmation)

**Main loop separator handling:**
The separator item is selectable in `choose from list`. When `itemAction` returns `"separator"`,
the main loop must `continue` without action (re-show the list).

**Logging requirements:**
- `log.Printf("wgtray: ui: rules editor list shown for %q, %d entries", name, len(rules.Entries))`
- `log.Printf("wgtray: ui: rules editor item selected: %q", item)`
- `log.Printf("wgtray: ui: edit rule [%d] %q → %q", idx, old, new)` after edit
- `log.Printf("wgtray: ui: delete rule [%d] %q from %q", idx, entry, name)` after delete
- `log.Printf("wgtray: ui: invalid entry rejected: %q", entry)` when isValidEntry fails

---

### Phase 3 — Tests

#### ~~Task 3: New `internal/wg/rules_test.go`~~ ✅

**Deliverable:** Unit tests for the wildcard helpers (no DNS, pure logic).

**Files:** `internal/wg/rules_test.go` (new file)

**Tests to write:**

| Test name | Input | Expected |
|-----------|-------|---------|
| `TestIsWildcardEntry_true` | `"*.example.com"` | `true` |
| `TestIsWildcardEntry_true_subdomain` | `"*.sub.example.com"` | `true` |
| `TestIsWildcardEntry_false_domain` | `"example.com"` | `false` |
| `TestIsWildcardEntry_false_cidr` | `"10.0.0.0/8"` | `false` |
| `TestIsWildcardEntry_false_star_only` | `"*"` | `false` |
| `TestWildcardBaseDomain_strips_prefix` | `"*.example.com"` | `"example.com"` |
| `TestWildcardBaseDomain_subdomain` | `"*.sub.example.com"` | `"sub.example.com"` |

**Build constraint:** `//go:build darwin` not required — these are pure logic tests.

---

#### ~~Task 4: Update `internal/ui/rules_editor_test.go`~~ ✅

**Deliverable:** Tests cover the new helpers; dead tests removed.

**Files:** `internal/ui/rules_editor_test.go`

**⚠️ Build constraint requirement:** The existing `rules_editor_test.go` has no build constraint,
but all functions it tests (`formatMode`, `applescriptEscape`, `extractTextReturned`) are defined
in `rules_editor_darwin.go`. The new helpers (`buildListItems`, `itemAction`, `ruleIndexFromItem`,
`isValidEntry`) are also darwin-only. To compile on non-darwin CI, **add `//go:build darwin`** at
the top of `rules_editor_test.go`. This matches the pattern: the test file tests darwin-only code.

Alternatively, extract the pure-logic helpers (`isValidEntry`, `buildListItems`, `itemAction`,
`ruleIndexFromItem`, `formatMode`, `formatRulesSummary`, `applescriptEscape`, `extractTextReturned`)
into a shared file without a build constraint — but this is a larger refactor outside the plan scope.
**Preferred approach: add `//go:build darwin` to the test file.**

**Remove tests** for deleted functions: any that referenced `addRule`, `deleteRule`, `changeMode`.

**Add/update tests:**

| Test | Purpose |
|------|---------|
| `TestBuildListItems_empty_rules` | `buildListItems(Rules{Mode:"exclude", Entries:[]})` — action rows present, no rule rows |
| `TestBuildListItems_with_rules` | Two entries present → list contains their formatted strings |
| `TestBuildListItems_wildcard_indicator` | `*.example.com` entry → item contains `「wildcard」` |
| `TestItemAction_add` | `listItemAdd` → `"add"` |
| `TestItemAction_mode` | Mode item string → `"mode"` |
| `TestItemAction_apply` | Apply item string → `"apply"` |
| `TestItemAction_separator` | Separator string → `"separator"` |
| `TestItemAction_rule` | A numbered rule item → `"rule"` |
| `TestRuleIndexFromItem_found` | Formatted entry for index 1 + entries slice → returns `1` |
| `TestRuleIndexFromItem_not_found` | Unknown item string → returns `-1` |
| `TestIsValidEntry_ip` | `"192.168.1.1"` → `true` |
| `TestIsValidEntry_cidr` | `"10.0.0.0/8"` → `true` |
| `TestIsValidEntry_domain` | `"example.com"` → `true` |
| `TestIsValidEntry_wildcard` | `"*.example.com"` → `true` |
| `TestIsValidEntry_empty` | `""` → `false` |
| `TestIsValidEntry_spaces` | `"bad entry"` → `false` |
| `TestFormatMode_exclude` | `"exclude"` → contains `"EXCLUDE"` and `"blacklist"` |
| `TestFormatMode_include` | `"include"` → contains `"INCLUDE"` and `"whitelist"` |

---

### Phase 4 — Docs

#### ~~Task 5: Update `docs/configuration.md`~~ ✅

**Deliverable:** Docs cover wildcard syntax and new rules editor UX.

**Files:** `docs/configuration.md`

**Changes:**

1. In the **Entries** section: add a row for `*.domain.com` wildcard syntax with an example and
   explanation that it resolves to the base domain's IPs.

2. Replace/update the **Rules Editor UI** section to describe the new list-centric flow:
   - Table-like list with action rows at top and numbered entries below
   - Click a rule entry to edit (pre-filled) or delete
   - Mode toggle and Apply visible as list items
   - Note: wildcard entries show a `「wildcard」` indicator

---

## Commit Plan

| After Task(s) | Commit message |
|---------------|----------------|
| 1 | `feat(wg): add wildcard domain support in ResolveEntries` |
| 2 | `feat(ui): redesign rules editor to list-centric table view with inline edit` |
| 3, 4 | `test(wg,ui): add wildcard resolver and redesigned editor helper tests` |
| 5 | `docs(configuration): document wildcard syntax and new rules editor UI` |

---

## Risk Notes

- `isWildcardEntry("*")` (star only) must return `false` — no valid base domain.
- `choose from list` items are plain strings — the separator item can be selected; `itemAction`
  must detect it and loop back without action.
- `display dialog with default answer` is the critical UX improvement for editing; ensure the
  osascript template uses `default answer` correctly.
- `isValidEntry` should use `net.ParseIP` / `net.ParseCIDR` for IP/CIDR and a minimal regex for
  domains + wildcards; avoid being overly strict (users should be able to enter any domain).
