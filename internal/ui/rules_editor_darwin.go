//go:build darwin

package ui

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"wgtray/internal/config"
	"wgtray/internal/notify"
	"wgtray/internal/wg"
)

// editorMu prevents opening multiple rules editor dialogs simultaneously.
var editorMu sync.Mutex

// List item constants for the main rules editor screen.
const (
	listItemAdd       = "[ Add New Rule ]"
	listItemApply     = "[ Apply & Reconnect ]"
	listItemSeparator = "────────────────────────────────"
)

// listItemMode formats the dynamic mode action row.
func listItemMode(rules config.Rules) string {
	return "[ Change Mode: " + formatMode(rules.Mode) + " ]"
}

// openRulesEditor opens the list-centric AppleScript rules editor for the
// named config. It is safe to call from any goroutine.
func openRulesEditor(name string, mgr *wg.Manager) {
	if !editorMu.TryLock() {
		log.Printf("wgtray: ui: rules editor already open, ignoring request for %q", name)
		return
	}
	defer editorMu.Unlock()

	log.Printf("wgtray: ui: rules editor opened for %q", name)

	// Load current rules from disk.
	cfgs, err := config.LoadConfigs()
	if err != nil {
		log.Printf("wgtray: ui: rules editor %s: load configs: %v", name, err)
		notify.Error("Rules editor error", fmt.Sprintf("%s: %v", name, err))
		return
	}

	var currentCfg *config.Config
	for i := range cfgs {
		if cfgs[i].Name == name {
			currentCfg = &cfgs[i]
			break
		}
	}
	if currentCfg == nil {
		log.Printf("wgtray: ui: rules editor: config %q not found", name)
		notify.Error("Rules editor error", fmt.Sprintf("config %q not found", name))
		return
	}

	rules := currentCfg.Rules
	if rules.Entries == nil {
		rules.Entries = []string{}
	}

	// Main editor loop — re-shows the list after every action.
	for {
		items := buildListItems(rules)
		log.Printf("wgtray: ui: rules editor list shown for %q, %d entries", name, len(rules.Entries))

		var quoted []string
		for _, it := range items {
			quoted = append(quoted, applescriptQuoteItem(it))
		}
		listLiteral := "{" + strings.Join(quoted, ", ") + "}"

		script := fmt.Sprintf(
			`choose from list %s `+
				`with title "Rules Editor — %s" `+
				`with prompt "Mode: %s\n\nSelect an action or click a rule to edit:" `+
				`OK button name "Select" cancel button name "Done"`,
			listLiteral,
			applescriptEscape(name),
			applescriptEscape(formatMode(rules.Mode)),
		)

		result, err := runAppleScript(script)
		if err != nil || strings.TrimSpace(result) == "false" {
			log.Printf("wgtray: ui: rules editor closed without applying for %q", name)
			return
		}

		item := strings.TrimSpace(result)
		log.Printf("wgtray: ui: rules editor item selected: %q", item)

		switch itemAction(item) {
		case "separator":
			// Separator was accidentally selected — re-show list.
			continue

		case "add":
			entry, ok := editRuleEntry(name, "")
			if !ok {
				continue
			}
			if !isValidEntry(entry) {
				log.Printf("wgtray: ui: invalid entry rejected: %q", entry)
				showAlert(fmt.Sprintf("Invalid entry: %q\nExpected IP, CIDR, domain, or *.domain.", entry))
				continue
			}
			log.Printf("wgtray: ui: add rule %q to %q", entry, name)
			rules.Entries = append(rules.Entries, entry)

		case "mode":
			if rules.Mode == "include" {
				rules.Mode = "exclude"
			} else {
				rules.Mode = "include"
			}
			log.Printf("wgtray: ui: mode changed to %q for %q", rules.Mode, name)

		case "apply":
			applyRules(name, rules, currentCfg.FilePath, mgr)
			return

		case "rule":
			idx := ruleIndexFromItem(item, rules.Entries)
			if idx < 0 {
				log.Printf("wgtray: ui: rules editor: could not find entry for item %q", item)
				continue
			}
			oldEntry := rules.Entries[idx]
			rules = handleRuleAction(name, rules, idx, oldEntry)
		}
	}
}

