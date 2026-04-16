//go:build !darwin

package ui

import (
	"log"

	"wgtray/internal/config"
	"wgtray/internal/wg"
)

// openRulesEditor is a stub for non-Darwin platforms.
// The interactive rules editor requires macOS osascript and is not available here.
// The path to the rules JSON file is logged so the user can edit it manually.
func openRulesEditor(name string, mgr *wg.Manager) {
	path, err := config.EnsureRulesFile(name)
	if err != nil {
		log.Printf("wgtray: ui: rules editor (fallback) ensure rules file %s: %v", name, err)
		return
	}
	log.Printf("wgtray: ui: rules editor not available on this platform, edit manually: %s", path)
}
