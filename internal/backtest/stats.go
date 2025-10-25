package backtest

import (
	"fmt"
	"time"
)

type Statistics struct {
	// Basic
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64

	// P&L
	TotalPnL        float64
	TotalPnLPercent float64
	GrossProfit     float64
	GrossLoss       float64
	ProfitFactor    float64

	// Averages
	AvgWin        float64
	AvgLoss       float64
	ExpectedValue float64

	// Risk
	MaxDrawdown        float64
	MaxDrawdownPercent float64

	// Duration
	AvgTradeDuration time.Duration
}

func (r *Results) Calculate() *Statistics {
	// Return cached if already calculated
	if r.stats != nil {
		return r.stats
	}

	stats := &Statistics{
		TotalTrades: len(r.Trades),
	}

	if len(r.Trades) == 0 {
		r.stats = stats
		return stats
	}

	var totalWin, totalLoss float64
	var totalDuration time.Duration
	var peak float64 = r.InitialBalance
	var maxDD float64
	runningBalance := r.InitialBalance

	for _, trade := range r.Trades {
		// Win/Loss counting
		if trade.PnL > 0 {
			stats.WinningTrades++
			totalWin += trade.PnL
		} else if trade.PnL < 0 {
			stats.LosingTrades++
			totalLoss += trade.PnL // Already negative
		}

		// Drawdown calculation
		runningBalance += trade.PnL
		if runningBalance > peak {
			peak = runningBalance
		}
		dd := peak - runningBalance
		if dd > maxDD {
			maxDD = dd
		}

		// Duration
		duration := trade.ExitTime.Sub(trade.EntryTime)
		totalDuration += duration
	}

	// Win Rate
	stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100

	// P&L Stats
	stats.GrossProfit = totalWin
	stats.GrossLoss = totalLoss
	stats.TotalPnL = r.FinalBalance - r.InitialBalance
	stats.TotalPnLPercent = (stats.TotalPnL / r.InitialBalance) * 100

	// Profit Factor
	if totalLoss != 0 {
		stats.ProfitFactor = totalWin / -totalLoss
	}

	// Averages
	if stats.WinningTrades > 0 {
		stats.AvgWin = totalWin / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		stats.AvgLoss = totalLoss / float64(stats.LosingTrades)
	}
	stats.ExpectedValue = stats.TotalPnL / float64(stats.TotalTrades)

	// Drawdown
	stats.MaxDrawdown = maxDD
	if peak > 0 {
		stats.MaxDrawdownPercent = (maxDD / peak) * 100
	}

	// Duration
	stats.AvgTradeDuration = totalDuration / time.Duration(stats.TotalTrades)

	r.stats = stats
	return stats
}

func (s *Statistics) Print() {
	fmt.Println("\n=== Backtest Results ===")
	fmt.Printf("Total Trades:     %d\n", s.TotalTrades)
	fmt.Printf("Winning Trades:   %d (%.2f%%)\n", s.WinningTrades, s.WinRate)
	fmt.Printf("Losing Trades:    %d\n\n", s.LosingTrades)

	fmt.Printf("Total P&L:        £%.2f (%.2f%%)\n", s.TotalPnL, s.TotalPnLPercent)
	fmt.Printf("Gross Profit:     £%.2f\n", s.GrossProfit)
	fmt.Printf("Gross Loss:       £%.2f\n", s.GrossLoss)
	fmt.Printf("Profit Factor:    %.2f\n\n", s.ProfitFactor)

	fmt.Printf("Avg Win:          £%.2f\n", s.AvgWin)
	fmt.Printf("Avg Loss:         £%.2f\n", s.AvgLoss)
	fmt.Printf("Expected Value:   £%.2f per trade\n\n", s.ExpectedValue)

	fmt.Printf("Max Drawdown:     £%.2f (%.2f%%)\n", s.MaxDrawdown, s.MaxDrawdownPercent)
	fmt.Printf("Avg Duration:     %s\n", s.AvgTradeDuration.Round(time.Minute))
}

func (r *Results) PrintTrades() {
	fmt.Println("\n=== Trade List ===")
	for i, trade := range r.Trades {
		fmt.Printf("#%d | %s | Entry: %.5f | Exit: %.5f | P&L: £%.2f | %s | %s\n",
			i+1,
			trade.Direction,
			trade.EntryPrice,
			trade.ExitPrice,
			trade.PnL,
			trade.ExitReason,
			trade.EntryTime.Format("2006-01-02 15:04"),
		)
	}
}
