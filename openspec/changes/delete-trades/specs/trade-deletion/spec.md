## ADDED Requirements

### Requirement: Delete trade by ID
The system SHALL allow a tenant to delete a single trade by its trade ID via
`DELETE /api/v1/trades/{tradeId}`. The operation SHALL be scoped to the
authenticated tenant — a trade belonging to a different tenant SHALL NOT be
deleted. If the trade does not exist or belongs to a different tenant the system
SHALL return HTTP 404. If the trade contributes to an open position the system
SHALL return HTTP 409 and refuse the deletion. On success the system SHALL
return HTTP 200.

#### Scenario: Successful deletion
- **WHEN** `DELETE /api/v1/trades/abc-123` is called and the trade exists for the authenticated tenant and does not contribute to an open position
- **THEN** the system SHALL return HTTP 200 with body `{"deleted": "abc-123"}`
- **AND** the trade SHALL no longer appear in `GET /api/v1/accounts/{accountId}/trades`

#### Scenario: Trade not found
- **WHEN** `DELETE /api/v1/trades/nonexistent-id` is called and no trade with that ID exists for the authenticated tenant
- **THEN** the system SHALL return HTTP 404 with body `{"error": "trade not found"}`

#### Scenario: Trade belongs to different tenant
- **WHEN** `DELETE /api/v1/trades/{tradeId}` is called and the trade exists but belongs to a different tenant
- **THEN** the system SHALL return HTTP 404 (indistinguishable from not found)

#### Scenario: Trade contributes to an open position
- **WHEN** `DELETE /api/v1/trades/{tradeId}` is called and the trade is the only buy trade for an open spot position
- **THEN** the system SHALL return HTTP 409 with body `{"error": "trade contributes to an open position and cannot be deleted"}`
- **AND** the trade SHALL remain in the ledger unchanged

#### Scenario: Unauthenticated request
- **WHEN** `DELETE /api/v1/trades/{tradeId}` is called without a valid API key
- **THEN** the system SHALL return HTTP 401
