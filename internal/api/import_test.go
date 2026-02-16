package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportTrades_EmptyArray(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	body := `{"trades": []}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "trades array is empty" {
		t.Errorf("expected 'trades array is empty', got %q", resp["error"])
	}
}

func TestImportTrades_InvalidJSON(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	body := `not json`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestImportTrades_ValidationError(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// Missing required fields
	body := `{"trades": [{"trade_id": "t-1"}]}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected validation error message")
	}
}

func TestImportTrades_InvalidMarketType(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	body := `{"trades": [{"trade_id":"t-1","account_id":"live","symbol":"BTC-USD","side":"buy","quantity":1,"price":50000,"fee":5,"fee_currency":"USD","market_type":"options","timestamp":"2025-01-15T10:00:00Z"}]}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected market_type validation error")
	}
}

func TestImportTrades_TooMany(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// Build array with 1001 trades
	trades := make([]map[string]interface{}, 1001)
	for i := range trades {
		trades[i] = map[string]interface{}{
			"trade_id":     "t-" + string(rune(i)),
			"account_id":   "live",
			"symbol":       "BTC-USD",
			"side":         "buy",
			"quantity":     1,
			"price":        50000,
			"fee":          5,
			"fee_currency": "USD",
			"market_type":  "spot",
			"timestamp":    "2025-01-15T10:00:00Z",
		}
	}
	data, _ := json.Marshal(map[string]interface{}{"trades": trades})

	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "too many trades: max 1000 per request" {
		t.Errorf("expected max trades error, got %q", resp["error"])
	}
}

func TestImportTrades_RouteRegistered(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// Verify POST /api/v1/import doesn't return 404 or 405
	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("POST /api/v1/import returned 404 — route not registered")
	}
	if w.Code == http.StatusMethodNotAllowed {
		t.Error("POST /api/v1/import returned 405 — POST not allowed")
	}
}

func TestImportTrades_GETNotAllowed(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	req := httptest.NewRequest("GET", "/api/v1/import", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/v1/import: expected 405, got %d", w.Code)
	}
}

func TestImportTrades_SecondValidationFails(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// First trade valid, second invalid — whole batch should fail validation
	body := `{"trades": [
		{"trade_id":"t-1","account_id":"live","symbol":"BTC-USD","side":"buy","quantity":1,"price":50000,"fee":5,"fee_currency":"USD","market_type":"spot","timestamp":"2025-01-15T10:00:00Z"},
		{"trade_id":"t-2","account_id":"live","symbol":"BTC-USD","side":"buy","quantity":1,"price":50000,"fee":5,"fee_currency":"USD","market_type":"invalid","timestamp":"2025-01-15T11:00:00Z"}
	]}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	// Error should mention trade[1]
	if resp["error"] == "" {
		t.Error("expected validation error for trade[1]")
	}
}
