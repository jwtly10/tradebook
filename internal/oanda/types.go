package oanda

import "time"

type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// https://developer.oanda.com/rest-live-v20/pricing-ep/

type Candlestick struct {
	Time     string          `json:"time"`
	Bid      CandleStickData `json:"bid"`
	Ask      CandleStickData `json:"ask"`
	Mid      CandleStickData `json:"mid"`
	Volume   int             `json:"volume"`
	Complete bool            `json:"complete"`
}

type CandleStickData struct {
	O PriceValue `json:"o"`
	H PriceValue `json:"h"`
	L PriceValue `json:"l"`
	C PriceValue `json:"c"`
}

type PriceValue string
type InstrumentName string

type CandlestickGranularity string

type CandlestickResponse struct {
	Candles     []Candlestick          `json:"candles"`
	Instrument  InstrumentName         `json:"instrument"`
	Granularity CandlestickGranularity `json:"granularity"`
}

type OandaService struct {
	AccountId string
	ApiKey    string
	ApiUrl    string
}

type CandleRequest struct {
	Instrument  InstrumentName         `json:"instrument"`
	Granularity CandlestickGranularity `json:"granularity,omitempty"` // Default S5
	Count       int                    `json:"count,omitempty"`       // Default 500, max 5000
	From        time.Time              `json:"from"`                  // RFC 3339
	To          time.Time              `json:"to"`                    // RFC 3339
}
