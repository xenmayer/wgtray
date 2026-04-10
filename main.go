package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/systray"
	"wgtray/internal/config"
	"wgtray/internal/ui"
)

func main() {
	// Set up file logging.
	if err := config.EnsureConfigDir(); err == nil {
		logPath := filepath.Join(config.ConfigDir(), "wgtray.log")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			log.SetOutput(f)
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		}
	}

	systray.Run(ui.OnReady, ui.OnExit)
}
