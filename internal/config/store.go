package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Rules defines the routing rules for a WireGuard tunnel.
type Rules struct {
	Mode    string   `json:"mode"`    // "exclude" or "include"
	Entries []string `json:"entries"` // IPs, CIDRs, or domain names
}

// Config holds a WireGuard config and its associated routing rules.
type Config struct {
	Name     string
	FilePath string
	Rules    Rules
}

// ConfigDir returns the path to the wgtray configuration directory.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/wgtray"
	}
	return filepath.Join(home, ".config", "wgtray")
}

// EnsureConfigDir creates the config directory if it does not exist.
func EnsureConfigDir() error {
	return os.MkdirAll(ConfigDir(), 0o755)
}

// LoadConfigs reads all *.conf files from the config directory.
func LoadConfigs() ([]Config, error) {
	dir := ConfigDir()
	if err := EnsureConfigDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var configs []Config
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".conf")
		cfg := Config{
			Name:     name,
			FilePath: filepath.Join(dir, e.Name()),
		}

		rulesPath := filepath.Join(dir, name+".rules.json")
		if r, err := loadRulesFile(rulesPath); err == nil {
			cfg.Rules = r
		} else {
			cfg.Rules = Rules{Mode: "exclude", Entries: []string{}}
		}

		configs = append(configs, cfg)
	}
	return configs, nil
}

func loadRulesFile(path string) (Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Rules{}, err
	}
	var r Rules
	if err := json.Unmarshal(data, &r); err != nil {
		return Rules{}, err
	}
	return r, nil
}

// EnsureRulesFile creates a default rules file if none exists and returns its path.
func EnsureRulesFile(name string) (string, error) {
	path := filepath.Join(ConfigDir(), name+".rules.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		r := Rules{Mode: "exclude", Entries: []string{}}
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
	}
	return path, nil
}

// CopyConfigFile copies src into the config directory as <name>.conf.
func CopyConfigFile(src, name string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(ConfigDir(), name+".conf")
	return os.WriteFile(dst, data, 0o600)
}

// SaveRules persists rules for the named config to <ConfigDir>/<name>.rules.json.
func SaveRules(name string, rules Rules) error {
	path := filepath.Join(ConfigDir(), name+".rules.json")
	log.Printf("wgtray: config: saving rules for %q to %s", name, path)

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("save rules %s: %w", name, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save rules %s: %w", name, err)
	}

	log.Printf("wgtray: config: rules saved for %q", name)
	return nil
}
