# WGTray

<p align="center">
  <img src="icon/icon.png" width="128" alt="WGTray icon" />
</p>

A minimal WireGuard VPN client that lives in the macOS menu bar.

## Features

- **Multiple configs** — run several WireGuard configs simultaneously
- **Routing rules** — per-config exclude (split-tunnel) or include (full-tunnel override) lists of IPs and domains
- **Touch ID** — authentication via Touch ID before every connect/disconnect (macOS only)
- **Auto-detect** — shows active interface status in the tray tooltip
- **Self-contained** — single binary, no daemon, no background service

## Requirements

```bash
brew install wireguard-tools
```

macOS 12+ recommended (Touch ID prompt requires LocalAuthentication).

## Install

Download the latest release from [Releases](../../releases) and move `WGTray.app` to `/Applications`.

On first launch the app will install a `sudoers` rule so that `wg-quick` can run without a password after Touch ID confirmation.

## Configuration

Drop your `.conf` WireGuard files into `~/.config/wgtray/configs/`.

Optionally place a `.rules.json` alongside each config to control routing:

```json
{
  "mode": "exclude",
  "entries": [
    "192.168.1.0/24",
    "example.com"
  ]
}
```

| Field | Values | Description |
|-------|--------|-------------|
| `mode` | `exclude` / `include` | `exclude` = route only these IPs/domains outside the VPN; `include` = route only these through the VPN |
| `entries` | list of IPs/CIDRs/domains | Domains are resolved at connect time |

## Build from source

```bash
git clone git@github.com:xenmayer/wgtray.git
cd wgtray
make bundle          # builds WGTray.app in ./dist/
make install         # copies to /Applications
```

### Cross-compile

```bash
make build-linux     # outputs dist/wgtray-linux-amd64
```

Linux build requires `libayatana-appindicator3-dev`.

## License

MIT
