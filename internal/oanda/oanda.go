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
	NAS100 InstrumentName = "NAS100_USD"
)

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

func (s *OandaService) FetchBars(ctx context.Context, req CandleRequest) ([]Bar, error) {
	resp, err := s.fetchHistoricCandles(ctx, req)
	if err != nil {
		return nil, err
	}

	bars := make([]Bar, 0, len(resp.Candles))
	for _, candle := range resp.Candles {
		timestamp, err := time.Parse(time.RFC3339, candle.Time)
		if err != nil {
			slog.Warn("Failed to parse candle time", "time", candle.Time, "error", err)
			continue
		}

		o, _ := strconv.ParseFloat(string(candle.Mid.O), 64)
		h, _ := strconv.ParseFloat(string(candle.Mid.H), 64)
		l, _ := strconv.ParseFloat(string(candle.Mid.L), 64)
		c, _ := strconv.ParseFloat(string(candle.Mid.C), 64)
		v := float64(candle.Volume)

		bars = append(bars, Bar{
			Timestamp: timestamp,
			Open:      o,
			High:      h,
			Low:       l,
			Close:     c,
			Volume:    v,
		})
	}

	return bars, nil
}

func (s *OandaService) fetchHistoricCandles(ctx context.Context, req CandleRequest) (*CandlestickResponse, error) {
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
