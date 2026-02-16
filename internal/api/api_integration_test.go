//go:build integration

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"ledger/internal/api"
	"ledger/internal/ingest"
	"ledger/internal/store"
)

// Integration test requires:
// - PostgreSQL running on DATABASE_URL
// - NATS running on NATS_URLS
//
// Run with: go test -tags=integration ./internal/api/ -v

func TestAPIIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://spot:spot@localhost:5432/spot_canvas?sslmode=disable"
	}
	natsURL := os.Getenv("NATS_URLS")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Set up database
	repo, err := store.NewRepository(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to db: %v", err)
	}
	defer repo.Close()

	if err := store.RunMigrations(ctx, repo.Pool()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Connect to NATS
	nc, err := natsgo.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect to nats: %v", err)
	}
	defer nc.Close()

	// Start consumer
	consumer := ingest.NewConsumer(nc, repo)
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()
	go consumer.Start(consumerCtx)
	time.Sleep(time.Second)

	// Publish test trades
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("create jetstream: %v", err)
	}

	tradeID := fmt.Sprintf("api-test-%d", time.Now().UnixNano())
	event := ingest.TradeEvent{
		TradeID:     tradeID,
		AccountID:   "api-test-account",
		Symbol:      "ETH-USD",
		Side:        "buy",
		Quantity:    2.0,
		Price:       3000,
		Fee:         6,
		FeeCurrency: "USD",
		MarketType:  "spot",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	if _, err := js.Publish(ctx, "ledger.trades.api-test-account.spot", data); err != nil {
		t.Fatalf("publish trade: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Set up API server
	srv := api.NewServer(repo, nc)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// Test GET /health
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("health: expected 200, got %d", resp.StatusCode)
	}

	// Test GET /api/v1/accounts
	resp, err = http.Get(ts.URL + "/api/v1/accounts")
	if err != nil {
		t.Fatalf("accounts request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("accounts: expected 200, got %d", resp.StatusCode)
	}

	// Test GET /api/v1/accounts/{accountId}/trades
	resp, err = http.Get(ts.URL + "/api/v1/accounts/api-test-account/trades?symbol=ETH-USD")
	if err != nil {
		t.Fatalf("trades request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("trades: expected 200, got %d", resp.StatusCode)
	}

	var tradeResult store.TradeListResult
	json.NewDecoder(resp.Body).Decode(&tradeResult)
	resp.Body.Close()

	found := false
	for _, tr := range tradeResult.Trades {
		if tr.TradeID == tradeID {
			found = true
			break
		}
	}
	if !found {
		t.Error("ingested trade not found in API response")
	}

	// Test GET /api/v1/accounts/{accountId}/positions
	resp, err = http.Get(ts.URL + "/api/v1/accounts/api-test-account/positions?status=open")
	if err != nil {
		t.Fatalf("positions request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("positions: expected 200, got %d", resp.StatusCode)
	}

	// Test GET /api/v1/accounts/{accountId}/portfolio
	resp, err = http.Get(ts.URL + "/api/v1/accounts/api-test-account/portfolio")
	if err != nil {
		t.Fatalf("portfolio request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("portfolio: expected 200, got %d", resp.StatusCode)
	}

	// Test 404 for non-existent account portfolio
	resp, err = http.Get(ts.URL + "/api/v1/accounts/nonexistent-account-xyz/portfolio")
	if err != nil {
		t.Fatalf("portfolio 404 request: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("portfolio 404: expected 404, got %d", resp.StatusCode)
	}
}

func TestImportTradesIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://spot:spot@localhost:5432/spot_canvas?sslmode=disable"
	}

	// Set up database
	repo, err := store.NewRepository(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to db: %v", err)
	}
	defer repo.Close()

	if err := store.RunMigrations(ctx, repo.Pool()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	srv := api.NewServer(repo, nil)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// Import a batch of historic trades
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	accountID := "import-test-" + suffix

	importBody := fmt.Sprintf(`{"trades": [
		{"trade_id":"imp-buy1-%s","account_id":"%s","symbol":"BTC-USD","side":"buy","quantity":1.0,"price":40000,"fee":20,"fee_currency":"USD","market_type":"spot","timestamp":"2024-06-01T10:00:00Z"},
		{"trade_id":"imp-buy2-%s","account_id":"%s","symbol":"BTC-USD","side":"buy","quantity":0.5,"price":42000,"fee":10.50,"fee_currency":"USD","market_type":"spot","timestamp":"2024-06-15T10:00:00Z"},
		{"trade_id":"imp-sell1-%s","account_id":"%s","symbol":"BTC-USD","side":"sell","quantity":0.5,"price":45000,"fee":11.25,"fee_currency":"USD","market_type":"spot","timestamp":"2024-07-01T10:00:00Z"}
	]}`, suffix, accountID, suffix, accountID, suffix, accountID)

	resp, err := http.Post(ts.URL+"/api/v1/import", "application/json", bytes.NewBufferString(importBody))
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("import: expected 200, got %d", resp.StatusCode)
	}

	var importResp api.ImportResponse
	json.NewDecoder(resp.Body).Decode(&importResp)
	resp.Body.Close()

	if importResp.Total != 3 {
		t.Errorf("expected total 3, got %d", importResp.Total)
	}
	if importResp.Inserted != 3 {
		t.Errorf("expected inserted 3, got %d", importResp.Inserted)
	}
	if importResp.Duplicates != 0 {
		t.Errorf("expected 0 duplicates, got %d", importResp.Duplicates)
	}
	if importResp.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", importResp.Errors)
	}

	// Verify trades via GET
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/accounts/%s/trades", ts.URL, accountID))
	if err != nil {
		t.Fatalf("list trades: %v", err)
	}
	var tradeResult store.TradeListResult
	json.NewDecoder(resp.Body).Decode(&tradeResult)
	resp.Body.Close()

	if len(tradeResult.Trades) != 3 {
		t.Errorf("expected 3 trades, got %d", len(tradeResult.Trades))
	}

	// Verify position was built correctly: bought 1.5 BTC, sold 0.5 → 1.0 remaining
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/accounts/%s/positions?status=open", ts.URL, accountID))
	if err != nil {
		t.Fatalf("list positions: %v", err)
	}
	var positions []struct {
		Symbol   string  `json:"symbol"`
		Quantity float64 `json:"quantity"`
	}
	json.NewDecoder(resp.Body).Decode(&positions)
	resp.Body.Close()

	if len(positions) != 1 {
		t.Fatalf("expected 1 open position, got %d", len(positions))
	}
	if positions[0].Quantity != 1.0 {
		t.Errorf("expected position quantity 1.0, got %f", positions[0].Quantity)
	}

	// Re-import same trades — all should be duplicates
	resp, err = http.Post(ts.URL+"/api/v1/import", "application/json", bytes.NewBufferString(importBody))
	if err != nil {
		t.Fatalf("re-import request: %v", err)
	}

	var reimportResp api.ImportResponse
	json.NewDecoder(resp.Body).Decode(&reimportResp)
	resp.Body.Close()

	if reimportResp.Inserted != 0 {
		t.Errorf("re-import: expected 0 inserted, got %d", reimportResp.Inserted)
	}
	if reimportResp.Duplicates != 3 {
		t.Errorf("re-import: expected 3 duplicates, got %d", reimportResp.Duplicates)
	}
}