// handleRuleAction shows the Edit|Delete submenu for a selected rule and
// applies the chosen action.
func handleRuleAction(name string, rules config.Rules, idx int, entry string) config.Rules {
	script := fmt.Sprintf(
		`choose from list {"Edit", "Delete"} `+
			`with title "Rule: %s" `+
			`with prompt "Choose action for rule:" `+
			`OK button name "Select" cancel button name "Cancel"`,
		applescriptEscape(entry),
	)
	result, err := runAppleScript(script)
	if err != nil || strings.TrimSpace(result) == "false" {
		return rules
	}

	action := strings.TrimSpace(result)
	log.Printf("wgtray: ui: rule action %q for entry [%d] %q in %q", action, idx, entry, name)

	switch action {
	case "Edit":
		newEntry, ok := editRuleEntry(name, entry)
		if !ok {
			return rules
		}
		if !isValidEntry(newEntry) {
			log.Printf("wgtray: ui: invalid entry rejected: %q", newEntry)
			showAlert(fmt.Sprintf("Invalid entry: %q\nExpected IP, CIDR, domain, or *.domain.", newEntry))
			return rules
		}
		log.Printf("wgtray: ui: edit rule [%d] %q → %q", idx, entry, newEntry)
		rules.Entries[idx] = newEntry

	case "Delete":
		log.Printf("wgtray: ui: delete rule [%d] %q from %q", idx, entry, name)
		rules.Entries = append(rules.Entries[:idx], rules.Entries[idx+1:]...)
	}

	return rules
}

// buildListItems constructs the full items slice for the main choose-from-list
// dialog: action rows, a visual separator, then numbered rule entries.
func buildListItems(rules config.Rules) []string {
	items := []string{
		listItemAdd,
		listItemMode(rules),
		listItemApply,
		listItemSeparator,
	}
	for i, e := range rules.Entries {
		label := fmt.Sprintf("  %d.  %s", i+1, e)
		if strings.HasPrefix(e, "*.") {
			label += "  「wildcard」"
		}
		items = append(items, label)
	}
	return items
}

// itemAction returns the semantic action for a selected list item string.
// Returns one of: "add", "mode", "apply", "separator", "rule".
func itemAction(item string) string {
	switch {
	case item == listItemAdd:
		return "add"
	case strings.HasPrefix(item, "[ Change Mode:"):
		return "mode"
	case item == listItemApply:
		return "apply"
	case item == listItemSeparator:
		return "separator"
	default:
		return "rule"
	}
}

// ruleIndexFromItem parses an item string of the form "  N.  <entry>[  「wildcard」]"
// and returns the 0-based index into entries. Returns -1 when not found.
func ruleIndexFromItem(item string, entries []string) int {
	trimmed := strings.TrimSpace(item)
	for i, e := range entries {
		expected := fmt.Sprintf("%d.  %s", i+1, e)
		// The item may have a trailing wildcard indicator.
		if trimmed == expected || strings.HasPrefix(trimmed, expected) {
			return i
		}
	}
	return -1
}

// editRuleEntry shows a pre-filled display dialog for entering or editing a
// rule. Returns (entry, true) on OK, or ("", false) on Cancel.
func editRuleEntry(name, current string) (string, bool) {
	prompt := "Enter rule (IP, CIDR, domain, or *.domain):"
	script := fmt.Sprintf(
		`display dialog %s default answer %s with title "Edit Rule"`,
		applescriptQuote(prompt),
		applescriptQuote(current),
	)
	result, err := runAppleScript(script)
	if err != nil {
		log.Printf("wgtray: ui: edit rule entry cancelled for %q", name)
		return "", false
	}
	entry := strings.TrimSpace(extractTextReturned(result))
	log.Printf("wgtray: ui: edit rule entry input %q for %q", entry, name)
	return entry, true
}

// isValidEntry returns true when s is a plausible routing rule: an IP address,
// CIDR block, bare domain, or wildcard domain (*.example.com).
func isValidEntry(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Reject entries that contain whitespace.
	if strings.ContainsAny(s, " \t") {
		return false
	}
	// Valid CIDR?
	if _, _, err := net.ParseCIDR(s); err == nil {
		return true
	}
	// Valid bare IP?
	if net.ParseIP(s) != nil {
		return true
	}
	// Wildcard domain: *.something.tld
	if strings.HasPrefix(s, "*.") {
		return isValidDomain(s[2:])
	}
	// Bare domain.
	return isValidDomain(s)
}

