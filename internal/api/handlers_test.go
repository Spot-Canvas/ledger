package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint_NilNATS(t *testing.T) {
	srv := &Server{
		repo: nil,
		nc:   nil,
	}

	router := srv.Router()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without a real database, we expect 503 or a panic-recovered 500
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 503 or 500, got %d", w.Code)
	}
}

func TestMethodNotAllowed_PUTAndDELETE(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// PUT and DELETE should be 405 on all endpoints
	methods := []string{"PUT", "DELETE", "PATCH"}
	paths := []string{
		"/api/v1/accounts",
		"/api/v1/accounts/live/portfolio",
		"/api/v1/accounts/live/positions",
		"/api/v1/accounts/live/trades",
		"/api/v1/accounts/live/orders",
		"/api/v1/import",
	}

	for _, method := range methods {
		for _, path := range paths {
			req := httptest.NewRequest(method, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: expected 405, got %d", method, path, w.Code)
			}
		}
	}
}

func TestRouterHasCorrectGETRoutes(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	paths := []string{
		"/health",
		"/api/v1/accounts",
		"/api/v1/accounts/test/portfolio",
		"/api/v1/accounts/test/positions",
		"/api/v1/accounts/test/trades",
		"/api/v1/accounts/test/orders",
	}

	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Errorf("GET %s: got 404, route not registered", path)
		}
		if w.Code == http.StatusMethodNotAllowed {
			t.Errorf("GET %s: got 405, GET should be allowed", path)
		}
	}
}

func TestImportRouteAcceptsPOST(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	body := `{"trades": []}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 400 (empty trades) not 404 or 405
	if w.Code == http.StatusNotFound {
		t.Error("POST /api/v1/import: got 404, route not registered")
	}
	if w.Code == http.StatusMethodNotAllowed {
		t.Error("POST /api/v1/import: got 405, POST should be allowed")
	}
}

func TestJSONContentType(t *testing.T) {
	srv := &Server{nc: nil}
	router := srv.Router()

	// Use the import endpoint which returns a JSON error body
	body := `{"trades": []}`
	req := httptest.NewRequest("POST", "/api/v1/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("response body is not valid JSON: %v", err)
	}
}
