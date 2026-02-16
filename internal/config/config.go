package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the ledger service.
type Config struct {
	// HTTP server port
	HTTPPort string

	// Database settings
	DatabaseURL string

	// Cloud SQL settings (for GCP deployment)
	CloudSQLInstance string
	DBName           string
	DBUser           string
	DBPassword       string

	// NATS settings
	NATSURLs      string
	NATSCredsFile string
	NATSCreds     string

	// Logging
	LogLevel    string
	Environment string
}

// Load reads configuration from environment variables with .env support.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://spot:spot@localhost:5432/spot_canvas?sslmode=disable"),
		CloudSQLInstance: os.Getenv("CLOUDSQL_INSTANCE"),
		DBName:           getEnv("DB_NAME", "spot_canvas"),
		DBUser:           getEnv("DB_USER", "spot"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		NATSURLs:         getEnv("NATS_URLS", "nats://localhost:4222"),
		NATSCredsFile:    os.Getenv("NATS_CREDS_FILE"),
		NATSCreds:        os.Getenv("NATS_CREDS"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		Environment:      getEnv("ENVIRONMENT", "development"),
	}

	// Build Cloud SQL connection string if instance is specified
	if cfg.CloudSQLInstance != "" {
		cfg.DatabaseURL = cfg.buildCloudSQLURL()
	}

	return cfg, nil
}

// buildCloudSQLURL constructs a PostgreSQL connection string for Cloud SQL
// using Unix socket connection (required for Cloud Run).
func (c *Config) buildCloudSQLURL() string {
	socketDir := "/cloudsql"
	socketPath := fmt.Sprintf("%s/%s", socketDir, c.CloudSQLInstance)

	var sb strings.Builder
	sb.WriteString("postgres://")
	sb.WriteString(c.DBUser)
	if c.DBPassword != "" {
		sb.WriteString(":")
		sb.WriteString(c.DBPassword)
	}
	sb.WriteString("@/")
	sb.WriteString(c.DBName)
	sb.WriteString("?host=")
	sb.WriteString(socketPath)

	return sb.String()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
