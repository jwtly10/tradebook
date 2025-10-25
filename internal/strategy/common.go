package strategy

import (
	"log/slog"

	"github.com/jwtly10/tradebook/internal/account"
	"github.com/jwtly10/tradebook/internal/oanda"
	"github.com/jwtly10/tradebook/internal/types"
)

type Base struct {
	// Execution context
	symbol string
	period string

	riskPercentage float64
	riskRatio      float64
	balanceToRisk  float64
	stopLossPips   int
}

type Strategy interface {
	OnBar(bars []types.Bar, currentIndex int, account *account.Account) []types.Signal

	GetRiskPercentage() float64
	GetRiskRatio() float64
	GetBalanceToRisk() float64
	GetStopLossPips() int
	GetSymbol() string
	GetPeriod() string
}

func NewBaseStrategy(symbol, period string, riskPercentage, riskRatio, balanceToRisk float64, stopLossPips int) *Base {
	return &Base{
		symbol,
		period,
		riskPercentage,
		riskRatio,
		balanceToRisk,
		stopLossPips,
	}
}

// Abs returns the absolute value of a float64
func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// IndicatorsReady calls .Ready() on all indicators and returns true if all are ready
func IndicatorsReady(indicators ...Indicator) bool {
	for _, ind := range indicators {
		if !ind.Ready() {
			return false
		}
	}
	return true
}

// GetPipsFromInstr returns the pip size for a given instrument
func GetPipsFromInstr(ins string) float64 {
	// TODO: This method will support all broker/data sources
	// for now just NAS100
	if ins == string(oanda.NAS100) {
		return 0.1
	} else {
		panic("GetPipsFromInstr: Unsupported instrument " + ins)
	}
}

// pipsToPrice converts pips to price units based on the symbol's pip size
func PipsToPrice(pips int, pipSize float64) float64 {
	return float64(pips) * pipSize
}

// OpenLong creates a long trade signal based on the strategy configuration and current bar
func OpenLong(s Strategy, bar types.Bar, acc *account.Account) types.Signal {
	entryPrice := bar.Close
	stopLoss := entryPrice - PipsToPrice(s.GetStopLossPips(), GetPipsFromInstr(s.GetSymbol()))
	takeProfit := entryPrice + PipsToPrice(s.GetStopLossPips(), GetPipsFromInstr(s.GetSymbol()))*s.GetRiskRatio()

	size := calculatePositionSize(s, acc, entryPrice, stopLoss)

	return types.Signal{
		Type:   types.OPEN,
		Action: types.BUY,
		Price:  entryPrice,
		SL:     stopLoss,
		TP:     takeProfit,
		Size:   size,
	}
}

// OpenShort creates a short trade signal based on the strategy configuration and current bar
func OpenShort(s Strategy, bar types.Bar, acc *account.Account) types.Signal {
	entryPrice := bar.Close
	stopLoss := entryPrice + PipsToPrice(s.GetStopLossPips(), GetPipsFromInstr(s.GetSymbol()))
	takeProfit := entryPrice - PipsToPrice(s.GetStopLossPips(), GetPipsFromInstr(s.GetSymbol()))*s.GetRiskRatio()

	size := calculatePositionSize(s, acc, entryPrice, stopLoss)

	return types.Signal{
		Type:   types.OPEN,
		Action: types.SELL,
		Price:  entryPrice,
		SL:     stopLoss,
		TP:     takeProfit,
		Size:   size,
	}
}

// calculatePositionSize calculates the position size based on risk management parameters
// based on the strategy and account state
func calculatePositionSize(s Strategy, acc *account.Account, entryPrice, stopLoss float64) float64 {
	// Using static balance if available
	// (So ever trade has the same risk - it doesn't scale based on balance)
	balanceToUse := s.GetBalanceToRisk()
	if balanceToUse == 0 {
		balanceToUse = acc.Balance
	}

	riskAmount := balanceToUse * (s.GetRiskPercentage() / 100)
	stopDistance := Abs(entryPrice - stopLoss)
	size := riskAmount / stopDistance
	slog.Debug("Calculated position size", "size", size, "riskAmount", riskAmount, "entryPrice", entryPrice, "stopLoss", stopLoss, "stopDistance", stopDistance)

	return size
}
