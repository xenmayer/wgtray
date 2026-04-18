package ui

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
	"wgtray/icon"
	"wgtray/internal/auth"
	"wgtray/internal/config"
	"wgtray/internal/notify"
	"wgtray/internal/wg"
)

const maxSlots = 20

// slot represents one config entry in the menu.
type slot struct {
	mu       sync.Mutex
	mainItem *systray.MenuItem
	editItem *systray.MenuItem
	name     string // current config name; empty when slot is unused
}

var (
	mgr      *wg.Manager
	slots    [maxSlots]*slot
	mAdd     *systray.MenuItem
	mOpenDir *systray.MenuItem
	mLogs    *systray.MenuItem
	mQuit    *systray.MenuItem
)

// OnReady is called by systray once the tray is initialised.
func OnReady() {
	mgr = wg.NewManager()

	systray.SetIcon(icon.Disconnected())
	systray.SetTooltip("WG VPN\nNo configs")

	// Pre-allocate menu slots (hidden until a config occupies them).
	for i := 0; i < maxSlots; i++ {
		s := &slot{}
		s.mainItem = systray.AddMenuItem("", "")
		s.mainItem.Hide()
		s.editItem = s.mainItem.AddSubMenuItem("Edit Rules...", "Open rules editor for this config")
		s.editItem.Hide()
		slots[i] = s
		go watchSlot(s)
	}

	systray.AddSeparator()
	mAdd = systray.AddMenuItem("Add Config…", "Copy a WireGuard .conf file to the config directory")
	mOpenDir = systray.AddMenuItem("Open Config Dir", "Open ~/.config/wgtray in Finder")
	mLogs = systray.AddMenuItem("View Logs", "Open wgtray.log in TextEdit")
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Quit", "Quit WG Tray")

	go watchStaticItems()

	// Upgrade outdated sudoers rule (e.g. missing ARM64 Homebrew paths).
	if auth.IsSetupDone() && !auth.IsSetupCurrent() {
		log.Println("wgtray: auth: sudoers rule exists but missing ARM64 paths, will re-install")
		if err := auth.RunFirstTimeSetup(); err != nil {
			log.Printf("wgtray: auth: sudoers upgrade failed: %v", err)
		} else {
			log.Println("wgtray: auth: sudoers rule upgraded successfully")
		}
	}

	// Immediate refresh + periodic 3-second polling.
	doRefresh()
	go func() {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for range t.C {
			doRefresh()
		}
	}()
}

// OnExit is called by systray just before the app exits.
func OnExit() {
	if mgr != nil {
		mgr.DisconnectAll()
	}
	log.Println("wgtray: exited")
}

// watchSlot listens for clicks on a single config slot.
func watchSlot(s *slot) {
	for {
		select {
		case <-s.mainItem.ClickedCh:
			s.mu.Lock()
			name := s.name
			s.mu.Unlock()
			if name == "" {
				continue
			}
			toggleTunnel(name)
			doRefresh()

		case <-s.editItem.ClickedCh:
			s.mu.Lock()
			name := s.name
			s.mu.Unlock()
			if name == "" {
				continue
			}
			log.Printf("wgtray: ui: edit rules clicked for %q", name)
			go openRulesEditor(name, mgr)
		}
	}
}

// watchStaticItems listens for clicks on the static bottom menu items.
func watchStaticItems() {
	for {
		select {
		case <-mAdd.ClickedCh:
			addConfig()

		case <-mOpenDir.ClickedCh:
			if err := config.EnsureConfigDir(); err == nil {
				exec.Command("open", config.ConfigDir()).Start() //nolint:errcheck
			}

		case <-mLogs.ClickedCh:
			logPath := filepath.Join(config.ConfigDir(), "wgtray.log")
			exec.Command("open", "-e", logPath).Start() //nolint:errcheck

		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// toggleTunnel connects or disconnects the named tunnel.
func toggleTunnel(name string) {
	cfgs, err := config.LoadConfigs()
	if err != nil {
		log.Printf("wgtray: load configs: %v", err)
		notify.Error("Config error", err.Error())
		return
	}

	var targetCfg *config.Config
	for i, cfg := range cfgs {
		if cfg.Name == name {
			targetCfg = &cfgs[i]
			break
		}
	}
	if targetCfg == nil {
		log.Printf("wgtray: config %q not found", name)
		notify.Error("Config not found", name)
		return
	}

	if mgr.IsActive(name) || wg.InterfaceForConfig(targetCfg.FilePath) != "" {
		if err := mgr.Disconnect(name); err != nil {
			log.Printf("wgtray: disconnect %s: %v", name, err)
			notify.Error("Disconnect failed", fmt.Sprintf("%s: %v", name, err))
		} else {
			notify.Info("Disconnected", name)
		}
		return
	}

	if err := mgr.Connect(*targetCfg); err != nil {
		log.Printf("wgtray: connect %s: %v", name, err)
		notify.Error("Connect failed", fmt.Sprintf("%s: %v", name, err))
	} else {
		notify.Info("Connected", name)
	}
}

// addConfig shows a file picker and copies the chosen .conf into the config dir.
func addConfig() {
	script := `POSIX path of (choose file with prompt "Select WireGuard config (.conf):")`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return // user cancelled or osascript error
	}
	srcPath := strings.TrimSpace(string(out))
	if srcPath == "" {
		return
	}
	base := filepath.Base(srcPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if err := config.CopyConfigFile(srcPath, name); err != nil {
		log.Printf("wgtray: copy config %s: %v", srcPath, err)
		return
	}
	doRefresh()
}

// doRefresh updates icon, tooltip, and menu items to reflect current state.
func doRefresh() {
	cfgs, err := config.LoadConfigs()
	if err != nil {
		log.Printf("wgtray: refresh load configs: %v", err)
		cfgs = nil
	}

	anyConnected := false

	for i := 0; i < maxSlots; i++ {
		s := slots[i]

		if i < len(cfgs) {
			cfg := cfgs[i]
			connected := mgr.IsActive(cfg.Name) || wg.InterfaceForConfig(cfg.FilePath) != ""
			if connected {
				anyConnected = true
			}

			label := fmt.Sprintf("  %s (disconnected)", cfg.Name)
			if connected {
				label = fmt.Sprintf("✓ %s (connected)", cfg.Name)
			}

			// Update shared name under lock; UI calls outside the lock.
			s.mu.Lock()
			s.name = cfg.Name
			s.mu.Unlock()

			s.mainItem.SetTitle(label)
			s.mainItem.Show()
			s.editItem.Show()
		} else {
			s.mu.Lock()
			s.name = ""
			s.mu.Unlock()

			s.mainItem.Hide()
			s.editItem.Hide()
		}
	}

	// Build tooltip.
	tooltip := "WG VPN"
	for _, cfg := range cfgs {
		if mgr.IsActive(cfg.Name) || wg.InterfaceForConfig(cfg.FilePath) != "" {
			tooltip += fmt.Sprintf("\n● %s — connected", cfg.Name)
		} else {
			tooltip += fmt.Sprintf("\n○ %s — disconnected", cfg.Name)
		}
	}
	if len(cfgs) == 0 {
		tooltip += "\nNo configs — place .conf files in ~/.config/wgtray/"
	}

	if anyConnected {
		systray.SetIcon(icon.Connected())
	} else {
		systray.SetIcon(icon.Disconnected())
	}
	systray.SetTooltip(tooltip)
}
