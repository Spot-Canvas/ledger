package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// defaultNATSCreds holds embedded Synadia NGS credentials scoped to
// subscribe-only access. Publish is denied on all subjects ("*"), so this
// user can only receive messages — safe to embed in a public binary.
//
// This is the "CLI" user re-issued via the Synadia web console with
// pub.deny=["*"] added. The NKey is identical; only the JWT changed.
//
// Override at runtime: set nats_creds_file in ~/.config/trader/config.yaml or
// TRADER_NATS_CREDS_FILE env var to use different credentials.
const defaultNATSCreds = `-----BEGIN NATS USER JWT-----
eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJMTklBM1VUNk5VT1U0WlhPUkQyUEFKWkxSRE41UlFLRU9YVFVRU0VYRFdKRE80TzdXQTRBIiwiaWF0IjoxNzcxNjA4MTEzLCJpc3MiOiJBQUMzQlhCNE9VVVU2R0paT1pDNkpRQlBMMkxVS1BLSDJCRDNKVDdNWjRJSUpDNEdKM1lYV1VOMiIsIm5hbWUiOiJDTEkiLCJzdWIiOiJVREZTM0dZUVVGV1FCQlBPR1JSNExZSk9WN0hRNVNMVElMR1NPMkhLQlhPNEk2S1BPNjdNSEFPUiIsIm5hdHMiOnsicHViIjp7ImRlbnkiOlsiKiJdfSwic3ViIjp7fSwic3VicyI6LTEsImRhdGEiOi0xLCJwYXlsb2FkIjotMSwiaXNzdWVyX2FjY291bnQiOiJBRDdDNEpYTDVKR01MQ1hYTjVJR1BZWFNYRVNPUE1LWVdYSTM1UkpXRjNVSk9UWENCUENVR1dUNiIsInR5cGUiOiJ1c2VyIiwidmVyc2lvbiI6Mn19.f74k_pg8ZpW5uvdcwYonbNn7cniZwoWNCUPvZxJt70NWA5Izkyk-9U2wGpUVecyyOKdXjNK1IVSv2YpkR9_-DQ
------END NATS USER JWT------

-----BEGIN USER NKEY SEED-----
SUAJZ64KHDJ62K4YRMOCSO57HI6D5XX5MVH5WYPPAOZ2BJSET43GKMPN5M
------END USER NKEY SEED------`

// SignalPayload is the NATS message payload for a trading signal.
type SignalPayload struct {
	Strategy      string             `json:"strategy"`
	Product       string             `json:"product"`
	Exchange      string             `json:"exchange"`
	AccountID     string             `json:"account_id"`
	Action        string             `json:"action"`
	Market        string             `json:"market"`
	Leverage      int                `json:"leverage"`
	Price         float64            `json:"price"`
	Confidence    float64            `json:"confidence"`
	Reason        string             `json:"reason"`
	StopLoss      float64            `json:"stop_loss"`
	TakeProfit    float64            `json:"take_profit"`
	RiskReasoning string             `json:"risk_reasoning"`
	PositionPct   float64            `json:"position_pct"`
	Indicators    map[string]float64 `json:"indicators"`
	Timestamp     int64              `json:"timestamp"` // Unix seconds
}

var (
	sigExchange    string
	sigProduct     string
	sigGranularity string
	sigStrategy    string
)

var signalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Stream live strategy signals from NATS (Ctrl-C to stop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		useJSON, _ := cmd.Flags().GetBool("json")

		subject := buildSubject(sigExchange, sigProduct, sigGranularity, sigStrategy)

		creds, err := resolveNATSCreds()
		if err != nil {
			return err
		}

		natsURL := viper.GetString("nats_url")
		if natsURL == "" {
			natsURL = "tls://connect.ngs.global"
		}

		// Fetch the user's trading configs and build an allowlist.
		allowlist, err := buildSignalAllowlist()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch trading configs for filtering: %v\n", err)
			fmt.Fprintf(os.Stderr, "No signals will be shown. Check your api_key and api_url config.\n")
			os.Exit(1)
		}

		nc, err := nats.Connect(natsURL, nats.UserCredentials(creds))
		if err != nil {
			return fmt.Errorf("connect to NATS: %w", err)
		}
		defer nc.Close()

		fmt.Fprintf(os.Stderr, "Connected to NATS. Subscribing to: %s\n", subject)
		if len(allowlist) > 0 {
			fmt.Fprintf(os.Stderr, "Filtering to %d strategy slot(s) from your trading config.\n", len(allowlist))
		}

		sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			exchange, product, granularity, strategy := parseSignalSubject(msg.Subject)

			if !allowlist.allows(exchange, product, granularity, strategy) {
				return
			}

			if useJSON {
				var payload SignalPayload
				if err := json.Unmarshal(msg.Data, &payload); err == nil {
					out := map[string]any{
						"exchange":       exchange,
						"product":        product,
						"granularity":    granularity,
						"strategy":       strategy,
						"account_id":     payload.AccountID,
						"action":         payload.Action,
						"market":         payload.Market,
						"leverage":       payload.Leverage,
						"price":          payload.Price,
						"confidence":     payload.Confidence,
						"reason":         payload.Reason,
						"stop_loss":      payload.StopLoss,
						"take_profit":    payload.TakeProfit,
						"risk_reasoning": payload.RiskReasoning,
						"position_pct":   payload.PositionPct,
						"indicators":     payload.Indicators,
						"timestamp":      payload.Timestamp,
					}
					b, _ := json.Marshal(out)
					fmt.Println(string(b))
				} else {
					fmt.Printf("{\"subject\":%q,\"data\":%q}\n", msg.Subject, string(msg.Data))
				}
				return
			}

			// Human-readable output
			var payload SignalPayload
			ts := time.Now().Format("15:04:05")
			if err := json.Unmarshal(msg.Data, &payload); err == nil {
				if payload.Timestamp != 0 {
					ts = time.Unix(payload.Timestamp, 0).UTC().Format("15:04:05")
				}
				fmt.Printf("%s  %-10s  %-15s  %-15s  %-25s  %-12s  %-5s  price=%.4f  conf=%.2f  sl=%.4f  tp=%.4f\n",
					ts, exchange, product, granularity, strategy,
					payload.AccountID, payload.Action, payload.Price, payload.Confidence, payload.StopLoss, payload.TakeProfit)
			} else {
				fmt.Printf("%s  %s  %s\n", ts, msg.Subject, string(msg.Data))
			}
		})
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}
		defer func() {
			_ = sub.Unsubscribe()
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		fmt.Fprintln(os.Stderr, "\nUnsubscribing...")
		return nil
	},
}

