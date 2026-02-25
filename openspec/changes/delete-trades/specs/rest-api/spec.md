## MODIFIED Requirements

### Requirement: Read-only API
All API endpoints SHALL be read-only (GET requests only) with the exception of
the trade import endpoint (`POST /api/v1/import`) and the trade deletion
endpoint (`DELETE /api/v1/trades/{tradeId}`). The system SHALL NOT expose any
other endpoints that create, update, or delete data. All other writes SHALL come
exclusively through NATS trade ingestion.

#### Scenario: Non-GET request to unsupported API endpoint
- **WHEN** a POST, PUT, or DELETE request is made to any `/api/v1/` endpoint other than `/api/v1/import` or `/api/v1/trades/{tradeId}`
- **THEN** the system SHALL return HTTP 405 Method Not Allowed

#### Scenario: DELETE request to trade deletion endpoint
- **WHEN** a DELETE request is made to `/api/v1/trades/{tradeId}` with a valid API key
- **THEN** the system SHALL process the deletion and return an appropriate response (200, 404, or 409)
