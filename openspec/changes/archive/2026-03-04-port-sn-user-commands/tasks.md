## 1. Config and client foundation

- [x] 1.1 Add `api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file` to `validConfigKeys` and `configDefaults` in `config.go`
- [x] 1.2 Extend `loadConfig()` to set env prefix bindings for the five new keys (`TRADER_API_URL`, `TRADER_WEB_URL`, `TRADER_INGESTION_URL`, `TRADER_NATS_URL`, `TRADER_NATS_CREDS_FILE`)
- [x] 1.3 Add `--api-url`, `--ingestion-url`, `--web-url` persistent flags to the root command in `main.go` and bind them via `viper.BindPFlag`
- [x] 1.4 Implement `PlatformClient` struct and `newPlatformClient()` in `client.go` (or new `platform_client.go`): `api_url` + `ingestion_url` fields, Bearer API key auth on all requests, no `auth/resolve` call
- [x] 1.5 Add `Get`, `Post`, `Put`, `Patch`, `Delete`, `GetRaw` methods to `PlatformClient` mirroring sn's client; attach auth header to API requests; attach ingestion secret header to ingestion mutations (empty string for non-admin users)
- [x] 1.6 Update error message in `resolveAPIKey()` to reference `trader auth login` instead of `sn auth login`

## 2. Config command update

- [x] 2.1 Add the five new keys to the `config show` output in `cmd_config.go`
- [x] 2.2 Update `configSource()` to handle the new keys (env vars, trader config, default)
- [x] 2.3 Verify `config set` and `config get` accept all new keys; add test for unknown key rejection

## 3. Auth command

- [x] 3.1 Create `cmd_auth.go` with `auth` parent command and `login`, `logout`, `status` subcommands
- [x] 3.2 Implement `auth login`: find free port, start local HTTP callback server, build login URL from `web_url` (fallback `api_url`), open browser, wait up to 120s for callback, write `api_key` to trader config
- [x] 3.3 Implement `auth logout`: remove `api_key` from trader config, print `Logged out.`
- [x] 3.4 Implement `auth status`: resolve API key via existing `resolveAPIKey()`, print masked key or not-authenticated message

## 4. Strategies command

- [x] 4.1 Create `cmd_strategies.go` with `strategies` parent command
- [x] 4.2 Implement `strategies list`: call `GET /strategies` via `PlatformClient.apiURL`; render table with TYPE, NAME, DESCRIPTION, ACTIVE columns; `-` in ACTIVE for built-ins; `yes`/`no` for user strategies; support `--active` flag
- [x] 4.3 Implement `strategies get <id>`: call `GET /user-strategies/<id>`; render key-value table; print source below table when non-empty; support `--json`
- [x] 4.4 Implement `strategies validate --name --file`: read file, POST to ingestion URL `/user-strategies/validate`; print `✓` or `✗` with error; exit non-zero on failure
- [x] 4.5 Implement `strategies create --name --file`: read file, POST to `api_url /user-strategies`; optional `--description`, `--params` (JSON); print confirmation; support `--json`
- [x] 4.6 Implement `strategies update <id> --file`: read file, PUT to `/user-strategies/<id>`; optional `--description`, `--params`; print confirmation; support `--json`
- [x] 4.7 Implement `strategies activate <id>` and `strategies deactivate <id>`: POST to respective endpoints; print confirmation
- [x] 4.8 Implement `strategies delete <id>`: DELETE `/user-strategies/<id>`; print confirmation; handle 404
- [x] 4.9 Implement `strategies backtest <id>`: POST to ingestion URL `/user-strategies/<id>/backtest`; required `--exchange`, `--product`, `--granularity`; optional `--mode`, `--start`, `--end`, `--leverage`; print progress and render result table; support `--json`
- [x] 4.10 Share `printBacktestResult()` helper with the backtest command (extract to `output.go` or a shared file)

## 5. Trading config command

- [x] 5.1 Create `cmd_trading.go` with `trading` parent command
- [x] 5.2 Implement `trading list [account]`: call `GET /config/trading` with optional `?account_id=` and `?enabled=true`; render table; support `--enabled` flag and `--json`
- [x] 5.3 Implement `trading get <account> <exchange> <product>`: call `GET /config/trading/<exchange>/<product>?account_id=<account>`; render detail table; support `--json`
- [x] 5.4 Implement `trading set <account> <exchange> <product>`: fetch existing config, merge flags, PUT; implement `--params` parsing (`strategy:key=value` and `strategy:clear`); support all optional strategy/leverage/enable flags; support `--json`
- [x] 5.5 Implement `trading delete <account> <exchange> <product>`: DELETE with `?account_id=<account>`; print confirmation; handle 404
- [x] 5.6 Extract `fmtStrategyParams()` and `mergeStrategyParams()` helpers (copy from sn, adapt to package)

