package account

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jwtly10/tradebook/internal/types"
)

const (
	LONG  Direction = "LONG"
	SHORT Direction = "SHORT"
)

type Direction string

type Account struct {
	Balance        float64
	openPositions  []*Position
	nextPositionID int
}

type Position struct {
	ID         int
	OpenTime   time.Time
	Direction  Direction
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

func (t Trade) Print() {
	fmt.Printf("#%d | %s | Entry: %.5f @ %s | Exit: %.5f @ %s | P&L: £%.2f | %s\n",
		t.ID,
		t.Direction,
		t.EntryPrice,
		t.EntryTime.Format("2006-01-02 15:04"),
		t.ExitPrice,
		t.ExitTime.Format("2006-01-02 15:04"),
		t.PnL,
		t.ExitReason,
	)
}

func NewAccount(initialBalance float64) *Account {
	return &Account{
		Balance:        initialBalance,
		openPositions:  []*Position{},
		nextPositionID: 1,
	}
}

func (a *Account) OpenTrade(signal types.Signal, timestamp time.Time) *Position {
	slog.Info("Opening trade", "action", signal.Action, "id", a.nextPositionID, "price", signal.Price, "size", signal.Size, "tp", signal.TP, "sl", signal.SL, "timestamp", timestamp)
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
			if bar.Low <= pos.StopLoss {
				slog.Debug("Stop loss hit", "position_id", pos.ID, "stop_loss", pos.StopLoss, "bar_low", bar.Low, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.StopLoss, bar.Timestamp, "STOP_LOSS")
				closed = true
			}
			// Check take profit
			if bar.High >= pos.TakeProfit {
				slog.Debug("Take profit hit", "position_id", pos.ID, "take_profit", pos.TakeProfit, "bar_high", bar.High, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.TakeProfit, bar.Timestamp, "TAKE_PROFIT")
				closed = true
			}
		} else { // DIR_SHORT
			// Check stop loss
			if bar.High >= pos.StopLoss {
				slog.Debug("Stop loss hit", "position_id", pos.ID, "stop_loss", pos.StopLoss, "bar_high", bar.High, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.StopLoss, bar.Timestamp, "STOP_LOSS")
				closed = true
			}
			// Check take profit
			if bar.Low <= pos.TakeProfit {
				slog.Debug("Take profit hit", "position_id", pos.ID, "take_profit", pos.TakeProfit, "bar_low", bar.Low, "timestamp", bar.Timestamp)
				trade = a.closePosition(pos, pos.TakeProfit, bar.Timestamp, "TAKE_PROFIT")
				closed = true
			}
		}

		if closed {
			closedTrades = append(closedTrades, trade)
		} else {
			remainingPositions = append(remainingPositions, pos)
		}
	}

	a.openPositions = remainingPositions
	return closedTrades
}

func (a *Account) closePosition(pos *Position, exitPrice float64, exitTime time.Time, reason string) Trade {
	var pnl float64

	if pos.Direction == LONG {
		pnl = (exitPrice - pos.EntryPrice) * pos.Size
		slog.Debug("Calculating PnL for LONG", "exit_price", exitPrice, "entry_price", pos.EntryPrice, "size", pos.Size, "pnl", pnl)
	} else { // SHORT
		pnl = (pos.EntryPrice - exitPrice) * pos.Size
		slog.Debug("Calculating PnL for SHORT", "exit_price", exitPrice, "entry_price", pos.EntryPrice, "size", pos.Size, "pnl", pnl)
	}

	a.Balance += pnl

	slog.Info("Closed position", "id", pos.ID, "exit_price", exitPrice, "stop_loss", pos.StopLoss, "take_profit", pos.TakeProfit, "pnl", pnl, "reason", reason, "timestamp", exitTime)

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
