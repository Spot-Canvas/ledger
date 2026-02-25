package store

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

// TestDeleteTrade_SentinelErrors verifies the sentinel errors are distinct and
// can be identified with errors.Is.
func TestDeleteTrade_SentinelErrors(t *testing.T) {
	if ErrTradeNotFound == nil {
		t.Fatal("ErrTradeNotFound must not be nil")
	}
	if ErrTradeHasOpenPosition == nil {
		t.Fatal("ErrTradeHasOpenPosition must not be nil")
	}
	if errors.Is(ErrTradeNotFound, ErrTradeHasOpenPosition) {
		t.Error("ErrTradeNotFound and ErrTradeHasOpenPosition must be distinct")
	}

	// Wrapping should still be detectable via errors.Is.
	wrapped := errors.Join(ErrTradeNotFound)
	if !errors.Is(wrapped, ErrTradeNotFound) {
		t.Error("errors.Is must match wrapped ErrTradeNotFound")
	}
	wrapped2 := errors.Join(ErrTradeHasOpenPosition)
	if !errors.Is(wrapped2, ErrTradeHasOpenPosition) {
		t.Error("errors.Is must match wrapped ErrTradeHasOpenPosition")
	}
}

// TestCursorRoundTrip verifies that encodeCursor/decodeCursor are inverses.
func TestCursorRoundTrip(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 123456789, time.UTC)
	tradeID := "abc-123-def"

	cursor := encodeCursor(ts, tradeID)
	if cursor == "" {
		t.Fatal("encodeCursor returned empty string")
	}

	gotTS, gotID, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decodeCursor error: %v", err)
	}
	if !gotTS.Equal(ts) {
		t.Errorf("timestamp roundtrip: got %v, want %v", gotTS, ts)
	}
	if gotID != tradeID {
		t.Errorf("tradeID roundtrip: got %q, want %q", gotID, tradeID)
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, _, err := decodeCursor("!!!not-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64 cursor, got nil")
	}
}

func TestDecodeCursor_InvalidFormat(t *testing.T) {
	// Valid base64 but missing the pipe separator.
	bad := base64.URLEncoding.EncodeToString([]byte("no-pipe-here"))
	_, _, err := decodeCursor(bad)
	if err == nil {
		t.Error("expected error for cursor with no pipe separator, got nil")
	}
}
