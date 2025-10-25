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

	"github.com/jwtly10/tradebook/internal/types"
)

const (
	DefaultBaseUrl       = "https://api-fxpractice.oanda.com"
	MaxCandlesPerRequest = 4000 // Limit is 5000 but we maintain a buffer

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

var granularityToDuration = map[CandlestickGranularity]time.Duration{
	M1:  1 * time.Minute,
	M5:  5 * time.Minute,
	M15: 15 * time.Minute,
	M30: 30 * time.Minute,
	H1:  1 * time.Hour,
	H6:  6 * time.Hour,
	D:   24 * time.Hour,
	W:   7 * 24 * time.Hour,
	M:   30 * 24 * time.Hour, // Approx
}

func (g CandlestickGranularity) ToDuration() (time.Duration, error) {
	duration, ok := granularityToDuration[g]
	if !ok {
		return 0, fmt.Errorf("invalid granularity: %s", g)
	}
	return duration, nil
}

func (g CandlestickGranularity) MustToDuration() time.Duration {
	duration, err := g.ToDuration()
	if err != nil {
		panic(err)
	}
	return duration
}

func (g CandlestickGranularity) String() string {
	return string(g)
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

// FetchBars will iteratativly fetch all bars between 2 dates.
//
// Note: We are not limiting the number of candles returned here,
// so there is scope for memory issues if not used carefully.
func (s *OandaService) FetchBars(ctx context.Context, req CandleRequest) ([]types.Bar, error) {
	slog.Info("Initiating batched Oanda fetch", "instrument", req.Instrument, "from", req.From, "to", req.To, "period", req.Granularity.String())
	period, err := req.Granularity.ToDuration()
	if err != nil {
		return nil, err
	}

	if req.To.After(time.Now()) {
		req.To = time.Now()
		slog.Warn("Adjusted 'To' time to current time as it was in the future", "newTo", req.To)
	}

	var allBars []types.Bar
	currentFrom := req.From

	for currentFrom.Before(req.To) {
		batchTo := currentFrom.Add(period * time.Duration(MaxCandlesPerRequest))
		if batchTo.After(req.To) {
			batchTo = req.To
		}

		newReq := CandleRequest{
			Instrument:  req.Instrument,
			Granularity: req.Granularity,
			From:        currentFrom,
			To:          batchTo,
		}

		batchBars, err := s.fetchHistoricCandles(ctx, newReq)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch candles between %s and %s: %w", currentFrom, batchTo, err)
		}

		slog.Info("Found bars in latest fetch", "count", len(batchBars.Candles), "from", currentFrom, "to", batchTo)

		if len(batchBars.Candles) == 0 {
			break // No more data available
		}

		// Convert candles to bars
		convertedBars, err := s.candlesToBars(batchBars.Candles)
		if err != nil {
			return nil, fmt.Errorf("failed to convert candles to bars: %w", err)
		}

		allBars = append(allBars, convertedBars...)

		// Move to next batch
		currentFrom = convertedBars[len(convertedBars)-1].Timestamp.Add(period)
	}

	slog.Info("Completed fetching all oanda bars", "totalBars", len(allBars))
	return allBars, nil
}

func (s *OandaService) candlesToBars(candles []Candlestick) ([]types.Bar, error) {
	bars := make([]types.Bar, 0, len(candles))
	for _, candle := range candles {
		timestamp, err := time.Parse(time.RFC3339, candle.Time)
		if err != nil {
			return nil, fmt.Errorf("failed to parse candle time %s: %w", candle.Time, err)
		}

		o, err := strconv.ParseFloat(string(candle.Mid.O), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse candle open price %s: %w", candle.Mid.O, err)
		}
		h, err := strconv.ParseFloat(string(candle.Mid.H), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse candle high price %s: %w", candle.Mid.H, err)
		}
		l, err := strconv.ParseFloat(string(candle.Mid.L), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse candle low price %s: %w", candle.Mid.L, err)
		}
		c, err := strconv.ParseFloat(string(candle.Mid.C), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse candle close price %s: %w", candle.Mid.C, err)
		}
		v := float64(candle.Volume)

		bars = append(bars, types.Bar{
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
