## ADDED Requirements

### Requirement: signals stream
The CLI SHALL provide a `trader signals` subcommand that connects to NATS at `nats_url`, subscribes to the `signals.<exchange>.<product>.<granularity>.<strategy>` subject wildcard, and streams matching signals to stdout until interrupted with Ctrl-C. Optional filter flags: `--exchange`, `--product`, `--granularity`, `--strategy`. Unset filters SHALL use NATS wildcards (`*` for intermediate segments, `>` for the trailing strategy segment).

The CLI SHALL print the NATS subject and active filter count to stderr on startup. On Ctrl-C it SHALL print `Unsubscribing...` to stderr and exit zero.

#### Scenario: Connect and stream
- **WHEN** `trader signals` is run with valid credentials
- **THEN** the CLI SHALL connect to NATS, print `Connected to NATS. Subscribing to: signals.*.*.*.>` to stderr, and stream signals to stdout

#### Scenario: Filtered subscription
- **WHEN** `trader signals --exchange coinbase --product BTC-USD` is run
- **THEN** the CLI SHALL subscribe to `signals.coinbase.BTC-USD.*.*` and only deliver matching signals

#### Scenario: Graceful shutdown
- **WHEN** the user presses Ctrl-C
- **THEN** the CLI SHALL unsubscribe, print `Unsubscribing...` to stderr, and exit zero

#### Scenario: NATS connection failure
- **WHEN** the NATS server is unreachable
- **THEN** the CLI SHALL print a connection error and exit non-zero

---

### Requirement: signal output format
Each received signal SHALL be rendered as a single line to stdout. The default human-readable format SHALL include: timestamp (UTC HH:MM:SS), exchange, product, granularity, strategy, account ID, action, price, confidence, stop-loss, take-profit. With `--json` each signal SHALL be printed as a single-line JSON object including all payload fields.

#### Scenario: Human-readable signal line
- **WHEN** a signal is received in table mode
- **THEN** the CLI SHALL print a fixed-width line with timestamp, exchange, product, granularity, strategy, account, action, and key numeric fields

#### Scenario: JSON signal output
- **WHEN** `trader signals --json` is run and a signal is received
- **THEN** the CLI SHALL print a single-line JSON object with all signal fields

#### Scenario: Unparseable signal payload
- **WHEN** a NATS message cannot be parsed as a `SignalPayload`
- **THEN** the CLI SHALL print a fallback line with the raw subject and data rather than dropping the message silently

---

### Requirement: signal allowlist filtering
The CLI SHALL fetch `GET /config/trading` on startup to build an allowlist of `(exchange, product, granularity, strategy)` tuples from the authenticated user's enabled trading configs. Only signals whose tuple matches the allowlist SHALL be printed; all others SHALL be silently dropped.

The allowlist match SHALL be prefix-tolerant: a signal strategy `ml_xgboost_short` SHALL match an allowlist entry for `ml_xgboost` (the engine appends direction suffixes). Separators `_` and `+` SHALL be recognised for prefix splitting.

If the allowlist fetch fails the CLI SHALL print a warning to stderr and exit non-zero rather than showing all signals indiscriminately.

#### Scenario: Signal matches allowlist
- **WHEN** a signal for `(coinbase, BTC-USD, ONE_HOUR, ml_xgboost)` is received and the user has that config enabled
- **THEN** the signal SHALL be printed

#### Scenario: Signal blocked by allowlist
- **WHEN** a signal for a product/strategy not in the user's trading configs is received
- **THEN** the signal SHALL be silently dropped

#### Scenario: Suffix-tolerant matching
- **WHEN** a signal with strategy `ml_xgboost_short` is received and the allowlist contains `ml_xgboost`
- **THEN** the signal SHALL be printed (prefix match)

#### Scenario: Allowlist fetch failure
- **WHEN** `GET /config/trading` returns an error on startup
- **THEN** the CLI SHALL print a warning to stderr and exit non-zero

---

### Requirement: NATS credentials
The CLI SHALL use embedded read-only NGS credentials by default (subscribe-only, publish denied on all subjects). The CLI SHALL write these credentials to a temporary file and pass the path to the NATS client. If `nats_creds_file` is set in config or via `TRADER_NATS_CREDS_FILE`, that path SHALL be used instead and the embedded credentials SHALL be ignored. The temp file SHALL be created in the OS temp directory with a unique name.

#### Scenario: Embedded credentials used by default
- **WHEN** `trader signals` is run without `nats_creds_file` configured
- **THEN** the CLI SHALL write the embedded credentials to a temp file and connect using that file

#### Scenario: Custom credentials file used when configured
- **WHEN** `nats_creds_file` is set to `/home/user/.nats/custom.creds`
- **THEN** the CLI SHALL use that file directly without writing any temp file
