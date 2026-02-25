## 1. Store Layer

- [x] 1.1 Add `DeleteTrade(ctx context.Context, tenantID uuid.UUID, tradeID string) error` to `internal/store/trades.go`
- [x] 1.2 Implement open-position guard in `DeleteTrade`: query `ledger_positions` for any open position whose account/symbol matches the trade; return a sentinel error (e.g. `ErrTradeHasOpenPosition`) if found
- [x] 1.3 Implement the DELETE SQL: `DELETE FROM ledger_trades WHERE tenant_id = $1 AND trade_id = $2`; return `ErrTradeNotFound` if `RowsAffected() == 0`

## 2. REST API Handler

- [x] 2.1 Add `handleDeleteTrade` handler to `internal/api/handlers.go` that calls `repo.DeleteTrade`, maps `ErrTradeNotFound` → 404, `ErrTradeHasOpenPosition` → 409, and success → 200 `{"deleted": "<tradeId>"}`
- [x] 2.2 Register the route in `internal/api/router.go`: `r.Delete("/trades/{tradeId}", s.handleDeleteTrade)` inside the `/api/v1` auth-protected block
- [x] 2.3 Add `"DELETE"` to `AllowedMethods` in the CORS config in `internal/api/router.go`

## 3. CLI Subcommand

- [x] 3.1 Add `tradesDeleteCmd` (`ledger trades delete <trade-id>`) to `cmd/ledger/cmd_trades.go` with `--confirm` and `--json` flags
- [x] 3.2 Implement guard: if `--confirm` not set, print `use --confirm to delete a trade` and exit non-zero without making any HTTP request
- [x] 3.3 Implement HTTP call: `DELETE /api/v1/trades/{tradeId}` using the existing `client`; map 404 → `trade not found`, 409 → server error message, success → `deleted trade <id>`
- [x] 3.4 Wire `tradesDeleteCmd` into `tradesCmd` in the `init()` function

## 4. Agent Skill

- [x] 4.1 Add a "Deleting a trade" section to `.pi/skills/record-trade/SKILL.md` documenting `ledger trades delete <trade-id> --confirm` and when to use it (test trade cleanup only)

## 5. Tests

- [x] 5.1 Add unit tests for `DeleteTrade` in `internal/store/trades.go` covering: success, trade not found, open position guard
- [x] 5.2 Add handler tests in `internal/api/handlers_test.go` for `DELETE /api/v1/trades/{tradeId}` covering: 200, 404, 409, 401
