package types

import "time"

const (
	BUY  Action = "BUY"
	SELL Action = "SELL"

	OPEN Type = "OPEN_TRADE"
)

type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

type Action string
type Type string

type Signal struct {
	Type   Type   // OPEN_TRADE
	Action Action // "BUY", "SELL"
	Price  float64
	TP     float64
	SL     float64
	Size   float64 // Lot size
}
