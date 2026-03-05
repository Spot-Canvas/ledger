package engine

import (
	"context"
	"testing"

	"github.com/Signal-ngn/trader/internal/config"
	"github.com/Signal-ngn/trader/internal/domain"
)

// ── mockBinanceFuturesClient ──────────────────────────────────────────────────

// mockBinanceFuturesClient records calls to the binanceFuturesClient interface.
// It is intentionally simple: each method records its arguments and returns
// the configured result or zero values.
type mockBinanceFuturesClient struct {
	// setLeverage recording
	leverageSet    int
	leverageSymbol string
	leverageErr    error

	// newOrder result
	orderResult *binanceOrderResult
	orderErr    error

	// getBalance result
	balance    float64
	balanceErr error

	// getPositionQty result
	positionQty    float64
	positionQtyErr error
}

func (m *mockBinanceFuturesClient) setLeverage(_ context.Context, symbol string, leverage int) error {
	m.leverageSymbol = symbol
	m.leverageSet = leverage
	return m.leverageErr
}

func (m *mockBinanceFuturesClient) newOrder(_ context.Context, _, _, _, _, _ string) (*binanceOrderResult, error) {
	if m.orderResult != nil {
		return m.orderResult, m.orderErr
	}
	return &binanceOrderResult{AvgPrice: 50000, Quantity: 0.002}, m.orderErr
}

func (m *mockBinanceFuturesClient) getBalance(_ context.Context) (float64, error) {
	return m.balance, m.balanceErr
}

func (m *mockBinanceFuturesClient) getPositionQty(_ context.Context, _ string) (float64, error) {
	return m.positionQty, m.positionQtyErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestExchange(mock *mockBinanceFuturesClient) *BinanceFuturesExchange {
	cfg := &config.Config{
		BinanceAPIKey:    "test-key",
		BinanceAPISecret: "test-secret",
	}
	return NewBinanceFuturesExchange(cfg).WithClient(mock)
}

// ── leverage tests ────────────────────────────────────────────────────────────

func TestOpenPosition_SpotSignalSetsLeverageOne(t *testing.T) {
	mock := &mockBinanceFuturesClient{}
	ex := newTestExchange(mock)

	_, err := ex.OpenPosition(context.Background(), OpenPositionRequest{
		Symbol:  "BTC-USD",
		Side:    domain.PositionSideLong,
		SizeUSD: 1000,
		Price:   50000,
		Leverage: 0, // spot-like signal carries no leverage
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.leverageSet != 1 {
		t.Errorf("setLeverage: want 1, got %d", mock.leverageSet)
	}
}

func TestOpenPosition_FuturesSignalSetsConfiguredLeverage(t *testing.T) {
	mock := &mockBinanceFuturesClient{}
	ex := newTestExchange(mock)

	_, err := ex.OpenPosition(context.Background(), OpenPositionRequest{
		Symbol:   "BTC-USD",
		Side:     domain.PositionSideLong,
		SizeUSD:  1000,
		Price:    50000,
		Leverage: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.leverageSet != 5 {
		t.Errorf("setLeverage: want 5, got %d", mock.leverageSet)
	}
}

// ── executedQty test ──────────────────────────────────────────────────────────

func TestOpenPosition_FillQuantityFromExecutedQty(t *testing.T) {
	mock := &mockBinanceFuturesClient{
		orderResult: &binanceOrderResult{
			AvgPrice: 50000,
			Quantity: 0.00123, // parsed from executedQty in the real HTTP client
		},
	}
	ex := newTestExchange(mock)

	result, err := ex.OpenPosition(context.Background(), OpenPositionRequest{
		Symbol:   "BTC-USD",
		Side:     domain.PositionSideLong,
		SizeUSD:  100,
		Price:    50000,
		Leverage: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const want = 0.00123
	if result.Quantity != want {
		t.Errorf("Quantity: want %v, got %v", want, result.Quantity)
	}
}
