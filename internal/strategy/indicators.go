package strategy

import (
	"log/slog"
	"math"

	"github.com/jwtly10/tradebook/internal/logging"
	"github.com/jwtly10/tradebook/internal/types"
)

var (
	atrLog       = logging.New("atr")
	atrCandleLog = logging.New("atrcandle")
	emaLog       = logging.New("ema")
	smaLog       = logging.New("sma")
)

// EMA - Exponential Moving Average
type EMA struct {
	period int
	value  float64
	alpha  float64
	init   bool
}

func NewEMA(period int) *EMA {
	return &EMA{
		period: period,
		alpha:  2.0 / float64(period+1),
	}
}

func (e *EMA) Update(price float64) {
	oldValue := e.value
	if !e.init {
		e.value = price
		e.init = true
		emaLog.Debug("EMA initialized", "period", e.period, "price", price, "value", e.value)
	} else {
		e.value = (price * e.alpha) + (e.value * (1 - e.alpha))
		emaLog.Debug("EMA updated", "period", e.period, "price", price, "oldValue", oldValue, "newValue", e.value)
	}
}

func (e *EMA) Value() float64 {
	return e.value
}

func (e *EMA) Ready() bool {
	return e.init
}

// SMA - Simple Moving Average
type SMA struct {
	period int
	values []float64
}

func NewSMA(period int) *SMA {
	return &SMA{
		period: period,
		values: make([]float64, 0, period),
	}
}

func (s *SMA) Update(price float64) {
	s.values = append(s.values, price)
	if len(s.values) > s.period {
		s.values = s.values[1:]
	}
	smaLog.Debug("SMA updated", "period", s.period, "price", price, "value", s.Value(), "ready", s.Ready())
}

func (s *SMA) Value() float64 {
	if len(s.values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range s.values {
		sum += v
	}
	return sum / float64(len(s.values))
}

func (s *SMA) Ready() bool {
	return len(s.values) >= s.period
}

// ATR - Average True Range
type ATR struct {
	period  int
	ema     *EMA
	prevBar *types.Bar
	ready   bool
	warmup  int
}

func NewATR(period int) *ATR {
	return &ATR{
		period: period,
		ema:    NewEMA(period),
		warmup: 0,
	}
}

func (a *ATR) Update(bar types.Bar) {
	if a.prevBar == nil {
		a.prevBar = &bar
		atrLog.Debug("ATR first bar", "timestamp", bar.Timestamp, "close", bar.Close)
		return
	}

	// True Range = max of:
	// 1. Current High - Current Low
	// 2. |Current High - Previous Close|
	// 3. |Current Low - Previous Close|
	tr1 := bar.High - bar.Low
	tr2 := math.Abs(bar.High - a.prevBar.Close)
	tr3 := math.Abs(bar.Low - a.prevBar.Close)

	tr := math.Max(tr1, math.Max(tr2, tr3))

	atrLog.Debug("ATR calculation",
		"timestamp", bar.Timestamp,
		"tr1", tr1,
		"tr2", tr2,
		"tr3", tr3,
		"trueRange", tr,
		"prevATR", a.ema.Value())

	a.ema.Update(tr)
	a.prevBar = &bar

	a.warmup++
	if a.warmup >= a.period {
		a.ready = true
	}

	atrLog.Debug("ATR updated",
		"timestamp", bar.Timestamp,
		"value", a.Value(),
		"ready", a.Ready(),
		"warmup", a.warmup)
}

func (a *ATR) Value() float64 {
	return a.ema.Value()
}

func (a *ATR) Ready() bool {
	return a.ready
}

// ATRCandle - Custom indicator that detects ATR violations with engulfing pattern
type ATRCandle struct {
	atrPeriod        int
	atrMultiplier    float64
	withRelativeSize bool
	relativeSize     float64
	atr              *ATR
	prevBar          *types.Bar
	violationCount   int
}

func NewATRCandle(atrPeriod int, atrMultiplier, relativeSize float64, withRelativeSize bool) *ATRCandle {
	return &ATRCandle{
		atrPeriod:        atrPeriod,
		atrMultiplier:    atrMultiplier,
		withRelativeSize: withRelativeSize,
		relativeSize:     relativeSize,
		atr:              NewATR(atrPeriod),
	}
}

func (a *ATRCandle) Update(bar types.Bar) {
	a.atr.Update(bar)

	if a.atr.Ready() {
		if a.checkViolation(bar) {
			atrCandleLog.Info("ATRCandle violation detected", "timestamp", bar.Timestamp, "close", bar.Close, "open", bar.Open)
			a.violationCount++
		} else {
			// TODO: I don't *love* this - it means we do not have access to historic indicator values
			// but may not be needed in all honesty - this is way more performant so potentially
			// acceptable tradeoff, until we need otherwise
			a.violationCount = 0
		}
	}

	a.prevBar = &bar
}

func (a *ATRCandle) checkViolation(currentBar types.Bar) bool {
	// Absolute difference between close and open (candle body size)
	absDiff := math.Abs(currentBar.Close - currentBar.Open)

	// ATR threshold
	atrThreshold := a.atr.Value() * a.atrMultiplier
	atrViolation := absDiff > atrThreshold

	atrCandleLog.Debug("ATRCandle check",
		"timestamp", currentBar.Timestamp,
		"candleSize", absDiff,
		"atrValue", a.atr.Value(),
		"atrThreshold", atrThreshold,
		"withRelativeSize", a.withRelativeSize,
		"atrViolation", atrViolation)

	// Can't check engulfing pattern without previous bar
	if a.prevBar == nil {
		slog.Warn("Previous bar is nil, cannot check engulfing pattern")
		return atrViolation
	}

	if !a.withRelativeSize {
		return atrViolation
	}

	// Previous candle body size
	prevAbsDiff := math.Abs(a.prevBar.Close - a.prevBar.Open)

	// Relative threshold (current candle must be X times larger than previous)
	relativeThreshold := prevAbsDiff * a.relativeSize
	isEngulfing := absDiff > relativeThreshold

	atrCandleLog.Debug("ATRCandle engulfing check",
		"prevCandleSize", prevAbsDiff,
		"relativeThreshold", relativeThreshold,
		"isEngulfing", isEngulfing,
		"finalResult", atrViolation && isEngulfing)

	// Both conditions must be true
	return atrViolation && isEngulfing
}

// Value returns 1 if last bar was a violation, 0 otherwise
func (a *ATRCandle) Value() float64 {
	if a.violationCount > 0 {
		return 1.0
	}
	return 0.0
}

func (a *ATRCandle) Ready() bool {
	return a.atr.Ready() && a.prevBar != nil
}
