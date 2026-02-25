## Why

Test trades accumulate in the ledger with no way to remove them, polluting
account history, portfolio state, and P&L figures. A targeted delete capability
is needed so test trades can be cleaned up without touching the database
directly.

## What Changes

- Add `DELETE /api/v1/trades/{tradeId}` endpoint to the REST API — **BREAKING**
  change to the read-only API contract, which currently forbids write endpoints.
- Add `ledger trades delete <id>` CLI subcommand that calls the new endpoint.
- Update the `record-trade` agent skill to document the delete command so agents
  can clean up their own test trades.

## Capabilities

### New Capabilities

- `trade-deletion`: Delete a single trade by ID via REST API and CLI.

### Modified Capabilities

- `rest-api`: The "Read-only API" requirement is changing — the API will now
  expose one write endpoint (`DELETE /api/v1/trades/{tradeId}`).
- `ledger-cli`: A new `trades delete <id>` subcommand is being added under the
  existing `trades` command group.

## Impact

- **REST API**: New `DELETE /api/v1/trades/{tradeId}` route; the read-only
  policy requirement in `rest-api/spec.md` must be updated to carve out this
  exception.
- **CLI** (`cmd/ledger`): New `trades delete` subcommand; auth and URL
  resolution reuse existing mechanisms.
- **Agent skill** (`record-trade` SKILL.md): Document the delete command so
  agents know how to remove test trades.
- **Database**: Trade deletion may affect derived state (portfolio positions,
  realized P&L). The implementation must decide whether to cascade-recalculate
  or reject deletion of trades that affect open positions.
