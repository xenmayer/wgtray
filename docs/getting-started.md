[Back to README](../README.md) · [Configuration →](configuration.md)

# Getting Started

Everything you need to install WGTray, complete first-run setup, and connect your first tunnel.

## Prerequisites

| Requirement | Version | Install |
|-------------|---------|---------|
| macOS | 12 Monterey+ (recommended) | — |
| wireguard-tools | latest | `brew install wireguard-tools` |
| WireGuard configs | `.conf` files | from your VPN provider |

> **Why `wireguard-tools`?** WGTray uses `wg` and `wg-quick` under the hood to bring tunnels up and down.

## Install

1. Download the latest **WGTray.app** from [GitHub Releases](https://github.com/your-username/wgtray/releases).
2. Move `WGTray.app` to `/Applications`.
3. Open `WGTray.app` — you'll see the 🔒 icon appear in the menu bar.

## First-Run: Admin Setup

On the **first launch**, WGTray needs a one-time password to install a sudoers rule:

```
/etc/sudoers.d/wgtray
```

This rule allows `wg-quick`, `route`, and `sh` to run without a password for the `%admin` group. Once installed, all future tunnel operations use **Touch ID only** — no password dialogs.

> **No rule installed?** WGTray will fall back to an AppleScript administrator-privileges prompt for every connect/disconnect. Run the app once from the terminal (`open /Applications/WGTray.app`) if the first-run dialog doesn't appear.

## Add a WireGuard Config

**Option A — Menu bar:**
1. Click the WGTray icon in the menu bar.
2. Choose **Add Config…**
3. Select your `.conf` file in the file picker.

**Option B — Copy manually:**
```bash
cp ~/Downloads/my-vpn.conf ~/.config/wgtray/
```

Config files live in `~/.config/wgtray/`. WGTray polls this directory every 3 seconds and picks up new files automatically.

## Connecting

1. Click the WGTray icon.
2. Click your config name (shown as `  my-vpn (disconnected)`).
3. Authenticate with Touch ID (or password on the first run).
4. The icon changes to ✓ and a notification confirms the connection.

To disconnect, click the same item again.

## Verify the Connection

```bash
wg show
```

Expected output shows your interface (e.g. `utun4`) with a peer and a handshake timestamp.

```bash
curl https://ifconfig.me   # should return your VPN exit IP
```

## Logs

If something goes wrong, check the log file:

```bash
open -e ~/.config/wgtray/wgtray.log
```

Or use **View Logs** from the menu bar.

## See Also

- [Configuration](configuration.md) — routing rules and config file options
- [Architecture](architecture.md) — how WGTray works internally
