## ADDED Requirements

### Requirement: auth login
The CLI SHALL provide a `trader auth login` subcommand that opens the system browser to the platform OAuth start URL, starts a local HTTP callback server on a random free port, waits up to 120 seconds for the callback, writes the received `api_key` to `~/.config/trader/config.yaml`, and prints a confirmation showing the authenticated email address.

The login URL SHALL be `{web_url}/oauth/start?cli_port=<port>`. If `web_url` is not configured, the CLI SHALL fall back to `api_url`.

If the browser cannot be opened automatically, the CLI SHALL print the login URL and instruct the user to visit it manually.

#### Scenario: Successful login
- **WHEN** `trader auth login` is run and the browser callback delivers `api_key` and `email`
- **THEN** the CLI SHALL write `api_key` to `~/.config/trader/config.yaml`
- **AND** print `Authenticated as <email>`

#### Scenario: Browser cannot open
- **WHEN** `trader auth login` is run and the system browser cannot be launched
- **THEN** the CLI SHALL print `Could not open browser automatically. Please visit: <url>`
- **AND** continue waiting for the callback

#### Scenario: Login timeout
- **WHEN** `trader auth login` is run and no callback is received within 120 seconds
- **THEN** the CLI SHALL print `login timed out after 120 seconds — please try again` and exit non-zero

#### Scenario: Callback missing api_key
- **WHEN** the OAuth callback URL arrives without an `api_key` query parameter
- **THEN** the CLI SHALL return an HTTP 400 to the browser and exit non-zero with an error

---

### Requirement: auth logout
The CLI SHALL provide a `trader auth logout` subcommand that removes `api_key` from `~/.config/trader/config.yaml` and prints `Logged out.`

#### Scenario: Successful logout
- **WHEN** `trader auth logout` is run and `api_key` is present in the config file
- **THEN** `api_key` SHALL be removed from `~/.config/trader/config.yaml`
- **AND** the CLI SHALL print `Logged out.`

#### Scenario: Logout when not logged in
- **WHEN** `trader auth logout` is run and no `api_key` is present in any config
- **THEN** the CLI SHALL still print `Logged out.` and exit zero (idempotent)

---

### Requirement: auth status
The CLI SHALL provide a `trader auth status` subcommand that prints whether the user is currently authenticated. If an API key is resolved (from any source), it SHALL print `Authenticated (API key: <masked>)`. If not, it SHALL print `Not authenticated. Run \`trader auth login\` to log in.`

#### Scenario: Authenticated
- **WHEN** `trader auth status` is run and an API key is resolved
- **THEN** the CLI SHALL print `Authenticated (API key: <first-8-chars>...)`

#### Scenario: Not authenticated
- **WHEN** `trader auth status` is run and no API key is found in any source
- **THEN** the CLI SHALL print `Not authenticated. Run \`trader auth login\` to log in.` and exit zero
