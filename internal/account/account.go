package account

import (
	"log/slog"
	"time"

	"github.com/jwtly10/tradebook/internal/types"
)

const (
	LONG  Direction = "LONG"
	SHORT Direction = "SHORT"
)

type Direction string

type Position struct {
	ID         int
	OpenTime   time.Time
	Direction  Direction // "LONG" or "SHORT"
	EntryPrice float64
	Size       float64
	StopLoss   float64
	TakeProfit float64
}

type Trade struct {
	ID         int
	EntryTime  time.Time
	ExitTime   time.Time
	Direction  Direction
	EntryPrice float64
	ExitPrice  float64
	Size       float64
	StopLoss   float64
	TakeProfit float64
	PnL        float64
	PnLPercent float64
	ExitReason string
}

type Account struct {
	Balance        float64
	openPositions  []*Position
	nextPositionID int
}

func NewAccount(initialBalance float64) *Account {
	return &Account{
		Balance:        initialBalance,
		openPositions:  []*Position{},
		nextPositionID: 1,
	}
}

func (a *Account) OpenTrade(signal types.Signal, timestamp time.Time) *Position {
	slog.Info("Opening trade", "action", signal.Action, "price", signal.Price, "size", signal.Size, "tp", signal.TP, "sl", signal.SL, "timestamp", timestamp)
	// TODO: Check if we have enough balance/margin
	// TODO: Apply risk management rules
	// OR potentially delegate this to the caller only

	var dir Direction
	switch signal.Action {
	case types.BUY:
		dir = LONG
	case types.SELL:
		dir = SHORT
	}

	pos := &Position{
		ID:         a.nextPositionID,
		OpenTime:   timestamp,
		Direction:  dir,
		EntryPrice: signal.Price,
		Size:       signal.Size,
		StopLoss:   signal.SL,
		TakeProfit: signal.TP,
	}

	a.nextPositionID++
	a.openPositions = append(a.openPositions, pos)

	return pos
}

// CheckExits checks all open positions against the given bar for stop loss or take profit hits.
func (a *Account) CheckExits(bar types.Bar) []Trade {
	var closedTrades []Trade
	remainingPositions := []*Position{}

	for _, pos := range a.openPositions {
		closed := false
		var trade Trade

		if pos.Direction == LONG {
			// Check stop loss
			slog.Debug("Checking LONG position for exits", "position_id", pos.ID, "stop_loss", pos.StopLoss, "take_profit", pos.TakeProfit, "bar_low", bar.Low, "bar_high", bar.High, "timestamp", bar.Timestamp)
			if bar.Low <= pos.StopLoss {
				slog.Info("Stop loss hit", "position_id", pos.ID, "stop_loss", pos.StopLoss, "bar_low", bar.Low, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.StopLoss, bar.Timestamp, "STOP_LOSS")
				closed = true
			}
			// Check take profit
			if bar.High >= pos.TakeProfit {
				slog.Info("Take profit hit", "position_id", pos.ID, "take_profit", pos.TakeProfit, "bar_high", bar.High, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.TakeProfit, bar.Timestamp, "TAKE_PROFIT")
				closed = true
			}
		} else { // DIR_SHORT
			slog.Debug("Checking SHORT position for exits", "position_id", pos.ID, "stop_loss", pos.StopLoss, "take_profit", pos.TakeProfit, "bar_low", bar.Low, "bar_high", bar.High, "timestamp", bar.Timestamp)
			// Check stop loss
			if bar.High >= pos.StopLoss {
				slog.Info("Stop loss hit", "position_id", pos.ID, "stop_loss", pos.StopLoss, "bar_high", bar.High, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.StopLoss, bar.Timestamp, "STOP_LOSS")
				closed = true
			}
			// Check take profit
			if bar.Low <= pos.TakeProfit {
				slog.Info("Take profit hit", "position_id", pos.ID, "take_profit", pos.TakeProfit, "bar_low", bar.Low, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.TakeProfit, bar.Timestamp, "TAKE_PROFIT")
				closed = true
			}
		}

		if closed {
			closedTrades = append(closedTrades, trade)
			slog.Info("Closed trade", "id", trade.ID, "exit_price", trade.ExitPrice, "pnl", trade.PnL, "reason", trade.ExitReason)
		} else {
			remainingPositions = append(remainingPositions, pos)
		}
	}

	a.openPositions = remainingPositions
	return closedTrades
}

func (a *Account) closePosition(pos *Position, exitPrice float64, exitTime time.Time, reason string) Trade {
	var pnl float64

	if pos.Direction == "LONG" {
		pnl = (exitPrice - pos.EntryPrice) * pos.Size
	} else { // SHORT
		pnl = (pos.EntryPrice - exitPrice) * pos.Size
	}

	a.Balance += pnl

	slog.Debug("Position closed", "id", pos.ID, "direction", pos.Direction, "entry_price", pos.EntryPrice, "exit_price", exitPrice, "pnl", pnl, "new_balance", a.Balance)

	return Trade{
		ID:         pos.ID,
		EntryTime:  pos.OpenTime,
		ExitTime:   exitTime,
		Direction:  pos.Direction,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  exitPrice,
		Size:       pos.Size,
		StopLoss:   pos.StopLoss,
		TakeProfit: pos.TakeProfit,
		PnL:        pnl,
		PnLPercent: (pnl / pos.EntryPrice) * 100,
		ExitReason: reason,
	}
}

func (a *Account) CloseAll(lastBar types.Bar) []Trade {
	var trades []Trade

	for _, pos := range a.openPositions {
		trade := a.closePosition(pos, lastBar.Close, lastBar.Timestamp, "END_OF_BACKTEST")
		trades = append(trades, trade)
	}

	a.openPositions = []*Position{}
	return trades
}

func (a *Account) OpenPositions() []*Position {
	return a.openPositions
}

func (a *Account) PositionCount() int {
	return len(a.openPositions)
}
