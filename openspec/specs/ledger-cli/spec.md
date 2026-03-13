## Requirements

### Requirement: API key resolution
The CLI SHALL resolve the API key using the following priority order:
1. `TRADER_API_KEY` environment variable
2. `api_key` in `~/.config/trader/config.yaml` (manual override)
3. `api_key` in `~/.config/sn/config.yaml` (written by `sn auth login` or `trader auth login` targeting sn config)

If no API key can be found from any source the CLI SHALL print a clear error
message directing the user to run `trader auth login` or set `TRADER_API_KEY`,
and exit non-zero.

#### Scenario: Key resolved from sn config
- **WHEN** `TRADER_API_KEY` is not set and `~/.config/sn/config.yaml` contains `api_key`
- **THEN** the CLI SHALL use that key for all requests

#### Scenario: TRADER_API_KEY overrides sn config
- **WHEN** `TRADER_API_KEY` is set in the environment
- **THEN** the CLI SHALL use that value regardless of any config file contents

#### Scenario: No API key found
- **WHEN** no API key can be resolved from any source
- **THEN** the CLI SHALL print an error referencing `trader auth login` and exit non-zero

---

### Requirement: Config management
The CLI SHALL support `trader config show`, `trader config set <key> <value>`, and `trader config get <key>` to manage `~/.config/trader/config.yaml`. Valid writable keys are `trader_url`, `tenant_id`, `api_key`, `api_url`, `web_url`, `ingestion_url`, `nats_url`, `nats_creds_file`. `config show` SHALL display all keys with their resolved values and sources. `api_key` SHALL be masked (first 8 chars + `...`). The source column SHALL indicate `[env]`, `[trader]`, `[sn]`, or `[default]`.

The `api_url` key is the primary endpoint used by all data commands. Its default value SHALL be `https://signalngn-api-potbdcvufa-ew.a.run.app`. The `trader_url` key is retained for `trader watch` only.

#### Scenario: Config show displays all keys with sources
- **WHEN** `trader config show` is run
- **THEN** the CLI SHALL print a table of all config keys, their resolved values, and sources
- **AND** `api_key` SHALL be masked (first 8 chars + `...`)
- **AND** the source column SHALL indicate whether the key came from `[sn]`, `[trader]`, `[env]`, or `[default]`

#### Scenario: Config show includes api_url with default
- **WHEN** `trader config show` is run on a fresh install
- **THEN** the table SHALL include `api_url` showing `https://signalngn-api-potbdcvufa-ew.a.run.app` with source `[default]`

#### Scenario: Config set writes new platform key
- **WHEN** `trader config set api_url https://my-api.example.com` is run
- **THEN** the value SHALL be written to `~/.config/trader/config.yaml`

#### Scenario: Config set writes ledger URL
- **WHEN** `trader config set trader_url https://my-traderd.example.com` is run
- **THEN** the value SHALL be written to `~/.config/trader/config.yaml`

#### Scenario: Config get unknown key
- **WHEN** `trader config get unknown_key` is run
- **THEN** the CLI SHALL print an error and exit non-zero

---

### Requirement: Global platform URL flags
The root command SHALL expose `--api-url`, `--ingestion-url`, and `--web-url` persistent flags in addition to the existing `--trader-url` flag, each overriding the corresponding config value for a single invocation.

#### Scenario: Override API URL via flag
- **WHEN** `trader --api-url http://localhost:9090 strategies list` is run
- **THEN** all platform API calls SHALL use `http://localhost:9090` as the base URL

#### Scenario: Existing trader-url flag unaffected
- **WHEN** `trader --trader-url http://localhost:8080 accounts list` is run
- **THEN** ledger calls SHALL use `http://localhost:8080` and platform calls SHALL use the configured `api_url`
