# Changelog

## [0.2.0] — 2026-04-11

### Fixed

- Use absolute path for `wg` binary in `ActiveInterfaces()` so tunnel detection works when launched from Finder (minimal PATH)
- Handle "already exists" error in `Connect()` by automatically bringing down the stale tunnel before retrying, preventing double password prompts and crashes
