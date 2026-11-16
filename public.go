package indodax

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Ticker mengambil harga terakhir pasangan trading.
// Pair harus lowercase sesuai format Indodax (misal "ethidr").
func (c *Client) Ticker(ctx context.Context, pair string) (float64, error) {
	endpoint := BaseURLPublic + "/api/ticker/" + url.PathEscape(strings.ToLower(pair))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build ticker request: %w", err)
	}

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("ticker fetch failed: %w", err)
	}

	var response struct {
		Ticker struct {
			Last string `json:"last"`
		} `json:"ticker"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("parse ticker JSON: %w", err)
	}

	lastPrice, err := strconv.ParseFloat(response.Ticker.Last, 64)
	if err != nil {
		return 0, fmt.Errorf("convert ticker price: %w", err)
	}

	return lastPrice, nil
}

// Pairs mengambil daftar semua pair beserta aturan minimum quantity.
// Mengembalikan slice PairInfo yang sudah difilter berdasarkan ID pair tertentu jika diperlukan oleh caller.
func (c *Client) Pairs(ctx context.Context) ([]PairInfo, error) {
	endpoint := BaseURLPublic + "/api/pairs"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build pairs request: %w", err)
	}

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pairs fetch failed: %w", err)
	}

	// Gunakan json.Number untuk handle mixed type (string/number) dari API
	type RawPairData struct {
		ID     string      `json:"id"`
		MinQty json.Number `json:"trade_min_traded_currency"`
	}

	var arr []RawPairData
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, fmt.Errorf("parse pairs JSON: %w", err)
	}

	results := make([]PairInfo, 0, len(arr))
	for _, p := range arr {
		minQty, errConv := p.MinQty.Float64()
		if errConv != nil || minQty <= 0 {
			// Skip invalid entries atau fallback? 
			// Keputusan: Skip silent, biarkan caller handle missing pair.
			continue 
		}
		results = append(results, PairInfo{
			ID:     strings.ToUpper(p.ID), // Normalize ID to uppercase for consistency
			MinQty: minQty,
		})
	}

	return results, nil
}
