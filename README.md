# WGTray

> Minimal macOS WireGuard tray app with per-config split tunneling.

<p align="center">
  <img src="icon/icon.png" width="128" alt="WGTray icon">
</p>

WGTray is a small menu bar app for people who already use WireGuard on macOS and want better route-level control without a heavy VPN client.

It manages multiple configs, supports include/exclude routing rules per config, uses Touch ID after the initial setup, and detects tunnels that were started outside the app.

## Why WGTray

- Keep a work tunnel active while local subnets, devices, or domains stay outside the VPN.
- Route only specific company traffic through WireGuard instead of tunneling everything.
- Manage several WireGuard configs from the menu bar without losing visibility into externally started tunnels.

## Who It's For

- macOS users who already have WireGuard `.conf` files.
- Developers, infra engineers, and power users who need split tunneling per config.
- People who want a lightweight tray workflow instead of a full VPN dashboard.

If you just want a generic plug-and-play VPN client, WGTray is probably not the right tool.

## Key Features

- Multiple WireGuard configs active at the same time.
- Per-config include/exclude rules for domains, IPs, and CIDRs.
- Rules editor in the tray menu for faster updates.
- Touch ID authentication after the one-time admin setup.
- Automatic detection of tunnels started outside WGTray.
- Connected/disconnected menu bar status with logs and notifications.

## Common Use Cases

| Use case | Mode | Example |
|----------|------|---------|
| Work VPN with local bypass | `exclude` | Keep `192.168.0.0/16` and `printer.local` outside the tunnel |
| Company-only routing | `include` | Send `internal.mycompany.com` and `10.10.0.0/16` through the VPN |
| Multiple environments | mixed | Keep several WireGuard configs available from the menu bar |

## Quick Start

**Prerequisite:** `brew install wireguard-tools`

1. Download the latest app from [GitHub Releases](https://github.com/xenmayer/wgtray/releases).
2. Move `WGTray.app` to `/Applications`.
3. Open it once and enter your password so WGTray can install its one-time sudoers rule.
4. Add a WireGuard `.conf` file with **Add Config...** or copy it into `~/.config/wgtray/`.
5. Optional: attach routing rules to `~/.config/wgtray/<name>.rules.json` and connect from the tray.

## Routing Example

```json
{
  "mode": "exclude",
  "entries": [
    "192.168.1.0/24",
    "printer.local",
    "example.com"
  ]
}
```

| Mode | Behavior |
|------|----------|
| `exclude` | Route listed IPs/domains directly and keep everything else on the VPN |
| `include` | Route only the listed IPs/domains through the VPN |

## Build From Source

**Prerequisites:** Go 1.20+, Xcode Command Line Tools

```bash
git clone https://github.com/xenmayer/wgtray.git
cd wgtray
make build
make bundle
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Installation, sudoers setup, first run, verification |
| [Configuration](docs/configuration.md) | Config directory, routing rules, modes, rules editor |
| [Architecture](docs/architecture.md) | Internal package structure and data flow |

## Notes

- WGTray is macOS-first. Linux support is partial.
- Routing rules resolve domains at connect time, so reconnect to refresh changed IPs.
- Config files are standard WireGuard configs; WGTray does not replace your provider workflow.
