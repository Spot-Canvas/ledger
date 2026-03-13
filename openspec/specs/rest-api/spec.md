## Requirements

### Requirement: Health check endpoint
The system SHALL expose a `GET /health` endpoint that returns HTTP 200 unconditionally. The traderd binary has no database or NATS dependency; the health check does not probe any external service.

#### Scenario: Health check always returns OK
- **WHEN** `GET /health` is called while traderd is running
- **THEN** the system SHALL return HTTP 200 with body `{"status": "ok"}`

### Requirement: JSON response format
All API endpoints SHALL return JSON responses with `Content-Type: application/json`. Error responses SHALL use the format `{"error": "<message>"}`.

#### Scenario: Successful response
- **WHEN** any API endpoint returns data successfully
- **THEN** the response SHALL have `Content-Type: application/json` and a valid JSON body

#### Scenario: Error response
- **WHEN** any API endpoint encounters an error
- **THEN** the response SHALL have an appropriate HTTP status code and body `{"error": "<descriptive message>"}`