## 6. Price command

- [x] 6.1 Create `cmd_price.go` with `price` parent command
- [x] 6.2 Implement `price <product>`: call `GET /prices/<exchange>/<product>?granularity=<g>`; render single-row table with EXCHANGE, PRODUCT, PRICE, OPEN, HIGH, LOW, VOLUME, AGE; `--exchange` defaults to `coinbase`, `--granularity` defaults to `ONE_MINUTE`; support `--json`
- [x] 6.3 Implement `price --all`: fetch `GET /ingestion/products?enabled=true`, concurrently fetch prices (semaphore of 10), sort by exchange then product, render table with `no data` fallback rows; support `--json` (successful results only)
- [x] 6.4 Implement `fmtAge()` helper: human-readable duration since `last_update`; prefix `!` for durations over 1 hour

## 7. Backtest command

- [x] 7.1 Create `cmd_backtest.go` with `backtest` parent command
- [x] 7.2 Implement `backtest run`: POST to `/backtests`; poll `GET /jobs/<id>` until completed or failed; print dots while waiting; fetch and render result on completion; support `--no-wait`, `--params` (key=value), `--json`
- [x] 7.3 Implement `backtest list`: GET `/backtests` with optional filters; sort client-side by date or winrate; render table; support `--limit 0` (all results); support `--json`
- [x] 7.4 Implement `backtest get <id>`: GET `/backtests/<id>`; render full detail table including conditional futures metrics rows; support `--json`
- [x] 7.5 Implement `backtest job <job-id>`: GET `/jobs/<id>`; branch on status (completed → fetch+render result, failed → error, pending → print poll hint); support `--json`
- [x] 7.6 Implement shared `parseParams()` helper (key=value string slice → map[string]float64)
- [x] 7.7 Implement shared `printBacktestResult()` that conditionally renders futures metric rows

## 8. Signals command

- [x] 8.1 Create `cmd_signals.go` with `signals` command
- [x] 8.2 Copy embedded read-only NGS NATS credentials from `sn/cmd/sn/signals.go` into `cmd_signals.go`
- [x] 8.3 Implement `resolveNATSCreds()`: check `nats_creds_file` config (with `~` expansion), otherwise write embedded creds to a temp file and return the path
- [x] 8.4 Implement `buildSignalAllowlist()`: call `GET /config/trading` via `PlatformClient`, expand enabled configs into `(exchange, product, granularity, strategy)` tuples; exit non-zero on fetch failure
- [x] 8.5 Implement allowlist `allows()` with prefix-tolerant matching (strip `_` and `+` suffixes iteratively)
- [x] 8.6 Implement `buildSubject()`: construct NATS subject from filter flags using `*` and `>` wildcards
- [x] 8.7 Implement signal subscription loop: connect to NATS, subscribe, filter via allowlist, render human-readable lines or `--json` single-line objects; block until SIGTERM/SIGINT; print `Unsubscribing...` on exit

## 9. Documentation

- [x] 9.1 Update `README.md` Installation section: remove "Install `sn` CLI (needed for login)" prerequisite; add `trader auth login` as the primary auth path
- [x] 9.2 Update `README.md` Authentication section: document `trader auth login` flow; keep env var and sn fallback as alternatives
- [x] 9.3 Add `README.md` sections for all new commands: `auth`, `strategies`, `trading`, `price`, `backtest`, `signals` — with examples matching the spec scenarios
- [x] 9.4 Update `README.md` Config key table: add `api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file` rows
- [x] 9.5 Update `README.md` Global flags table: add `--api-url`, `--ingestion-url`, `--web-url`
- [x] 9.6 Update `skills/trader/SKILL.md`: add all new commands with full flag reference and usage examples for `auth`, `strategies`, `trading`, `price`, `backtest`, `signals`
- [x] 9.7 Update `skills/trader/SKILL.md` Authentication section: document `trader auth login` as the primary path; update config key table
- [x] 9.8 Add bot patterns to `skills/trader/SKILL.md` for new commands: check live price before sizing, stream signals, list enabled trading configs, run a backtest
