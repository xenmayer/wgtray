//go:build darwin

package ui

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"wgtray/internal/config"
	"wgtray/internal/notify"
	"wgtray/internal/wg"
)

// editorMu prevents opening multiple rules editor dialogs simultaneously.
var editorMu sync.Mutex

// openRulesEditor opens a multi-step AppleScript dialog for editing routing
// rules for the named config. It is safe to call from any goroutine.
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

	// Main editor loop.
	for {
		summary := formatRulesSummary(rules)

		// build the action list for choose from list
		script := fmt.Sprintf(
			`choose from list {"Add Rule", "Delete Rule", "Change Mode", "Apply & Reconnect"} `+
				`with title "Rules Editor — %s" `+
				`with prompt %s `+
				`OK button name "Select" cancel button name "Cancel"`,
			applescriptEscape(name),
			applescriptQuote(summary),
		)

		result, err := runAppleScript(script)
		if err != nil || strings.TrimSpace(result) == "false" {
			// User clicked Cancel or dialog was dismissed.
			log.Printf("wgtray: ui: rules editor closed without applying for %q", name)
			return
		}

		action := strings.TrimSpace(result)
		log.Printf("wgtray: ui: rules editor action %q for %q", action, name)

		switch action {
		case "Add Rule":
			rules = addRule(name, rules)

		case "Delete Rule":
			if len(rules.Entries) == 0 {
				showAlert("No rules to delete.")
				continue
			}
			rules = deleteRule(name, rules)

		case "Change Mode":
			rules = changeMode(name, rules)

		case "Apply & Reconnect":
			applyRules(name, rules, currentCfg.FilePath, mgr)
			return
		}
	}
}

// addRule presents a dialog to enter a new rule and appends it to rules.
func addRule(name string, rules config.Rules) config.Rules {
	script := `display dialog "Enter rule (IP, CIDR, or domain):" default answer "" with title "Add Rule"`
	result, err := runAppleScript(script)
	if err != nil {
		// Cancelled.
		return rules
	}
	// result looks like: "button returned:OK, text returned:<entry>"
	entry := extractTextReturned(result)
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return rules
	}
	log.Printf("wgtray: ui: add rule %q to %q", entry, name)
	rules.Entries = append(rules.Entries, entry)
	return rules
}

// deleteRule presents a list dialog to pick a rule to remove.
func deleteRule(name string, rules config.Rules) config.Rules {
	// Build a quoted AppleScript list from entries.
	var items []string
	for _, e := range rules.Entries {
		items = append(items, applescriptQuoteItem(e))
	}
	listLiteral := "{" + strings.Join(items, ", ") + "}"

	script := fmt.Sprintf(
		`choose from list %s with title "Delete Rule" with prompt "Select rule to delete:" `+
			`OK button name "Delete" cancel button name "Cancel"`,
		listLiteral,
	)
	result, err := runAppleScript(script)
	if err != nil || strings.TrimSpace(result) == "false" {
		return rules
	}
	target := strings.TrimSpace(result)
	log.Printf("wgtray: ui: delete rule %q from %q", target, name)

	var remaining []string
	for _, e := range rules.Entries {
		if e != target {
			remaining = append(remaining, e)
		}
	}
	if remaining == nil {
		remaining = []string{}
	}
	rules.Entries = remaining
	return rules
}

// changeMode presents a list dialog to pick a new mode.
func changeMode(name string, rules config.Rules) config.Rules {
	script := `choose from list {"exclude (blacklist)", "include (whitelist)"} ` +
		`with title "Change Mode" with prompt "Select tunnel mode:" ` +
		`OK button name "Select" cancel button name "Cancel"`
	result, err := runAppleScript(script)
	if err != nil || strings.TrimSpace(result) == "false" {
		return rules
	}
	choice := strings.TrimSpace(result)
	if strings.HasPrefix(choice, "exclude") {
		rules.Mode = "exclude"
	} else {
		rules.Mode = "include"
	}
	log.Printf("wgtray: ui: mode changed to %q for %q", rules.Mode, name)
	return rules
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

// formatRulesSummary builds the prompt text shown in the main editor dialog.
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
	sb.WriteString("\nChoose an action:")
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
