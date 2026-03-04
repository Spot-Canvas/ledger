## ADDED Requirements

### Requirement: Dual-URL HTTP client
The CLI SHALL provide a `PlatformClient` that holds two base URLs — `api_url` and `ingestion_url` — separate from the existing ledger `Client`. It SHALL attach `Authorization: Bearer <api_key>` to all API server requests. It SHALL not require `tenant_id` — the platform API is tenant-scoped server-side via the API key.

#### Scenario: API request carries Bearer token
- **WHEN** any platform command issues a GET to `api_url`
- **THEN** the request SHALL include `Authorization: Bearer <api_key>`

#### Scenario: Missing API key blocks platform commands
- **WHEN** a platform command is run and no API key can be resolved
- **THEN** the CLI SHALL print a clear error referencing `trader auth login` and exit non-zero

---

### Requirement: Extended config keys
The CLI SHALL recognise the following additional config keys in `~/.config/trader/config.yaml` and their corresponding environment variables:

| Key | Default | Env var |
|---|---|---|
| `api_url` | `https://signalngn-api-potbdcvufa-ew.a.run.app` | `TRADER_API_URL` |
| `web_url` | _(none)_ | `TRADER_WEB_URL` |
| `ingestion_url` | `https://signalngn-ingestion-potbdcvufa-ew.a.run.app` | `TRADER_INGESTION_URL` |
| `nats_url` | `tls://connect.ngs.global` | `TRADER_NATS_URL` |
| `nats_creds_file` | _(none)_ | `TRADER_NATS_CREDS_FILE` |

#### Scenario: Default API URL used when not configured
- **WHEN** `api_url` is not set in any config or env var
- **THEN** the CLI SHALL use `https://signalngn-api-potbdcvufa-ew.a.run.app` as the API base URL

#### Scenario: Env var overrides config file
- **WHEN** `TRADER_API_URL` is set in the environment
- **THEN** the CLI SHALL use that value regardless of the config file

---

### Requirement: Extended global flags
The root command SHALL expose `--api-url`, `--ingestion-url`, and `--web-url` persistent flags that override the corresponding config values for a single invocation.

#### Scenario: API URL overridden via flag
- **WHEN** `trader --api-url http://localhost:9090 strategies list` is run
- **THEN** all platform API calls SHALL use `http://localhost:9090` as the base URL

#### Scenario: Flag takes priority over env var and config
- **WHEN** both `TRADER_API_URL` and `--api-url` are set
- **THEN** the `--api-url` flag value SHALL be used
