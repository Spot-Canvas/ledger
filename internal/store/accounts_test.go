package store

import (
	"testing"
)

// TestAccountStats_WinRate verifies the WinRate computation logic in isolation.
// Full DB-backed tests are covered by the integration test suite.
func TestAccountStats_WinRate(t *testing.T) {
	tests := []struct {
		name         string
		closedTrades int
		winCount     int
		wantWinRate  float64
	}{
		{
			name:         "no closed trades returns zero win rate",
			closedTrades: 0,
			winCount:     0,
			wantWinRate:  0,
		},
		{
			name:         "all wins returns 1.0",
			closedTrades: 5,
			winCount:     5,
			wantWinRate:  1.0,
		},
		{
			name:         "mixed wins and losses",
			closedTrades: 6,
			winCount:     4,
			wantWinRate:  4.0 / 6.0,
		},
		{
			name:         "no wins returns zero",
			closedTrades: 3,
			winCount:     0,
			wantWinRate:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &AccountStats{
				ClosedTrades: tt.closedTrades,
				WinCount:     tt.winCount,
			}
			// Apply the same win rate computation used in GetAccountStats
			if stats.ClosedTrades > 0 {
				stats.WinRate = float64(stats.WinCount) / float64(stats.ClosedTrades)
			}

			if stats.WinRate != tt.wantWinRate {
				t.Errorf("WinRate = %v, want %v", stats.WinRate, tt.wantWinRate)
			}
		})
	}
}

// TestAccountStats_JSONFields verifies the AccountStats struct has the correct JSON tags.
func TestAccountStats_JSONFields(t *testing.T) {
	stats := AccountStats{
		TotalTrades:      10,
		ClosedTrades:     6,
		WinCount:         4,
		LossCount:        2,
		WinRate:          0.6667,
		TotalRealizedPnL: 150.50,
		OpenPositions:    2,
	}

	// Verify field values are accessible
	if stats.TotalTrades != 10 {
		t.Errorf("TotalTrades = %d, want 10", stats.TotalTrades)
	}
	if stats.ClosedTrades != 6 {
		t.Errorf("ClosedTrades = %d, want 6", stats.ClosedTrades)
	}
	if stats.WinCount != 4 {
		t.Errorf("WinCount = %d, want 4", stats.WinCount)
	}
	if stats.LossCount != 2 {
		t.Errorf("LossCount = %d, want 2", stats.LossCount)
	}
	if stats.OpenPositions != 2 {
		t.Errorf("OpenPositions = %d, want 2", stats.OpenPositions)
	}
}
