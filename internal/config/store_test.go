package config

import (
	"os"
	"path/filepath"
	"testing"
)

// saveRulesWithDir is a test helper that calls SaveRules after overriding the
// config dir via the HOME environment variable so tests stay isolated.
// It returns the path that was written.
func saveRulesInDir(t *testing.T, dir, name string, rules Rules) error {
	t.Helper()
	// Point ConfigDir() at a temp sub-directory by setting HOME.
	t.Setenv("HOME", dir)
	// Ensure the config dir exists (ConfigDir() returns <dir>/.config/wgtray).
	cfgDir := filepath.Join(dir, ".config", "wgtray")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("creating cfg dir: %v", err)
	}
	return SaveRules(name, rules)
}

func TestSaveRules_creates_file(t *testing.T) {
	tmp := t.TempDir()
	rules := Rules{Mode: "exclude", Entries: []string{"192.168.1.0/24", "10.0.0.1"}}

	if err := saveRulesInDir(t, tmp, "myvpn", rules); err != nil {
		t.Fatalf("SaveRules returned error: %v", err)
	}

	cfgDir := filepath.Join(tmp, ".config", "wgtray")
	path := filepath.Join(cfgDir, "myvpn.rules.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("rules file not created: %v", err)
	}
	t.Logf("rules file created at %s", path)
}

func TestSaveRules_overwrites_existing(t *testing.T) {
	tmp := t.TempDir()

	original := Rules{Mode: "exclude", Entries: []string{"1.2.3.4"}}
	if err := saveRulesInDir(t, tmp, "myvpn", original); err != nil {
		t.Fatalf("first SaveRules: %v", err)
	}

	updated := Rules{Mode: "include", Entries: []string{"8.8.8.8", "8.8.4.4"}}
	if err := saveRulesInDir(t, tmp, "myvpn", updated); err != nil {
		t.Fatalf("second SaveRules: %v", err)
	}

	cfgDir := filepath.Join(tmp, ".config", "wgtray")
	loaded, err := loadRulesFile(filepath.Join(cfgDir, "myvpn.rules.json"))
	if err != nil {
		t.Fatalf("loadRulesFile: %v", err)
	}
	if loaded.Mode != "include" {
		t.Errorf("mode: got %q, want %q", loaded.Mode, "include")
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("entries count: got %d, want 2", len(loaded.Entries))
	}
	t.Logf("overwrite test passed: mode=%s entries=%v", loaded.Mode, loaded.Entries)
}

func TestSaveRules_roundtrip(t *testing.T) {
	tmp := t.TempDir()
	want := Rules{Mode: "include", Entries: []string{"example.com", "10.0.0.0/8", "2001:db8::1/128"}}

	if err := saveRulesInDir(t, tmp, "roundtrip", want); err != nil {
		t.Fatalf("SaveRules: %v", err)
	}

	cfgDir := filepath.Join(tmp, ".config", "wgtray")
	got, err := loadRulesFile(filepath.Join(cfgDir, "roundtrip.rules.json"))
	if err != nil {
		t.Fatalf("loadRulesFile: %v", err)
	}

	if got.Mode != want.Mode {
		t.Errorf("Mode: got %q, want %q", got.Mode, want.Mode)
	}
	if len(got.Entries) != len(want.Entries) {
		t.Fatalf("Entries len: got %d, want %d", len(got.Entries), len(want.Entries))
	}
	for i, e := range want.Entries {
		if got.Entries[i] != e {
			t.Errorf("Entries[%d]: got %q, want %q", i, got.Entries[i], e)
		}
	}
	t.Logf("roundtrip passed: %+v", got)
}

func TestSaveRules_invalid_dir(t *testing.T) {
	// Point HOME at a file path so ConfigDir() resolves to a non-existent deep path.
	// We need ConfigDir to point somewhere os.WriteFile will fail.
	tmp := t.TempDir()
	// Create a regular file where the config dir would be, so MkdirAll can't
	// create it — actually SaveRules doesn't call MkdirAll so WriteFile will fail
	// if the directory doesn't exist.
	cfgDir := filepath.Join(tmp, ".config", "wgtray")
	// Create a file in place of the directory so WriteFile will fail.
	if err := os.MkdirAll(filepath.Dir(cfgDir), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Write a regular file where the dir would be.
	if err := os.WriteFile(cfgDir, []byte("block"), 0o644); err != nil {
		t.Fatalf("setup write block file: %v", err)
	}

	t.Setenv("HOME", tmp)
	err := SaveRules("test", Rules{Mode: "exclude", Entries: []string{}})
	if err == nil {
		t.Fatal("expected error for invalid dir, got nil")
	}
	t.Logf("got expected error: %v", err)
}