// signalKey uniquely identifies a (exchange, product, granularity, strategy) tuple.
type signalKey struct {
	exchange    string
	product     string
	granularity string
	strategy    string
}

// signalAllowlist is the set of signal keys the authenticated user is allowed to see.
type signalAllowlist map[signalKey]struct{}

// allows returns true when the given signal tuple matches the allowlist.
//
// The strategy engine appends direction suffixes to base strategy names, so
// an exact match is not enough:
//   - exact:             "ml_xgboost"       matches "ml_xgboost"
//   - underscore suffix: "ml_xgboost_short" matches "ml_xgboost"
//   - plus suffix:       "ml_xgboost+trend" matches "ml_xgboost"
func (a signalAllowlist) allows(exchange, product, granularity, strategy string) bool {
	if _, ok := a[signalKey{exchange, product, granularity, strategy}]; ok {
		return true
	}
	for i := len(strategy) - 1; i >= 0; i-- {
		if strategy[i] == '_' || strategy[i] == '+' {
			base := strategy[:i]
			if _, ok := a[signalKey{exchange, product, granularity, base}]; ok {
				return true
			}
		}
	}
	return false
}

// buildSignalAllowlist fetches the user's trading configs and expands them into
// a flat set of (exchange, product, granularity, strategy) keys.
func buildSignalAllowlist() (signalAllowlist, error) {
	c := newPlatformClient()
	var configs []TradingConfig
	if err := c.Get(c.apiURL("/config/trading"), &configs); err != nil {
		return nil, err
	}

	allowlist := make(signalAllowlist)
	for _, tc := range configs {
		if !tc.Enabled {
			continue
		}
		allStrategies := append(append(tc.StrategiesLong, tc.StrategiesShort...), tc.StrategiesSpot...)
		for _, strat := range allStrategies {
			allowlist[signalKey{
				exchange:    tc.Exchange,
				product:     tc.ProductID,
				granularity: tc.Granularity,
				strategy:    strat,
			}] = struct{}{}
		}
	}
	return allowlist, nil
}

// buildSubject builds a NATS subject from filter flags.
// Unset filters become * (intermediate) or > (trailing strategy).
func buildSubject(exchange, product, granularity, strategy string) string {
	parts := make([]string, 4)

	if exchange != "" {
		parts[0] = exchange
	} else {
		parts[0] = "*"
	}
	if product != "" {
		parts[1] = product
	} else {
		parts[1] = "*"
	}
	if granularity != "" {
		parts[2] = granularity
	} else {
		parts[2] = "*"
	}
	if strategy != "" {
		parts[3] = strategy
	} else {
		parts[3] = ">"
	}

	return "signals." + strings.Join(parts, ".")
}

// parseSignalSubject extracts exchange, product, granularity, strategy from
// signals.{e}.{p}.{g}.{s}
func parseSignalSubject(subj string) (exchange, product, granularity, strategy string) {
	parts := strings.SplitN(subj, ".", 5)
	if len(parts) < 5 {
		return "", "", "", ""
	}
	return parts[1], parts[2], parts[3], parts[4]
}

// resolveNATSCreds returns the path to a credentials file. If nats_creds_file
// is configured, it expands ~ and returns that path. Otherwise it writes the
// embedded default credentials to a temp file and returns its path.
func resolveNATSCreds() (string, error) {
	credsFile := viper.GetString("nats_creds_file")
	if credsFile != "" {
		if strings.HasPrefix(credsFile, "~/") {
			home, _ := os.UserHomeDir()
			credsFile = home + credsFile[1:]
		}
		return credsFile, nil
	}

	tmp, err := os.CreateTemp("", "trader-nats-creds-*.creds")
	if err != nil {
		return "", fmt.Errorf("create temp creds file: %w", err)
	}
	if _, err := tmp.WriteString(defaultNATSCreds); err != nil {
		return "", fmt.Errorf("write creds: %w", err)
	}
	tmp.Close()
	return tmp.Name(), nil
}

func init() {
	signalsCmd.Flags().StringVar(&sigExchange, "exchange", "", "Filter by exchange")
	signalsCmd.Flags().StringVar(&sigProduct, "product", "", "Filter by product")
	signalsCmd.Flags().StringVar(&sigGranularity, "granularity", "", "Filter by granularity")
	signalsCmd.Flags().StringVar(&sigStrategy, "strategy", "", "Filter by strategy")

	rootCmd.AddCommand(signalsCmd)
}
