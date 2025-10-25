package backtest

import (
	"log/slog"

	"github.com/jwtly10/tradebook/internal/account"
	"github.com/jwtly10/tradebook/internal/types"
)

const (
	OPEN_TRADE = "OPEN_TRADE"
)

type Engine struct {
	Bars           []types.Bar
	initialBalance float64
}

func NewEngine(bars []types.Bar, initialBalance float64) *Engine {
	return &Engine{
		Bars:           bars,
		initialBalance: initialBalance,
	}
}

type Strategy interface {
	OnBar(bars []types.Bar, currentIndex int, account *account.Account) []types.Signal
}

func (e *Engine) Run(strategy Strategy) *Results {
	acc := account.NewAccount(e.initialBalance)
	results := &Results{
		InitialBalance: 10000,
		Trades:         []account.Trade{},
	}

	slog.Debug("Starting backtest", "initial_balance", e.initialBalance, "total_bars", len(e.Bars))

	for i, bar := range e.Bars {
		slog.Debug("Processing bar", "index", i, "timestamp", bar.Timestamp, "open", bar.Open, "high", bar.High, "low", bar.Low, "close", bar.Close)
		closedTrades := acc.CheckExits(bar)
		results.Trades = append(results.Trades, closedTrades...)

		signals := strategy.OnBar(e.Bars, i, acc)

		for _, signal := range signals {
			if signal.Type == OPEN_TRADE {
				acc.OpenTrade(signal, bar.Timestamp)
			}
		}
	}

	if len(e.Bars) > 0 {
		// Close anything at the end
		lastBar := e.Bars[len(e.Bars)-1]
		remainingTrades := acc.CloseAll(lastBar)
		results.Trades = append(results.Trades, remainingTrades...)
	}

	results.FinalBalance = acc.Balance

	return results
}
