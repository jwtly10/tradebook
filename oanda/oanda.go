package oanda

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	DefaultBaseUrl = "https://api-fxpractice.oanda.com"

	// Oanda granularities
	M1  CandlestickGranularity = "M1"
	M5  CandlestickGranularity = "M5"
	M15 CandlestickGranularity = "M15"
	M30 CandlestickGranularity = "M30"
	H1  CandlestickGranularity = "H1"
	H6  CandlestickGranularity = "H6"
	D   CandlestickGranularity = "D"
	W   CandlestickGranularity = "W"
	M   CandlestickGranularity = "M"

	// Oanda Instruments
	GBPUSD InstrumentName = "GBP_USD"
)

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

func NewOandaService(accountId, apiKey, apiUrl string) *OandaService {
	if apiUrl == "" {
		apiUrl = DefaultBaseUrl
	}

	return &OandaService{
		AccountId: accountId,
		ApiKey:    apiKey,
		ApiUrl:    apiUrl,
	}
}

type CandleRequest struct {
	Instrument  InstrumentName         `json:"instrument"`
	Granularity CandlestickGranularity `json:"granularity,omitempty"` // Default S5
	Count       int                    `json:"count,omitempty"`       // Default 500, max 5000
	From        time.Time              `json:"from"`                  // RFC 3339
	To          time.Time              `json:"to"`                    // RFC 3339
}

func (s *OandaService) FetchHistoricCandles(ctx context.Context, req CandleRequest) (*CandlestickResponse, error) {
	endpoint := s.ApiUrl + "/v3/accounts/" + s.AccountId + "/instruments/" + string(req.Instrument) + "/candles"

	params := url.Values{}
	if req.Granularity != "" {
		params.Add("granularity", string(req.Granularity))
	}
	if req.Count != 0 {
		params.Add("count", string(rune(req.Count)))
	}

	params.Add("from", strconv.FormatInt(req.From.Unix(), 10))
	params.Add("to", strconv.FormatInt(req.To.Unix(), 10))
	params.Add("includeFirst", "false")

	fullURL := endpoint + "?" + params.Encode()

	slog.Info("Fetching historic candles", "instrument", req.Instrument, "from", req.From, "to", req.To)
	slog.Debug("Request URL", "url", fullURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.ApiKey)
	httpReq.Header.Set("Accept-Datetime-Format", "RFC3339")

	slog.Debug("HTTP Request Headers", "headers", httpReq.Header)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Failed to read error response body",
				"statusCode", resp.StatusCode,
				"error", err)
			return nil, fmt.Errorf("failed to fetch candles: status code %d, could not read error body: %w", resp.StatusCode, err)
		}

		rawRespBody := string(bodyBytes)
		slog.Error("Failed to fetch candles: API returned an error status",
			"statusCode", resp.StatusCode,
			"rawResponse", rawRespBody)

		return nil, fmt.Errorf("failed to fetch candles: status code %d, API Response: %s", resp.StatusCode, rawRespBody)
	}

	var candleResp CandlestickResponse
	if err := json.NewDecoder(resp.Body).Decode(&candleResp); err != nil {
		return nil, fmt.Errorf("failed to decode candle response: %w", err)
	}

	return &candleResp, nil
}
