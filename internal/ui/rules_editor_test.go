//go:build darwin

package ui

import (
	"strings"
	"testing"

	"wgtray/internal/config"
)

// ---- formatRulesSummary / formatMode ----

func TestFormatRulesList_empty(t *testing.T) {
	rules := config.Rules{Mode: "exclude", Entries: []string{}}
	summary := formatRulesSummary(rules)
	if !strings.Contains(summary, "(no rules)") {
		t.Errorf("expected '(no rules)' in summary, got: %q", summary)
	}
	t.Logf("summary: %s", summary)
}

func TestFormatRulesList_multiple(t *testing.T) {
	rules := config.Rules{Mode: "exclude", Entries: []string{"10.0.0.0/8", "example.com", "1.2.3.4"}}
	summary := formatRulesSummary(rules)
	for i, entry := range rules.Entries {
		expected := strings.Contains(summary, entry)
		if !expected {
			t.Errorf("entry %d %q not found in summary", i+1, entry)
		}
	}
	// Verify line numbering is present.
	if !strings.Contains(summary, "1.") {
		t.Errorf("expected numbered list starting with '1.' in summary, got: %q", summary)
	}
	t.Logf("summary: %s", summary)
}

func TestFormatMode_exclude(t *testing.T) {
	got := formatMode("exclude")
	if !strings.Contains(got, "EXCLUDE") {
		t.Errorf("expected EXCLUDE in label, got %q", got)
	}
	if !strings.Contains(got, "blacklist") {
		t.Errorf("expected 'blacklist' in label, got %q", got)
	}
}

func TestFormatMode_include(t *testing.T) {
	got := formatMode("include")
	if !strings.Contains(got, "INCLUDE") {
		t.Errorf("expected INCLUDE in label, got %q", got)
	}
	if !strings.Contains(got, "whitelist") {
		t.Errorf("expected 'whitelist' in label, got %q", got)
	}
}

// ---- applescriptEscape ----

func TestApplescriptEscape_double_quotes(t *testing.T) {
	got := applescriptEscape(`say "hello"`)
	if strings.Contains(got, `"hello"`) {
		// raw double quotes should have been escaped
		t.Errorf("unescaped double quotes remain in: %q", got)
	}
	if !strings.Contains(got, `\"hello\"`) {
		t.Errorf("expected escaped quotes in: %q", got)
	}
}

func TestApplescriptEscape_backslash(t *testing.T) {
	got := applescriptEscape(`path\to\file`)
	if !strings.Contains(got, `\\`) {
		t.Errorf("expected escaped backslash in: %q", got)
	}
}

func TestApplescriptEscape_no_special(t *testing.T) {
	input := "192.168.1.0/24"
	got := applescriptEscape(input)
	if got != input {
		t.Errorf("plain string should be unchanged: got %q", got)
	}
}

// ---- extractTextReturned ----

func TestExtractTextReturned_standard(t *testing.T) {
	result := "button returned:OK, text returned:192.168.1.0/24"
	got := extractTextReturned(result)
	if got != "192.168.1.0/24" {
		t.Errorf("got %q, want %q", got, "192.168.1.0/24")
	}
}

func TestExtractTextReturned_no_marker(t *testing.T) {
	result := "example.com"
	got := extractTextReturned(result)
	// Falls back to returning the whole string.
	if got != result {
		t.Errorf("got %q, want %q", got, result)
	}
}

func TestExtractTextReturned_empty_entry(t *testing.T) {
	result := "button returned:OK, text returned:"
	got := extractTextReturned(result)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---- buildListItems ----

func TestBuildListItems_empty_entries(t *testing.T) {
	rules := config.Rules{Mode: "exclude", Entries: []string{}}
	items := buildListItems(rules)
	// Must contain the 4 fixed action rows.
	if len(items) != 4 {
		t.Errorf("expected 4 items for empty entries, got %d: %v", len(items), items)
	}
	if items[0] != listItemAdd {
		t.Errorf("first item should be listItemAdd, got %q", items[0])
	}
	if items[2] != listItemApply {
		t.Errorf("third item should be listItemApply, got %q", items[2])
	}
	if items[3] != listItemSeparator {
		t.Errorf("fourth item should be listItemSeparator, got %q", items[3])
	}
}

func TestBuildListItems_regular_entry(t *testing.T) {
	rules := config.Rules{Mode: "exclude", Entries: []string{"10.0.0.0/8"}}
	items := buildListItems(rules)
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
	if !strings.Contains(items[4], "10.0.0.0/8") {
		t.Errorf("rule item should contain entry, got %q", items[4])
	}
	if strings.Contains(items[4], "wildcard") {
		t.Errorf("regular entry should not have wildcard label, got %q", items[4])
	}
}

func TestBuildListItems_wildcard_entry(t *testing.T) {
	rules := config.Rules{Mode: "exclude", Entries: []string{"*.example.com"}}
	items := buildListItems(rules)
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
	if !strings.Contains(items[4], "*.example.com") {
		t.Errorf("wildcard item should contain entry, got %q", items[4])
	}
	if !strings.Contains(items[4], "wildcard") {
		t.Errorf("wildcard entry should have wildcard label, got %q", items[4])
	}
}

// ---- itemAction ----

func TestItemAction(t *testing.T) {
	tests := []struct {
		item string
		want string
	}{
		{listItemAdd, "add"},
		{listItemApply, "apply"},
		{listItemSeparator, "separator"},
		{"[ Change Mode: EXCLUDE (blacklist) ]", "mode"},
		{"  1.  10.0.0.0/8", "rule"},
	}
	for _, tc := range tests {
		if got := itemAction(tc.item); got != tc.want {
			t.Errorf("itemAction(%q) = %q, want %q", tc.item, got, tc.want)
		}
	}
}

// ---- ruleIndexFromItem ----

func TestRuleIndexFromItem_found(t *testing.T) {
	entries := []string{"10.0.0.0/8", "example.com"}
	item := "  2.  example.com"
	got := ruleIndexFromItem(item, entries)
	if got != 1 {
		t.Errorf("ruleIndexFromItem(%q, ...) = %d, want 1", item, got)
	}
}

func TestRuleIndexFromItem_not_found(t *testing.T) {
	entries := []string{"10.0.0.0/8"}
	got := ruleIndexFromItem("  5.  missing.com", entries)
	if got != -1 {
		t.Errorf("expected -1 for unrecognised item, got %d", got)
	}
}

// ---- isValidEntry ----

func TestIsValidEntry(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"10.0.0.0/8", true},
		{"192.168.1.1", true},
		{"example.com", true},
		{"*.example.com", true},
		{"", false},
		{"not valid entry", false},
	}
	for _, tc := range tests {
		if got := isValidEntry(tc.input); got != tc.want {
			t.Errorf("isValidEntry(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
