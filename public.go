package indodax

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url" // ← DIPAKAI untuk PathEscape
	"strings"
)

// BaseURLPublic adalah domain untuk endpoint tanpa auth (/api/*).
var BaseURLPublic = "https://indodax.com"

// Ticker mengambil data harga pasar lengkap untuk pair tertentu.
func (c *Client) Ticker(ctx context.Context, pair string) (TickerResponse, error) {
	if strings.TrimSpace(pair) == "" {
		return TickerResponse{}, fmt.Errorf("pair cannot be empty")
	}

	endpoint := BaseURLPublic + "/api/ticker/" + url.PathEscape(strings.ToLower(pair))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TickerResponse{}, fmt.Errorf("build ticker request: %w", err)
	}

	body, err := c.doRequest(ctx, req) 
	if err != nil {
		return TickerResponse{}, fmt.Errorf("ticker fetch failed: %w", err)
	}

	var result TickerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return TickerResponse{}, fmt.Errorf("parse ticker JSON: %w", err)
	}

	return result, nil
}

// Pairs mengambil daftar semua pair beserta aturan minimum quantity.
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
			continue 
		}
		results = append(results, PairInfo{
			ID:     strings.ToUpper(p.ID),
			MinQty: minQty,
		})
	}

	return results, nil
}
