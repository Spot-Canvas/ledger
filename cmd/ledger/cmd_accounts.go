package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountStats mirrors the JSON returned by GET /api/v1/accounts/{accountId}/stats.
type AccountStats struct {
	TotalTrades      int     `json:"total_trades"`
	ClosedTrades     int     `json:"closed_trades"`
	WinCount         int     `json:"win_count"`
	LossCount        int     `json:"loss_count"`
	WinRate          float64 `json:"win_rate"`
	TotalRealizedPnL float64 `json:"total_realized_pnl"`
	OpenPositions    int     `json:"open_positions"`
}

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage ledger accounts",
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts for the authenticated tenant",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		useJSON, _ := cmd.Flags().GetBool("json")

		if useJSON {
			_, raw, err := c.GetRaw(c.ledgerURL("/api/v1/accounts"))
			if err != nil {
				return err
			}
			fmt.Print(string(raw))
			return nil
		}

		var accounts []account
		if err := c.Get(c.ledgerURL("/api/v1/accounts"), &accounts); err != nil {
			return err
		}

		rows := make([][]string, len(accounts))
		for i, a := range accounts {
			rows[i] = []string{a.ID, a.Name, a.Type, fmtTime(a.CreatedAt)}
		}
		PrintTable([]string{"ID", "NAME", "TYPE", "CREATED"}, rows)
		return nil
	},
}

var accountsShowCmd = &cobra.Command{
	Use:   "show <account-id>",
	Short: "Show all-time aggregate statistics for an account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		c := newClient()
		useJSON, _ := cmd.Flags().GetBool("json")

		statusCode, raw, err := c.GetRaw(c.ledgerURL("/api/v1/accounts/" + accountID + "/stats"))
		if err != nil {
			return err
		}

		if statusCode == 404 {
			fmt.Fprintln(os.Stderr, "account not found")
			os.Exit(1)
		}
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("API error %d: %s", statusCode, string(raw))
		}

		if useJSON {
			fmt.Print(string(raw))
			return nil
		}

		var stats AccountStats
		if err := json.Unmarshal(raw, &stats); err != nil {
			return fmt.Errorf("decode stats: %w", err)
		}

		winRatePct := stats.WinRate * 100
		rows := [][]string{
			{"Account", accountID},
			{"Total Trades", fmt.Sprintf("%d", stats.TotalTrades)},
			{"Closed Trades", fmt.Sprintf("%d", stats.ClosedTrades)},
			{"Wins", fmt.Sprintf("%d", stats.WinCount)},
			{"Losses", fmt.Sprintf("%d", stats.LossCount)},
			{"Win Rate", fmt.Sprintf("%.2f%%", winRatePct)},
			{"Realized P&L", fmtFloat(stats.TotalRealizedPnL)},
			{"Open Positions", fmt.Sprintf("%d", stats.OpenPositions)},
		}
		PrintTable([]string{"FIELD", "VALUE"}, rows)
		return nil
	},
}

func init() {
	accountsListCmd.Flags().Bool("json", false, "Output raw JSON")
	accountsShowCmd.Flags().Bool("json", false, "Output raw JSON")
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsShowCmd)
	rootCmd.AddCommand(accountsCmd)
}
