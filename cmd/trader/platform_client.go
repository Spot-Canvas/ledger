package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/viper"
)

// PlatformClient wraps the dual-URL platform HTTP API (api_url + ingestion_url).
// It is separate from the ledger Client and does not require tenant_id.
type PlatformClient struct {
	APIURL       string
	IngestionURL string
	APIKey       string
	http         *http.Client
}

// newPlatformClient resolves the API key and returns a ready PlatformClient.
// Exits the process with a helpful message if the API key is missing.
func newPlatformClient() *PlatformClient {
	apiKey, _, err := resolveAPIKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return &PlatformClient{
		APIURL:       viper.GetString("api_url"),
		IngestionURL: viper.GetString("ingestion_url"),
		APIKey:       apiKey,
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

// platformDo performs an HTTP request against the platform API surfaces.
// API server requests receive Bearer api_key auth.
// Ingestion server mutating requests receive no Bearer token (non-admin users
// have no ingestion secret; the empty-string token would be rejected).
func (c *PlatformClient) platformDo(method, rawURL string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, rawURL, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	isIngestion := c.IngestionURL != "" && len(rawURL) >= len(c.IngestionURL) &&
		rawURL[:len(c.IngestionURL)] == c.IngestionURL
	if !isIngestion && c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{StatusCode: resp.StatusCode, Body: string(respBytes)}
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode response: %w\nbody: %s", err, string(respBytes))
		}
	}
	return nil
}

// apiURL builds a URL against the API server.
func (c *PlatformClient) apiURL(path string, params ...url.Values) string {
	u := c.APIURL + path
	if len(params) > 0 && params[0] != nil {
		q := params[0].Encode()
		if q != "" {
			u += "?" + q
		}
	}
	return u
}

// ingestionURL builds a URL against the ingestion server.
func (c *PlatformClient) ingestionURL(path string, params ...url.Values) string {
	u := c.IngestionURL + path
	if len(params) > 0 && params[0] != nil {
		q := params[0].Encode()
		if q != "" {
			u += "?" + q
		}
	}
	return u
}

// Get performs a GET request and unmarshals the response.
func (c *PlatformClient) Get(rawURL string, out any) error {
	return c.platformDo(http.MethodGet, rawURL, nil, out)
}

// GetRaw performs a GET and returns the raw bytes.
func (c *PlatformClient) GetRaw(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	isIngestion := c.IngestionURL != "" && len(rawURL) >= len(c.IngestionURL) &&
		rawURL[:len(c.IngestionURL)] == c.IngestionURL
	if !isIngestion && c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	return b, nil
}

// Post performs a POST request and unmarshals the response.
func (c *PlatformClient) Post(rawURL string, body, out any) error {
	return c.platformDo(http.MethodPost, rawURL, body, out)
}

// Put performs a PUT request and unmarshals the response.
func (c *PlatformClient) Put(rawURL string, body, out any) error {
	return c.platformDo(http.MethodPut, rawURL, body, out)
}

// Patch performs a PATCH request and unmarshals the response.
func (c *PlatformClient) Patch(rawURL string, body, out any) error {
	return c.platformDo(http.MethodPatch, rawURL, body, out)
}

// Delete performs a DELETE request.
func (c *PlatformClient) Delete(rawURL string) error {
	return c.platformDo(http.MethodDelete, rawURL, nil, nil)
}
