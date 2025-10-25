package backtest

import "github.com/jwtly10/tradebook/internal/account"

type Results struct {
	InitialBalance float64
	FinalBalance   float64
	Trades         []account.Trade

	stats *Statistics
}
