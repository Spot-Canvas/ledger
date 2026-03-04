## Why

The `trader` CLI is the end-user tool for the Signal NGN platform, but it currently only covers the ledger (trades, positions, orders, accounts). The `sn` CLI holds the full set of platform commands — mixing admin-only and user-facing ones together. Non-admin users working with `trader` have no CLI access to strategies, backtests, trading configs, live prices, or real-time signals without also installing `sn`. Consolidating the user-facing `sn` commands into `trader` gives non-admin users a single, purpose-built CLI.

## What Changes

- Add `auth` command (login, logout, status) — browser OAuth flow that writes `api_key` to `~/.config/trader/config.yaml`
- Extend `config` with `api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file` keys so trader can talk to the full platform API surface
- Extend the HTTP client to support dual base URLs (`api_url` + `ingestion_url`), mirroring sn's client
- Add `strategies` command — unified strategy management: `list` shows all built-in and user strategies with a TYPE column; full CRUD subcommands (get, validate, create, update, activate, deactivate, delete, backtest) operate on user-defined strategies
- Add `trading` command (list, get, set, delete) — manage server-side trading configs
  - **Interface change vs sn**: account ID is a positional argument, not a `--account` flag (`trader trading set <account> <exchange> <product>`)
  - `trading reload` is not ported — it requires ingestion server admin credentials and stays in `sn`
- Add `price` command — show live price for one product or all enabled products (`--all`)
- Add `backtest` command (run, list, get, job)
- Add `signals` command — stream live trading signals from NATS; filtered to the user's own trading configs

## Capabilities

### New Capabilities

- `trader-auth`: Browser OAuth login/logout/status; writes and reads `api_key` from `~/.config/trader/config.yaml`; `auth login` opens browser to `web_url/oauth/start?cli_port=<port>` and listens for the callback
- `platform-client`: Dual-URL HTTP client (`api_url` + `ingestion_url`) with Bearer API key auth for API requests and Bearer ingestion secret (empty for non-admin users) for ingestion mutations; extended config keys
- `strategy-management`: Unified `strategies` command — `list` shows all built-in and user strategies with a TYPE column distinguishing them; CRUD subcommands (get, validate, create, update, activate, deactivate, delete, backtest) target user-defined strategies only
- `trading-config`: List, get, set, and delete server-side trading configs; account ID as first positional argument
- `price-feed`: Show live candle price for one product; `--all` flag to fetch all enabled products concurrently
- `backtest-runner`: Submit backtest jobs, poll for results, list and inspect historical backtest results
- `signal-stream`: Subscribe to NATS `signals.>` and stream real-time strategy signals; filtered to the authenticated user's enabled trading configs

### Modified Capabilities

- `ledger-cli`: The existing `config` command gains new valid keys (`api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file`) and a new `[sn]` source label for keys read from `~/.config/sn/config.yaml`

## Impact

- `cmd/trader/main.go` — add persistent flags for `--api-url`, `--ingestion-url`, `--web-url`
- `cmd/trader/config.go` — extend `validConfigKeys`, `configDefaults`, `loadConfig`
- `cmd/trader/client.go` — add dual-URL `PlatformClient` (separate from the existing ledger `Client`)
- New files: `cmd_auth.go`, `cmd_strategies.go`, `cmd_trading.go`, `cmd_price.go`, `cmd_backtest.go`, `cmd_signals.go`
- `README.md` — remove the "install `sn` for auth" prerequisite; add all new commands to the CLI reference; update config key table; update global flags table
- `skills/trader/SKILL.md` — add all new commands, flags, and usage patterns so AI agents can use the full CLI
- `go.mod` — already has `nats.go`; no new dependencies needed
- No changes to the ledger server or its REST API
