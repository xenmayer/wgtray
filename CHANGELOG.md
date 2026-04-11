# Changelog

## [0.2.1] — 2026-04-11

### Fixed

- Separate `wg-quick up/down` from route commands so a failing route no longer leaves the tunnel up but untracked, preventing "already exists" errors and double password prompts on reconnect
- Use `route add -inet6` / `route delete -inet6` for IPv6 CIDRs on macOS (previously used `-net` which is IPv4-only)
- Make route add/delete non-fatal — individual route failures are logged but no longer block connect or disconnect

## [0.2.0] — 2026-04-11

### Fixed

- Use absolute path for `wg` binary in `ActiveInterfaces()` so tunnel detection works when launched from Finder (minimal PATH)
- Handle "already exists" error in `Connect()` by automatically bringing down the stale tunnel before retrying, preventing double password prompts and crashes
