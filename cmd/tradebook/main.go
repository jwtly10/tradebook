package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jwtly10/tradebook/internal/backtest"
	"github.com/jwtly10/tradebook/internal/oanda"
	"github.com/jwtly10/tradebook/internal/strategy"
)

func main() {
	accountId := os.Getenv("OANDA_ACCOUNT_ID")
	if accountId == "" {
		slog.Error("OANDA_ACCOUNT_ID not set")
		return
	}

	apiKey := os.Getenv("OANDA_API_KEY")
	if apiKey == "" {
		slog.Error("OANDA_API_KEY not set")
	}
	client := oanda.NewOandaService(accountId, apiKey, "")

	from := time.Date(2025, 10, 23, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 24, 0, 0, 0, 0, time.UTC)

	req := oanda.CandleRequest{
		Instrument:  oanda.NAS100,
		Granularity: oanda.M15,
		From:        from,
		To:          to,
	}

	bars, err := client.FetchBars(
		context.Background(),
		req,
	)
	if err != nil {
		slog.Error("Failed to initialise bar data", "error", err)
		return
	}

	slog.Info("Loaded bars", "count", len(bars))

	strategy := strategy.NewDJATRStrategy()

	engine := backtest.NewEngine(bars, 10000)
	results := engine.Run(strategy)

	stats := results.Calculate()
	stats.Print()

	fmt.Println()
	results.PrintTradesBetween(len(results.Trades)-5, len(results.Trades))
}
