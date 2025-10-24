//go:build integration
// +build integration

package oanda

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetchHistoricCandles_Integration(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug},
	)))

	accountID := os.Getenv("OANDA_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("OANDA_ACCOUNT_ID not set, skipping integration test")
	}

	apiKey := os.Getenv("OANDA_API_KEY")
	if apiKey == "" {
		t.Skip("OANDA_API_KEY not set, skipping integration test")
	}

	oanda := NewOandaService(accountID, apiKey, "")

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	req := CandleRequest{
		Instrument:  GBPUSD,
		Granularity: H1,
		From:        yesterday,
		To:          now,
	}

	resp, err := oanda.FetchHistoricCandles(context.Background(), req)
	if err != nil {
		t.Fatalf("integration test failed: %v", err)
	}

	// Note this *could* fail on a weekend etc, but this should be enough
	// to at least validate the integration works during normal
	// trading hours
	assert.True(t, len(resp.Candles) > 0, "expected at least one candle")

	t.Logf("Fetched %d candles for %s", len(resp.Candles), req.Instrument)
}
