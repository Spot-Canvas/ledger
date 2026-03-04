## ADDED Requirements

### Requirement: strategies list
The CLI SHALL provide a `trader strategies list` subcommand that calls `GET /strategies` and renders a single table with columns `TYPE`, `NAME`, `DESCRIPTION`, `ACTIVE`. Built-in strategies SHALL show `builtin` in TYPE and `-` in ACTIVE. User strategies SHALL show `user` in TYPE and `yes`/`no` in ACTIVE. With `--json` it SHALL print the raw JSON response.

#### Scenario: List shows both types
- **WHEN** `trader strategies list` is run and the platform has 3 built-in and 2 user strategies
- **THEN** the CLI SHALL print a table with 5 rows, TYPE column distinguishing `builtin` from `user`

#### Scenario: ACTIVE column for built-ins
- **WHEN** a built-in strategy appears in the list
- **THEN** its ACTIVE column SHALL show `-`

#### Scenario: ACTIVE column for user strategies
- **WHEN** a user strategy with `is_active: true` appears in the list
- **THEN** its ACTIVE column SHALL show `yes`

#### Scenario: List JSON output
- **WHEN** `trader strategies list --json` is run
- **THEN** the CLI SHALL print the raw JSON response from `GET /strategies`

#### Scenario: Filter active only
- **WHEN** `trader strategies list --active` is run
- **THEN** the CLI SHALL pass `?active=true` to the API and show only active user strategies alongside built-ins

---

### Requirement: strategies get
The CLI SHALL provide a `trader strategies get <id>` subcommand that calls `GET /user-strategies/<id>` and renders a detail view with fields ID, NAME, DESCRIPTION, ACTIVE, CREATED. If `--source` is passed (or the response includes source), the raw strategy source SHALL be printed below the table. With `--json` it SHALL print the full JSON response.

#### Scenario: Get user strategy table output
- **WHEN** `trader strategies get 42` is run
- **THEN** the CLI SHALL print a key-value table with ID, NAME, DESCRIPTION, ACTIVE, CREATED

#### Scenario: Get with source
- **WHEN** `trader strategies get 42` is run and the response includes a non-empty `source` field
- **THEN** the CLI SHALL print the source code below the table under a `--- Source ---` header

#### Scenario: Get JSON output
- **WHEN** `trader strategies get 42 --json` is run
- **THEN** the CLI SHALL print the full JSON object

#### Scenario: Strategy not found
- **WHEN** `trader strategies get 999` is run and the API returns 404
- **THEN** the CLI SHALL print an appropriate error and exit non-zero

---

### Requirement: strategies validate
The CLI SHALL provide a `trader strategies validate --name <name> --file <path>` subcommand that reads the source file and POSTs to `POST /user-strategies/validate` via the ingestion URL. If valid, it SHALL print `✓ Strategy is valid`. If invalid, it SHALL print `✗ Validation failed: <error>` and exit non-zero.

#### Scenario: Valid strategy
- **WHEN** `trader strategies validate --name my_strat --file strat.star` is run and the file is valid
- **THEN** the CLI SHALL print `✓ Strategy is valid` and exit zero

#### Scenario: Invalid strategy
- **WHEN** `trader strategies validate --name my_strat --file strat.star` is run and the server reports an error
- **THEN** the CLI SHALL print `✗ Validation failed: <error message>` to stderr and exit non-zero

#### Scenario: File not found
- **WHEN** `trader strategies validate --name x --file missing.star` is run
- **THEN** the CLI SHALL print a file-not-found error and exit non-zero

---

### Requirement: strategies create
The CLI SHALL provide a `trader strategies create --name <name> --file <path>` subcommand that reads the source file and POSTs to `POST /user-strategies`. It SHALL accept optional `--description` and `--params` (JSON object string) flags. On success it SHALL print `Created strategy "<name>" (ID: <id>)`. With `--json` it SHALL print the full JSON response.

