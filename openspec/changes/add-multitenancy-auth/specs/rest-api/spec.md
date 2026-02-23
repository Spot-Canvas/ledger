## MODIFIED Requirements

### Requirement: List accounts endpoint

The system SHALL expose a `GET /api/v1/accounts` endpoint that returns all trading accounts
**scoped to the authenticated tenant**. The endpoint SHALL require a valid Bearer API key;
unauthenticated requests SHALL return HTTP 401. Only accounts belonging to the resolved
`tenant_id` SHALL be returned.

#### Scenario: Multiple accounts exist for tenant
- **WHEN** `GET /api/v1/accounts` is called with a valid API key and accounts "live" and "paper" exist for that tenant
- **THEN** the system SHALL return HTTP 200 with a JSON array containing both accounts with their ID, name, type, and creation timestamp

#### Scenario: No accounts exist for tenant
- **WHEN** `GET /api/v1/accounts` is called and no accounts exist for the authenticated tenant
- **THEN** the system SHALL return HTTP 200 with an empty JSON array

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /api/v1/accounts` is called with no Authorization header
- **THEN** the system SHALL return HTTP 401 with `{"error": "unauthorized"}`

#### Scenario: Tenant isolation — accounts of other tenants not returned
- **WHEN** tenant A calls `GET /api/v1/accounts` and tenant B also has accounts
- **THEN** the system SHALL return only tenant A's accounts

---

### Requirement: Portfolio summary endpoint

The system SHALL expose a `GET /api/v1/accounts/{accountId}/portfolio` endpoint that returns
the portfolio summary for the specified account **within the authenticated tenant's namespace**.
The endpoint SHALL require a valid Bearer API key. The account lookup SHALL be scoped to the
resolved `tenant_id`. Each position in the summary SHALL include metadata fields when present:
`stop_loss`, `take_profit`, and `confidence`.

#### Scenario: Account with open positions
- **WHEN** `GET /api/v1/accounts/live/portfolio` is called with a valid API key and the tenant has open positions in the "live" account
- **THEN** the system SHALL return HTTP 200 with a JSON object containing open positions (symbol, quantity, average entry price, market type, and any metadata fields) and aggregate realized P&L

#### Scenario: Account not found within tenant
- **WHEN** `GET /api/v1/accounts/nonexistent/portfolio` is called
- **THEN** the system SHALL return HTTP 404 with an error message

#### Scenario: Cross-tenant account isolation
- **WHEN** tenant A calls `GET /api/v1/accounts/live/portfolio` and tenant B also has a "live" account
- **THEN** the system SHALL return only tenant A's portfolio data

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /api/v1/accounts/live/portfolio` is called with no Authorization header
- **THEN** the system SHALL return HTTP 401 with `{"error": "unauthorized"}`

---

### Requirement: List positions endpoint

The system SHALL expose a `GET /api/v1/accounts/{accountId}/positions` endpoint that returns
all positions for the specified account **scoped to the authenticated tenant**. An optional query
parameter `status` SHALL filter by open or closed positions (default: open). Each position in
the response SHALL include metadata fields when present: `exit_price`, `exit_reason`,
`stop_loss`, `take_profit`, and `confidence`. Metadata fields SHALL be omitted from the JSON
response when NULL.

#### Scenario: List open positions
- **WHEN** `GET /api/v1/accounts/live/positions?status=open` is called with a valid API key
- **THEN** the system SHALL return HTTP 200 with only open positions for the authenticated tenant's "live" account

#### Scenario: List all positions including closed
- **WHEN** `GET /api/v1/accounts/live/positions?status=all` is called with a valid API key
- **THEN** the system SHALL return HTTP 200 with both open and closed positions for the authenticated tenant

#### Scenario: Closed position with exit metadata in response
- **WHEN** a closed position has exit_price 55000 and exit_reason "take profit reached"
- **THEN** the position JSON object SHALL include `"exit_price": 55000` and `"exit_reason": "take profit reached"`

#### Scenario: Open position with stop loss and take profit in response
- **WHEN** an open position has stop_loss 48000 and take_profit 55000
- **THEN** the position JSON object SHALL include `"stop_loss": 48000` and `"take_profit": 55000`

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /api/v1/accounts/live/positions` is called with no Authorization header
- **THEN** the system SHALL return HTTP 401 with `{"error": "unauthorized"}`

---

### Requirement: List trades endpoint

The system SHALL expose a `GET /api/v1/accounts/{accountId}/trades` endpoint that returns
trades for the specified account **scoped to the authenticated tenant**. It SHALL support query
parameters: `symbol`, `side`, `market_type`, `start`, `end`, `cursor`, and `limit`. Each trade
in the response SHALL include metadata fields when present: `strategy`, `entry_reason`,
`exit_reason`, `confidence`, `stop_loss`, and `take_profit`. Metadata fields SHALL be omitted
from the JSON response when NULL.

#### Scenario: List trades with filters
- **WHEN** `GET /api/v1/accounts/live/trades?symbol=BTC-USD&limit=10` is called with a valid API key
- **THEN** the system SHALL return HTTP 200 with up to 10 BTC-USD trades for the authenticated tenant's "live" account, ordered by timestamp descending, with a next-page cursor if more results exist

#### Scenario: Paginate through trades
- **WHEN** `GET /api/v1/accounts/live/trades?cursor=abc123` is called with a valid cursor
- **THEN** the system SHALL return HTTP 200 with the next page of trades

#### Scenario: Invalid cursor
- **WHEN** `GET /api/v1/accounts/live/trades?cursor=invalid` is called
- **THEN** the system SHALL return HTTP 400 with an error message

#### Scenario: Trade with metadata fields in response
- **WHEN** a trade has strategy "macd-rsi-v2" and confidence 0.85
- **THEN** the trade JSON object SHALL include `"strategy": "macd-rsi-v2"` and `"confidence": 0.85`

#### Scenario: Trade without metadata fields in response
- **WHEN** a trade has no metadata fields (all NULL)
- **THEN** the trade JSON object SHALL omit the metadata fields

#### Scenario: Tenant isolation — trades of other tenants not returned
- **WHEN** tenant A calls `GET /api/v1/accounts/live/trades` and tenant B also has trades in a "live" account
- **THEN** the system SHALL return only tenant A's trades

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /api/v1/accounts/live/trades` is called with no Authorization header
- **THEN** the system SHALL return HTTP 401 with `{"error": "unauthorized"}`

---

### Requirement: List orders endpoint

The system SHALL expose a `GET /api/v1/accounts/{accountId}/orders` endpoint that returns
orders for the specified account **scoped to the authenticated tenant**. It SHALL support query
parameters: `status`, `symbol`, `cursor`, and `limit`.

#### Scenario: List open orders
- **WHEN** `GET /api/v1/accounts/live/orders?status=open` is called with a valid API key
- **THEN** the system SHALL return HTTP 200 with only open orders for the authenticated tenant's "live" account

#### Scenario: List all orders
- **WHEN** `GET /api/v1/accounts/live/orders` is called with no status filter
- **THEN** the system SHALL return HTTP 200 with all orders for the authenticated tenant ordered by created timestamp descending

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /api/v1/accounts/live/orders` is called with no Authorization header
- **THEN** the system SHALL return HTTP 401 with `{"error": "unauthorized"}`
