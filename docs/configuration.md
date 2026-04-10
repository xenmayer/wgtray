[← Getting Started](getting-started.md) · [Back to README](../README.md) · [Architecture →](architecture.md)

# Configuration

Reference for WGTray's configuration directory, WireGuard config files, and routing rules.

## Config Directory

All WGTray files live in `~/.config/wgtray/`:

```
~/.config/wgtray/
├── my-vpn.conf           # WireGuard config (standard format)
├── my-vpn.rules.json     # Routing rules for my-vpn (optional)
├── work.conf
├── work.rules.json
├── tmp/                  # Temp rewritten configs (include mode) — auto-managed
└── wgtray.log            # Application log
```

WGTray polls this directory every **3 seconds**. Adding or removing a `.conf` file takes effect without a restart.

## WireGuard Config Files

Standard WireGuard `.conf` format — WGTray does not modify these files (except writing a temporary copy in `tmp/` for include-mode rules):

```ini
[Interface]
PrivateKey = <your-private-key>
Address = 10.0.0.2/24
DNS = 1.1.1.1

[Peer]
PublicKey = <server-public-key>
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0
```

Files must have the `.conf` extension. The filename (without extension) becomes the tunnel name shown in the menu.

## Routing Rules

Create `~/.config/wgtray/<name>.rules.json` to control which traffic goes through the VPN for a specific config. If no rules file exists, the tunnel uses the `AllowedIPs` from the `.conf` file unchanged.

```json
{
  "mode": "exclude",
  "entries": ["192.168.1.0/24", "example.com", "10.0.0.1"]
}
```

### Modes

| Mode | Behaviour |
|------|-----------|
| `exclude` | Route listed IPs/domains **directly** — everything else goes through the VPN |
| `include` | Route **only** listed IPs/domains through the VPN — everything else goes directly |

### Entries Format

Each entry in the `entries` array can be:

| Format | Example | Resolved to |
|--------|---------|-------------|
| CIDR | `192.168.1.0/24` | Used as-is |
| Bare IPv4 | `10.0.0.1` | `10.0.0.1/32` |
| Bare IPv6 | `2001:db8::1` | `2001:db8::1/128` |
| Domain name | `example.com` | DNS lookup → all resolved IPs as `/32` or `/128` |

> **Note:** Domain names are resolved at **connect time**. If the domain resolves to different IPs later, the routes are not updated until you reconnect.

### How Each Mode Works

**Exclude mode** (most common):

1. WGTray captures the default gateway before starting the tunnel.
2. `wg-quick up` brings the tunnel up (using `AllowedIPs` from the config, typically `0.0.0.0/0`).
3. For each resolved entry, WGTray adds a direct host route: `route add -net <cidr> <gateway>`.
4. On disconnect, those static routes are removed.

**Include mode:**

1. WGTray rewrites the `.conf` file's `AllowedIPs` in `[Peer]` sections to only include the resolved entries.
2. The rewritten config is saved to `~/.config/wgtray/tmp/<name>.conf`.
3. `wg-quick up` is called with the rewritten config.
4. On disconnect, the temp file is removed.

### Example: Work VPN, bypass local network

```json
{
  "mode": "exclude",
  "entries": [
    "192.168.0.0/16",
    "10.0.0.0/8",
    "printer.local"
  ]
}
```

### Example: Route only company traffic through VPN

```json
{
  "mode": "include",
  "entries": [
    "10.10.0.0/16",
    "internal.mycompany.com",
    "172.16.0.0/12"
  ]
}
```

## Editing Rules

Click **Edit Rules…** under any config name in the menu to open the `.rules.json` file in TextEdit. Save the file and reconnect to apply the new rules.

Or edit directly:

```bash
open -e ~/.config/wgtray/my-vpn.rules.json
```

## Permissions

| File | Mode | Notes |
|------|------|-------|
| `*.conf` | `0600` | Owner read/write only — contains private keys |
| `*.rules.json` | `0644` | World-readable |
| `tmp/` | `0700` | Owner only |

## See Also

- [Getting Started](getting-started.md) — installation and first run
- [Architecture](architecture.md) — how routing rules are applied internally