// domainRe matches a hostname/domain made of dot-separated alphanumeric labels.
var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// isValidDomain reports whether s is a syntactically valid domain name.
func isValidDomain(s string) bool {
	return len(s) > 0 && len(s) <= 253 && domainRe.MatchString(s)
}

// applyRules saves rules and reconnects if the tunnel is active.
func applyRules(name string, rules config.Rules, cfgFilePath string, mgr *wg.Manager) {
	if err := config.SaveRules(name, rules); err != nil {
		log.Printf("wgtray: ui: rules editor %s: save rules: %v", name, err)
		notify.Error("Rules save failed", fmt.Sprintf("%s: %v", name, err))
		return
	}

	isActive := mgr.IsActive(name) || wg.InterfaceForConfig(cfgFilePath) != ""
	log.Printf("wgtray: ui: applying rules for %q, active=%v", name, isActive)

	if isActive {
		if err := mgr.Disconnect(name); err != nil {
			log.Printf("wgtray: ui: rules editor %s: disconnect: %v", name, err)
			notify.Error("Disconnect failed", fmt.Sprintf("%s: %v", name, err))
			return
		}

		// Reload config from disk so new rules are picked up.
		cfgs, err := config.LoadConfigs()
		if err != nil {
			log.Printf("wgtray: ui: rules editor %s: reload configs: %v", name, err)
			notify.Error("Reload failed", fmt.Sprintf("%s: %v", name, err))
			return
		}
		var updatedCfg *config.Config
		for i := range cfgs {
			if cfgs[i].Name == name {
				updatedCfg = &cfgs[i]
				break
			}
		}
		if updatedCfg == nil {
			log.Printf("wgtray: ui: rules editor: config %q not found after reload", name)
			notify.Error("Config not found", name)
			return
		}

		if err := mgr.Connect(*updatedCfg); err != nil {
			log.Printf("wgtray: ui: rules editor %s: reconnect: %v", name, err)
			notify.Error("Reconnect failed", fmt.Sprintf("%s: %v", name, err))
			return
		}
	}

	notify.Info("Rules applied", name)
	doRefresh()
}

// showAlert displays a simple informational alert via osascript.
func showAlert(msg string) {
	script := fmt.Sprintf(`display alert %s`, applescriptQuote(msg))
	runAppleScript(script) //nolint:errcheck
}

// runAppleScript executes an AppleScript one-liner and returns its output.
func runAppleScript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// formatRulesSummary builds a human-readable summary of the current rules.
func formatRulesSummary(rules config.Rules) string {
	modeLabel := formatMode(rules.Mode)
	var sb strings.Builder
	sb.WriteString("Mode: ")
	sb.WriteString(modeLabel)
	sb.WriteString("\n\nRules:\n")
	if len(rules.Entries) == 0 {
		sb.WriteString("  (no rules)")
	} else {
		for i, e := range rules.Entries {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, e))
		}
	}
	return sb.String()
}

// formatMode converts a rules mode string to a display label.
func formatMode(mode string) string {
	if mode == "include" {
		return "INCLUDE (whitelist)"
	}
	return "EXCLUDE (blacklist)"
}

// applescriptEscape escapes a string for safe embedding inside an AppleScript
// double-quoted string. It escapes backslashes and double-quotes.
func applescriptEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// applescriptQuote wraps s in AppleScript double quotes, escaping as needed.
func applescriptQuote(s string) string {
	return `"` + applescriptEscape(s) + `"`
}

// applescriptQuoteItem wraps a list item value in AppleScript double quotes.
func applescriptQuoteItem(s string) string {
	return `"` + applescriptEscape(s) + `"`
}

// extractTextReturned parses "button returned:OK, text returned:<value>" from
// an AppleScript display dialog result.
func extractTextReturned(result string) string {
	const marker = "text returned:"
	idx := strings.Index(result, marker)
	if idx < 0 {
		return result
	}
	return result[idx+len(marker):]
}