#### Scenario: Create strategy
- **WHEN** `trader strategies create --name trend_follow --file strat.star` is run
- **THEN** the CLI SHALL POST to `/user-strategies` and print `Created strategy "trend_follow" (ID: <N>)`

#### Scenario: Create with description and params
- **WHEN** `trader strategies create --name x --file x.star --description "My strat" --params '{"THRESHOLD":2.0}'` is run
- **THEN** the request body SHALL include `description` and `parameters` fields

#### Scenario: Invalid params JSON
- **WHEN** `--params` is provided but is not valid JSON
- **THEN** the CLI SHALL print a parse error and exit non-zero without calling the API

---

### Requirement: strategies update
The CLI SHALL provide a `trader strategies update <id> --file <path>` subcommand that reads the source file and PUTs to `PUT /user-strategies/<id>`. It SHALL accept optional `--description` and `--params` flags. On success it SHALL print `Updated strategy ID <id>`. With `--json` it SHALL print the full JSON response.

#### Scenario: Update strategy source
- **WHEN** `trader strategies update 42 --file updated.star` is run
- **THEN** the CLI SHALL PUT the new source to `/user-strategies/42` and print a confirmation

#### Scenario: File not found
- **WHEN** `trader strategies update 42 --file missing.star` is run
- **THEN** the CLI SHALL print a file-not-found error and exit non-zero

---

### Requirement: strategies activate / deactivate
The CLI SHALL provide `trader strategies activate <id>` and `trader strategies deactivate <id>` subcommands that POST to `/user-strategies/<id>/activate` and `/user-strategies/<id>/deactivate` respectively. On success they SHALL print `Activated strategy ID <id>` or `Deactivated strategy ID <id>`.

#### Scenario: Activate
- **WHEN** `trader strategies activate 42` is run
- **THEN** the CLI SHALL POST to `/user-strategies/42/activate` and print `Activated strategy ID 42`

#### Scenario: Deactivate
- **WHEN** `trader strategies deactivate 42` is run
- **THEN** the CLI SHALL POST to `/user-strategies/42/deactivate` and print `Deactivated strategy ID 42`

---

### Requirement: strategies delete
The CLI SHALL provide a `trader strategies delete <id>` subcommand that calls `DELETE /user-strategies/<id>`. On success it SHALL print `Deleted strategy ID <id>`.

#### Scenario: Delete strategy
- **WHEN** `trader strategies delete 42` is run
- **THEN** the CLI SHALL DELETE `/user-strategies/42` and print `Deleted strategy ID 42`

#### Scenario: Strategy not found
- **WHEN** `trader strategies delete 999` is run and the API returns 404
- **THEN** the CLI SHALL print an appropriate error and exit non-zero

---

### Requirement: strategies backtest
The CLI SHALL provide a `trader strategies backtest <id>` subcommand that POSTs to `POST /user-strategies/<id>/backtest` via the ingestion URL. Required flags: `--exchange`, `--product`, `--granularity`. Optional flags: `--mode` (default `spot`), `--start`, `--end`, `--leverage`. On submission it SHALL print `Running backtest for user strategy <id>...`, wait for the result, then render a backtest result table. With `--json` it SHALL print the full JSON result.

#### Scenario: Run user strategy backtest
- **WHEN** `trader strategies backtest 42 --exchange coinbase --product BTC-USD --granularity ONE_HOUR` is run
- **THEN** the CLI SHALL POST to `/user-strategies/42/backtest` and render the result table on completion

#### Scenario: Backtest with futures mode and leverage
- **WHEN** `trader strategies backtest 42 --exchange coinbase --product BTC-USD --granularity ONE_HOUR --mode futures-long --leverage 5` is run
- **THEN** the request body SHALL include `market_mode: "futures-long"` and `leverage: 5`

#### Scenario: Backtest JSON output
- **WHEN** `trader strategies backtest 42 --exchange coinbase --product BTC-USD --granularity ONE_HOUR --json` is run
- **THEN** the CLI SHALL print the full backtest result JSON
