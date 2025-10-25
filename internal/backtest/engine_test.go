package backtest

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jwtly10/tradebook/internal/account"
	"github.com/jwtly10/tradebook/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestEngine_RunAndCanOpenAndCloseTrades(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	bars := []types.Bar{
		{
			Timestamp: TimeFromString("2024-01-01T00:00:00Z"),
			Open:      100.0, High: 100, Low: 100.0, Close: 100.0, Volume: 1000,
		},
		{
			Timestamp: TimeFromString("2024-01-01T00:15:00Z"),
			Open:      100.0, High: 105.0, Low: 105.0, Close: 105.0, Volume: 1200,
		},
		{
			Timestamp: TimeFromString("2024-01-01T00:30:00Z"),
			Open:      99.5, High: 110.0, Low: 110.0, Close: 110.0, Volume: 1100,
		},
		{
			Timestamp: TimeFromString("2024-01-01T00:45:00Z"),
			Open:      101.0, High: 120.0, Low: 120.0, Close: 120.0, Volume: 1300,
		},
	}

	engine := NewEngine(bars, 10000.0)
	strategy := &TestStrategy{}

	results := engine.Run(strategy)

	// Trade 1:
	// Open BUY at 100 with quantity 1. Closed at 105 = +5 profit
	// Trade 2:
	// Open SELL at 105 with quantity 1. Closed at 120 due to strategy end - stop loss is not being factored in
	expectedFinalBalance := 10000.0 + 5.0 - 15.0

	assert.Equal(t, float64(100), results.Trades[0].EntryPrice, "First trade entry price should be 100.0")
	assert.Equal(t, float64(105), results.Trades[0].ExitPrice, "First trade exit price should be 105.0")
	assert.Equal(t, float64(5.0), results.Trades[0].PnL, "First trade PnL should be 5.0")

	assert.Equal(t, float64(105), results.Trades[1].EntryPrice, "Second trade entry price should be 105.0")
	assert.Equal(t, float64(120), results.Trades[1].ExitPrice, "Second trade exit price should be 120.0")
	assert.Equal(t, float64(-15.0), results.Trades[1].PnL, "Second trade PnL should be -15.0")

	assert.Equal(t, expectedFinalBalance, results.FinalBalance, "Final balance should match expected value")
	assert.Equal(t, 2, len(results.Trades), "There should be 2 closed trades")
}

type TestStrategy struct{}

func (s *TestStrategy) OnBar(bars []types.Bar, currentIndex int, account *account.Account) []types.Signal {
	// Opens BUY trade on the first bar, with a very small TP.
	// The next candle should always close it, and be profitable
	if currentIndex == 0 {
		bar := bars[currentIndex]
		signal := types.Signal{
			Type:   OPEN_TRADE,
			Action: types.BUY,
			Price:  bar.Close,
			TP:     bar.Close + 5,
			SL:     bar.Close - 50, // Too large to be hit
			Size:   1.0,
		}
		return []types.Signal{signal}
	}

	// Opens SELL trade on the second bar, with a very large stop loss
	// this will be closed at strategy end, and should be negative.
	if currentIndex == 1 {
		bar := bars[currentIndex]
		signal := types.Signal{
			Type:   OPEN_TRADE,
			Action: types.SELL,
			Price:  bar.Close,
			TP:     bar.Close - 50, // Too large to be hit
			SL:     bar.Close + 100,
			Size:   1.0,
		}
		return []types.Signal{signal}
	}

	return []types.Signal{}
}

func TimeFromString(timeStr string) (t time.Time) {
	t, _ = time.Parse(time.RFC3339, timeStr)
	return
}
