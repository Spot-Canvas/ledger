## ADDED Requirements

### Requirement: Delete trade subcommand
The CLI SHALL provide a `ledger trades delete <trade-id>` subcommand that calls
`DELETE /api/v1/trades/{tradeId}`. The command SHALL require a `--confirm` flag
to prevent accidental deletion. Without `--confirm` it SHALL print an error and
exit non-zero. On success it SHALL print the deleted trade ID. On HTTP 404 it
SHALL print `trade not found` and exit non-zero. On HTTP 409 it SHALL print the
server error message and exit non-zero.

#### Scenario: Successful deletion with confirm flag
- **WHEN** `ledger trades delete abc-123 --confirm` is run and the trade exists and is deletable
- **THEN** the CLI SHALL print `deleted trade abc-123` and exit zero

#### Scenario: Missing confirm flag
- **WHEN** `ledger trades delete abc-123` is run without `--confirm`
- **THEN** the CLI SHALL print an error such as `use --confirm to delete a trade` and exit non-zero
- **AND** no DELETE request SHALL be sent to the API

#### Scenario: Trade not found
- **WHEN** `ledger trades delete nonexistent-id --confirm` is run and the API returns HTTP 404
- **THEN** the CLI SHALL print `trade not found` and exit non-zero

#### Scenario: Trade contributes to open position
- **WHEN** `ledger trades delete abc-123 --confirm` is run and the API returns HTTP 409
- **THEN** the CLI SHALL print the error message from the API response and exit non-zero

#### Scenario: JSON output on success
- **WHEN** `ledger trades delete abc-123 --confirm --json` is run and the deletion succeeds
- **THEN** the CLI SHALL print the raw JSON response from the API
