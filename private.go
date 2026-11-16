package indodax

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MinatoAka/hmacgo"
)

// Balances mengambil saldo IDR dan Aset dasar untuk pair tertentu.
// AssetRaw dikembalikan sebagai string mentah dari API untuk menjaga presisi 8-digit.
func (c *Client) Balances(ctx context.Context, pair string) (BalanceResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return BalanceResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	params := map[string]string{
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	query := hmacgo.BuildQuery(params)
	signature := hmacgo.SignSHA256(query, c.secretKey)

	endpoint := BaseURLPrivate + "/api/v2/account?" + query

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return BalanceResult{}, fmt.Errorf("build balances request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return BalanceResult{}, fmt.Errorf("balances fetch failed: %w", err)
	}

	var response struct {
		Balances []struct {
			Asset string `json:"asset"`
			Free  string `json:"free"`
		} `json:"balances"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return BalanceResult{}, fmt.Errorf("parse balances JSON: %w", err)
	}

	pairUpper := strings.ToUpper(pair)
	baseAsset := strings.TrimSuffix(pairUpper, "IDR")

	var res BalanceResult
	for _, b := range response.Balances {
		if strings.EqualFold(b.Asset, "IDR") {
			v, _ := strconv.ParseFloat(b.Free, 64)
			res.IDR = v
			continue
		}
		if strings.EqualFold(b.Asset, baseAsset) {
			res.AssetRaw = b.Free // STRING MENTAH
			v, _ := strconv.ParseFloat(b.Free, 64)
			res.Asset = v
		}
	}

	if res.AssetRaw == "" {
		res.AssetRaw = "0"
	}

	return res, nil
}

// OrderStatus mengambil detail order berdasarkan OrderID.
func (c *Client) OrderStatus(ctx context.Context, orderID int64) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	params := map[string]string{
		"id":         strconv.FormatInt(orderID, 10),
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	query := hmacgo.BuildQuery(params)
	signature := hmacgo.SignSHA256(query, c.secretKey)

	endpoint := BaseURLPrivate + "/api/v2/order?" + query

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return OrderResult{}, fmt.Errorf("build order status request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return OrderResult{}, fmt.Errorf("order status fetch failed: %w", err)
	}

	var response struct {
		OrderID             int64  `json:"orderId"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("parse order status JSON: %w", err)
	}

	eq, _ := strconv.ParseFloat(response.ExecutedQty, 64)
	cq, _ := strconv.ParseFloat(response.CummulativeQuoteQty, 64)

	price := 0.0
	if eq > 0 && cq > 0 {
		price = cq / eq
	}

	return OrderResult{
		OrderID:     response.OrderID,
		ExecutedQty: eq,
		CumQuoteQty: cq,
		Price:       price,
	}, nil
}

// SubmitOrder mengirim order baru (BUY/SELL).
// Quantity harus string presisi (disarankan pakai utils.FormatQty dari consumer).
func (c *Client) SubmitOrder(ctx context.Context, reqPayload SubmitOrderRequest) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	// Siapkan params untuk signing
	params := map[string]string{
		"symbol":     reqPayload.Symbol,
		"side":       reqPayload.Type,
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	// Masukkan optional fields ke params jika ada isinya
	if reqPayload.Quantity != "" {
		params["quantity"] = reqPayload.Quantity
	}
	if reqPayload.QuoteOrderQty != "" {
		params["quoteOrderQty"] = reqPayload.QuoteOrderQty
	}
	if reqPayload.TimeInForce != "" {
		params["timeInForce"] = reqPayload.TimeInForce
	}

	query := hmacgo.BuildQuery(params)
	signature := hmacgo.SignSHA256(query, c.secretKey)

	endpoint := BaseURLPrivate + "/api/v2/order?" + query

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return OrderResult{}, fmt.Errorf("build submit order request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return OrderResult{}, fmt.Errorf("submit order failed: %w", err)
	}

	var response struct {
		OrderID             int64  `json:"orderId"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("parse submit order JSON: %w", err)
	}

	eq, _ := strconv.ParseFloat(response.ExecutedQty, 64)
	cq, _ := strconv.ParseFloat(response.CummulativeQuoteQty, 64)

	price := 0.0
	if eq > 0 && cq > 0 {
		price = cq / eq
	}

	return OrderResult{
		OrderID:     response.OrderID,
		ExecutedQty: eq,
		CumQuoteQty: cq,
		Price:       price,
	}, nil
}
