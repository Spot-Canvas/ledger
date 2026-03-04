## ADDED Requirements

### Requirement: price single product
The CLI SHALL provide a `trader price <product>` subcommand that calls `GET /prices/<exchange>/<product>?granularity=<granularity>` and renders a table with columns `EXCHANGE`, `PRODUCT`, `PRICE`, `OPEN`, `HIGH`, `LOW`, `VOLUME`, `AGE`. The `--exchange` flag SHALL default to `coinbase`. The `--granularity` flag SHALL default to `ONE_MINUTE`. The AGE column SHALL show the time since `last_update` as a human-readable duration; durations over 1 hour SHALL be prefixed with `!` to signal staleness. With `--json` it SHALL print the raw JSON response.

#### Scenario: Single product price table
- **WHEN** `trader price BTC-USD` is run
- **THEN** the CLI SHALL call `GET /prices/coinbase/BTC-USD?granularity=ONE_MINUTE` and render a one-row table

#### Scenario: Custom exchange and granularity
- **WHEN** `trader price BTC-USD --exchange kraken --granularity ONE_HOUR` is run
- **THEN** the CLI SHALL call `GET /prices/kraken/BTC-USD?granularity=ONE_HOUR`

#### Scenario: AGE column freshness indicator
- **WHEN** `last_update` is more than 1 hour ago
- **THEN** the AGE column SHALL be prefixed with `!` (e.g. `!2h 15m`)

#### Scenario: Single product JSON output
- **WHEN** `trader price BTC-USD --json` is run
- **THEN** the CLI SHALL print the raw JSON candle object

#### Scenario: No product and no --all flag
- **WHEN** `trader price` is run without a product argument and without `--all`
- **THEN** the CLI SHALL print an error `product argument required (or use --all)` and exit non-zero

---

### Requirement: price all products
The CLI SHALL provide a `trader price --all` flag that fetches all enabled products from `GET /ingestion/products?enabled=true` and then concurrently fetches their prices (with a concurrency limit of 10). Results SHALL be sorted by exchange then product. Products with no price data SHALL render a row with `—` in all value columns and `no data` in AGE. With `--json` it SHALL print a JSON array of only the successful results.

#### Scenario: All products table
- **WHEN** `trader price --all` is run and 5 products are enabled
- **THEN** the CLI SHALL render a table with one row per product, sorted by exchange then product

#### Scenario: Product with no price data
- **WHEN** a product returns a non-2xx response when fetching its price
- **THEN** that product's row SHALL show `—` in all value columns and `no data` in AGE

#### Scenario: All products JSON output
- **WHEN** `trader price --all --json` is run
- **THEN** the CLI SHALL print a JSON array containing only products that returned successful price data

#### Scenario: No enabled products
- **WHEN** `trader price --all` is run and no products are enabled
- **THEN** the CLI SHALL print `No enabled products found.` and exit zero
