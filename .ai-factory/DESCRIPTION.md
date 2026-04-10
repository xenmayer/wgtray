# Project: WGTray

## Overview

WGTray is a minimalist macOS menu bar application that manages WireGuard VPN tunnels. It lives in the system tray, supports multiple simultaneous configs, per-config routing rules (exclude/include mode), Touch ID authentication, and auto-detects tunnels started externally by other tools.

## Core Features

- Multiple WireGuard configs active simultaneously
- Per-config routing rules: exclude specific IPs/domains/CIDRs from the VPN, or include only specified traffic through the VPN
- Touch ID authentication (macOS) — installs a sudoers rule on first run to avoid password prompts
- Auto-detects externally started WireGuard tunnels via peer public key matching
- Menu bar status icon (connected / disconnected)
- 3-second polling loop to reflect external state changes
- Add Config via file picker; open config dir in Finder; view logs in TextEdit

## Tech Stack

- **Language:** Go 1.20
- **UI/Tray:** fyne.io/systray v1.11.0
- **Platform:** macOS (primary); Linux partial support
- **Build:** Makefile (`make build`, `make bundle`, `make install`)
- **No database, no web framework, no ORM**

## Project Structure

```
main.go                     — Entry point; sets up file logging, runs systray
internal/
  config/store.go           — Config and Rules types; load/copy/ensure helpers
  wg/manager.go             — Tunnel lifecycle (connect/disconnect/disconnect-all)
  wg/rules.go               — Routing rule resolution (CIDR/domain → IP)
  wg/admin_darwin.go        — macOS admin execution (osascript / sudo)
  wg/admin_linux.go         — Linux admin execution stub
  wg/wgbin_darwin.go        — wg/wg-quick binary paths for macOS
  wg/wgbin_linux.go         — wg/wg-quick binary paths for Linux
  auth/touchid.go           — Touch ID authentication interface
  auth/touchid_darwin.{m,h} — Objective-C Touch ID implementation
  auth/setup.go             — sudoers rule installation
  notify/notify.go          — macOS user notifications
  ui/tray.go                — Systray menu, slot management, polling loop
icon/
  icon.go                   — Embedded icon bytes (connected/disconnected)
  *.png / *.icns             — Icon assets
Info.plist                  — macOS app metadata
Makefile                    — build / bundle / install / clean targets
```

## Architecture Notes

- Systray callbacks (`OnReady`, `OnExit`) drive the application lifecycle.
- A `Manager` struct (mutex-protected map) tracks tunnels started by this process.
- External tunnel detection uses peer public key comparison via `wg show <iface> peers`.
- Routing rules are applied at connect time: include mode rewrites AllowedIPs in a temp config; exclude mode adds static routes via `route add` after the tunnel is up.
- Admin operations are executed via `osascript` (AppleScript with administrator privileges) on macOS; a one-time sudoers install avoids repeated password prompts.
- Up to 20 config slots are pre-allocated in the menu to avoid dynamic systray item creation after startup.

## Architecture
See `.ai-factory/ARCHITECTURE.md` for detailed architecture guidelines.
Pattern: Layered Architecture

## Non-Functional Requirements

- Logging: File-based (`~/.config/wgtray/wgtray.log`) with `log.LstdFlags | log.Lshortfile`
- Error handling: Wrapped errors (`fmt.Errorf("context: %w", err)`); surfaced to user via system notifications
- Security: Configs stored in `~/.config/wgtray/` with 0600 permissions; sudoers rule scoped to `wg` and `wg-quick`
- No external network calls at runtime (all VPN operations are local CLI invocations)
