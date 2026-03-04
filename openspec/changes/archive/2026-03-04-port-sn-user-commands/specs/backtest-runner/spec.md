## ADDED Requirements

### Requirement: backtest run
The CLI SHALL provide a `trader backtest run` subcommand that submits a backtest job via `POST /backtests` and by default polls `GET /jobs/<id>` until the job completes, then fetches and renders the result. Required flags: `--exchange`, `--product`, `--strategy`, `--granularity`. Optional flags: `--mode` (default `spot`), `--start`, `--end`, `--leverage`, `--trend-filter`, `--no-wait`, `--params` (repeatable, format `key=value`).

With `--no-wait` the CLI SHALL print the job ID and a poll command then exit immediately. With `--json` it SHALL print the full result JSON. On job failure it SHALL print the error message and exit non-zero.

#### Scenario: Successful synchronous backtest
- **WHEN** `trader backtest run --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR` is run
- **THEN** the CLI SHALL submit the job, print `Job ID: <id>  Waiting.....done.`, fetch the result, and render the result table

#### Scenario: No-wait mode
- **WHEN** `trader backtest run --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR --no-wait` is run
- **THEN** the CLI SHALL print `Job ID: <id>  Status: pending` and `Poll:   trader backtest job <id>` then exit zero

#### Scenario: Backtest with params
- **WHEN** `--params confidence=0.80 --params exit_confidence=0.40` is provided
- **THEN** the request body SHALL include `params: {"confidence": 0.80, "exit_confidence": 0.40}`

#### Scenario: Invalid params format
- **WHEN** a `--params` value does not contain `=` or has an invalid float
- **THEN** the CLI SHALL print a parse error and exit non-zero without submitting

#### Scenario: Backtest job fails
- **WHEN** the job transitions to `failed` status while polling
- **THEN** the CLI SHALL print `backtest failed: <error>` and exit non-zero

#### Scenario: Missing required flags
- **WHEN** `trader backtest run` is run without `--exchange`, `--product`, `--strategy`, or `--granularity`
- **THEN** the CLI SHALL print an error listing the missing required flags and exit non-zero

#### Scenario: JSON output
- **WHEN** `trader backtest run [flags] --json` is run and the job completes
- **THEN** the CLI SHALL print the full backtest result JSON

---

### Requirement: backtest list
The CLI SHALL provide a `trader backtest list` subcommand that calls `GET /backtests` and renders a table with columns `ID`, `EXCHANGE`, `PRODUCT`, `STRATEGY`, `GRANULARITY`, `RETURN`, `WIN RATE`, `TRADES`, `DATE`, `PARAMS`. Optional flags: `--exchange`, `--product`, `--strategy`, `--limit` (default 20, 0 = all), `--sort` (`date` newest-first or `winrate` highest-first, default `date`). With `--json` it SHALL print the raw response.

#### Scenario: List backtests default
- **WHEN** `trader backtest list` is run
- **THEN** the CLI SHALL render the 20 most recent backtest results sorted by date descending

#### Scenario: Filter by strategy
- **WHEN** `trader backtest list --strategy ml_xgboost` is run
- **THEN** the CLI SHALL pass `?strategy=ml_xgboost` and show only matching results

#### Scenario: Sort by win rate
- **WHEN** `trader backtest list --sort winrate` is run
- **THEN** results SHALL be sorted by win rate descending

#### Scenario: Fetch all results
- **WHEN** `trader backtest list --limit 0` is run
- **THEN** the CLI SHALL request all results and render them all

#### Scenario: List JSON output
- **WHEN** `trader backtest list --json` is run
- **THEN** the CLI SHALL print the full JSON response including the `total` field

---

### Requirement: backtest get
The CLI SHALL provide a `trader backtest get <id>` subcommand that calls `GET /backtests/<id>` and renders a full result detail table. With `--json` it SHALL print the raw JSON. Futures-specific metrics (funding cost, near-liquidation count, min liquidation distance) SHALL be shown when `market_mode` is `futures-long` or `futures-short`.

#### Scenario: Get backtest detail
- **WHEN** `trader backtest get 123` is run
- **THEN** the CLI SHALL render a full detail table including all metrics

#### Scenario: Futures metrics shown for futures mode
- **WHEN** `trader backtest get 123` is run and the result has `market_mode: "futures-long"`
- **THEN** the table SHALL include rows for funding cost, near-liquidation count, and min liquidation distance

#### Scenario: Futures metrics omitted for spot mode
- **WHEN** `trader backtest get 123` is run and the result has `market_mode: "spot"`
- **THEN** futures-specific metric rows SHALL not appear in the table

#### Scenario: Get JSON output
- **WHEN** `trader backtest get 123 --json` is run
- **THEN** the CLI SHALL print the raw JSON object

---

### Requirement: backtest job
The CLI SHALL provide a `trader backtest job <job-id>` subcommand that calls `GET /jobs/<job-id>`. If the job is completed it SHALL fetch and render the result. If the job is failed it SHALL print the error and exit non-zero. If still pending or running it SHALL print the job status and a poll command. With `--json` it SHALL print the result or job JSON.

#### Scenario: Job completed
- **WHEN** `trader backtest job abc-123` is run and the job status is `completed`
- **THEN** the CLI SHALL fetch and render the backtest result

#### Scenario: Job still pending
- **WHEN** `trader backtest job abc-123` is run and the job status is `pending`
- **THEN** the CLI SHALL print `Job ID: abc-123  Status: pending` and `Poll:   trader backtest job abc-123`

#### Scenario: Job failed
- **WHEN** `trader backtest job abc-123` is run and the job status is `failed`
- **THEN** the CLI SHALL print `backtest failed: <error>` and exit non-zero
