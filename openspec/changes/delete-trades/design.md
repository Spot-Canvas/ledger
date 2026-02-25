## Context

The ledger currently has no way to remove trades. All writes flow exclusively
through NATS ingestion or the `/api/v1/import` HTTP endpoint, and the REST API
is explicitly read-only. Test trades accumulate alongside real trades with no
cleanup path.

The store layer (`internal/store/trades.go`) has `InsertTrade` but no delete
counterpart. Positions and P&L are derived state computed at ingestion time via
`UpsertPosition` — deleting a trade without touching those derived tables would
leave the ledger in an inconsistent state.

The CLI already has a `trades` command group with `list` and `add` subcommands
(`cmd/ledger/cmd_trades.go`), so the new `delete` subcommand slots in naturally.

## Goals / Non-Goals

**Goals:**
- Add `DELETE /api/v1/trades/{tradeId}` to the REST API, scoped to the
  authenticated tenant.
- Add `ledger trades delete <id>` CLI subcommand.
- Update the `record-trade` agent skill to document the delete command.
- Keep the operation safe: reject deletes that would corrupt open positions.

**Non-Goals:**
- Bulk delete (delete by filter / account wipe) — out of scope for this change.
- Cascading recalculation of derived position/P&L state — too complex and risky
  for what is fundamentally a test-data cleanup operation.
- Soft delete / audit trail — not needed for test data removal.
- Admin-only access control — deletion uses the same tenant-scoped auth as all
  other write operations.

## Decisions

### Decision 1: Restrict deletion to trades that do not affect open positions

**Choice**: Return `HTTP 409 Conflict` if the trade contributes to any
currently open position (i.e., if `DELETE` would leave a position with
incorrect quantity or P&L).

**Rationale**: Recalculating derived state (positions, realized P&L) after an
arbitrary trade deletion is complex and error-prone. Test trades are typically
recorded against test accounts (e.g. `paper`) or are the most recent trades on
an account. Rejecting unsafe deletes is the safest initial behaviour — the
user can close the position first, then delete the trades.

**Alternative considered**: Cascade-recalculate positions after deletion. This
would require replaying all remaining trades for the account/symbol, essentially
reimplementing the ingestion pipeline in reverse. High complexity, high risk of
divergence. Rejected for this change.

**Alternative considered**: Allow deletion unconditionally (accept stale
derived state). Rejected — silent data corruption is worse than a clear error.

### Decision 2: Scope by tenant from auth context, look up by trade_id

**Choice**: The endpoint is `DELETE /api/v1/trades/{tradeId}`. The tenant ID
comes from the auth middleware context (same pattern as all other handlers).
The `WHERE tenant_id = $1 AND trade_id = $2` guard prevents cross-tenant
deletion.

**Rationale**: Consistent with existing handler patterns. No account ID in the
path is needed — trade IDs are globally unique UUIDs, and tenant scoping via
auth context provides the security boundary.

### Decision 3: Return 404 for not-found or wrong-tenant trades

**Choice**: If no row is deleted (trade doesn't exist or belongs to a different
tenant), return `HTTP 404 Not Found`.

**Rationale**: Standard REST semantics. Also avoids leaking whether a trade ID
belongs to another tenant.

### Decision 4: New `DeleteTrade` store method, no transaction needed

**Choice**: Add `DeleteTrade(ctx, tenantID, tradeID) error` to
`internal/store/trades.go`. The safety check (open position guard) is a
read-before-delete within the same DB call using a CTE or a pre-check query.

**Rationale**: No position rows are modified, so no transaction is needed
beyond what pgx already provides for a single statement. The pre-check can be a
separate `SELECT` followed by a conditional `DELETE` — acceptable for this use
case since concurrent ingestion of the same trade after a delete-pre-check is
extremely unlikely (test data scenario).

### Decision 5: CORS — add DELETE to allowed methods

**Choice**: Update the CORS config in `router.go` to include `"DELETE"` in
`AllowedMethods`.

**Rationale**: The current config allows only `GET`, `POST`, `OPTIONS`. Without
this, browser-based tooling would be blocked by preflight checks.

## Risks / Trade-offs

- **Open-position check is not transactionally airtight** → If a new trade for
  the same symbol is ingested via NATS between the pre-check and the DELETE,
  the open-position guard could give a false negative. Acceptable: this is a
  manual test-cleanup operation, not a high-concurrency path.

- **No undo** → Deletion is permanent. Mitigation: the CLI will print the
  trade ID and prompt confirmation (or require `--confirm` flag) before
  calling the API.

- **Spec change is breaking** → The `rest-api` spec currently states the API
  is read-only. Any client that relied on `HTTP 405` for DELETE will now get
  `HTTP 200` or `HTTP 404`. Mitigation: document clearly in the spec delta.

## Migration Plan

1. Deploy updated `ledgerd` binary with the new DELETE route.
2. No database migration required — no schema changes.
3. Rollback: redeploy previous binary; the DELETE route simply disappears.
   No data is affected.

## Open Questions

- Should the CLI require a `--confirm` flag, or print a confirmation prompt?
  → Decision: use a `--confirm` flag (non-interactive friendly for agents).
