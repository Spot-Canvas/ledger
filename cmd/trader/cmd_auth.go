package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the SignalNGN platform",
}

// authLoginCmd implements `trader auth login`.
// It opens the web browser to the platform login page, starts a local HTTP
// server to receive the callback, and writes the api_key to the trader config.
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in via browser and store the API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Pick a random available local port
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("find free port: %w", err)
		}
		port := ln.Addr().(*net.TCPAddr).Port
		ln.Close()

		// Result channel — receives (apiKey, email) or error
		type result struct {
			apiKey string
			email  string
			err    error
		}
		ch := make(chan result, 1)

		// Start local HTTP callback server
		mux := http.NewServeMux()
		srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			apiKey := q.Get("api_key")
			email := q.Get("email")
			if apiKey == "" {
				http.Error(w, "missing api_key", http.StatusBadRequest)
				ch <- result{err: fmt.Errorf("callback missing api_key")}
				return
			}
			fmt.Fprintf(w, `<html><body><h2>Authenticated as %s — you may close this tab.</h2></body></html>`, email)
			ch <- result{apiKey: apiKey, email: email}
			// Shut down server after responding
			go func() { _ = srv.Shutdown(context.Background()) }()
		})

		go func() { _ = srv.ListenAndServe() }()

		// Build login URL — use web_url if configured, otherwise fall back to api_url.
		baseURL := viper.GetString("web_url")
		if baseURL == "" {
			baseURL = viper.GetString("api_url")
		}
		loginURL := fmt.Sprintf("%s/oauth/start?cli_port=%d", baseURL, port)

		fmt.Printf("Opening browser for login: %s\n", loginURL)
		if err := openBrowser(loginURL); err != nil {
			fmt.Printf("Could not open browser automatically. Please visit: %s\n", loginURL)
		}

		// Wait up to 120 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		select {
		case res := <-ch:
			if res.err != nil {
				return fmt.Errorf("auth callback error: %w", res.err)
			}
			if err := writeConfigValue("api_key", res.apiKey); err != nil {
				return fmt.Errorf("write api_key to config: %w", err)
			}
			fmt.Printf("Authenticated as %s\n", res.email)
			return nil
		case <-ctx.Done():
			_ = srv.Shutdown(context.Background())
			return fmt.Errorf("login timed out after 120 seconds — please try again")
		}
	},
}

// authLogoutCmd implements `trader auth logout`.
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := deleteConfigValue("api_key"); err != nil {
			return fmt.Errorf("remove api_key from config: %w", err)
		}
		fmt.Println("Logged out.")
		return nil
	},
}

// authStatusCmd implements `trader auth status`.
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, _, err := resolveAPIKey()
		if err != nil || apiKey == "" {
			fmt.Println("Not authenticated. Run `trader auth login` to log in.")
			return nil
		}
		fmt.Printf("Authenticated (API key: %s)\n", maskValue(apiKey))
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

// openBrowser opens the given URL in the system default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", strings.ReplaceAll(url, "&", "^&")}
	default: // Linux / BSD
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
