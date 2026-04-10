# WGTray

**Minimalist WireGuard VPN tray app for macOS**

<p align="center">
  <img src="icon/icon.png" width="128" alt="WGTray icon">
</p>

WGTray lives in the menu bar and manages WireGuard tunnels with per-config routing rules,
Touch ID authentication, and automatic detection of externally started tunnels.

## Features

- Multiple WireGuard configs simultaneously
- Per-config routing rules: exclude or include specific IPs, domains, or CIDRs
- Touch ID authentication (macOS)
- Auto-detects externally started tunnels
- Menu bar status with connection indicator

## Install

**Prerequisites:** `brew install wireguard-tools`

1. Download the latest release from [GitHub Releases](../../releases).
2. Move **WGTray.app** to `/Applications`.
3. On first launch enter your password once — WGTray installs a `sudoers` rule so that all future operations use Touch ID only.

## Configuration

Place `.conf` files in `~/.config/wgtray/`, or use **Add Config…** from the tray menu.

Routing rules live in `~/.config/wgtray/<name>.rules.json`:

```json
{
  "mode": "exclude",
  "entries": ["192.168.1.0/24", "example.com", "10.0.0.1"]
}
```

| Mode | Behaviour |
|------|-----------|
| `exclude` | Route listed IPs/domains **directly** (bypass VPN) |
| `include` | Route **only** listed IPs/domains through VPN |

## Build from source

**Prerequisites:** Go 1.20+, Xcode Command Line Tools

```bash
git clone https://github.com/your-username/wgtray.git
cd wgtray
make build        # produces ./wgtray binary
make bundle       # produces WGTray.app
```

---

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Detailed installation, sudoers setup, first run |
| [Configuration](docs/configuration.md) | Config directory, routing rules, modes |
| [Architecture](docs/architecture.md) | Internal package structure and data flow |
