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

// BaseURLPrivate adalah domain untuk endpoint signed (/api/v2/*).
// Wajib pakai header X-APIKEY & Sign.
var BaseURLPrivate = "https://api.indodax.com"

// Balances mengambil saldo IDR dan Aset dasar untuk pair tertentu.
// Sesuai Doc Resmi: GET endpoint -> Parameters via Query String (pakai "?").
func (c *Client) Balances(ctx context.Context, pair string) (BalanceResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return BalanceResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	params := map[string]string{
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	query := hmacgo.BuildQuery(params) // Menghasilkan "recvWindow=5000&timestamp=..."
	signature := hmacgo.SignSHA256(query, c.secretKey)

	// FIX: Tambahin "?" sebagai pemisah Path vs Query String sesuai RFC & Doc Indodax
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

// OrderStatus mengambil detail order berdasarkan OrderID atau ClientOrderID (origClientOrderId).
// Sesuai Doc Resmi: GET /api/v2/order membutuhkan 'symbol' dan salah satu identifier (orderId / origClientOrderId).
func (c *Client) OrderStatus(ctx context.Context, pair string, orderID int64, clientOrderID string) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	params := map[string]string{
		"symbol":     strings.ToUpper(pair),
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	// Masukkan orderId jika ada, jika tidak cek clientOrderID (origClientOrderId)
	if orderID > 0 {
		params["orderId"] = strconv.FormatInt(orderID, 10)
	} else if clientOrderID != "" {
		params["origClientOrderId"] = clientOrderID
	} else {
		return OrderResult{}, fmt.Errorf("wajib menyertakan orderID atau clientOrderID")
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

	var response OrderResult
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("parse order status JSON: %w", err)
	}

	return response, nil
}

// SubmitOrder mengirim order baru (BUY/SELL).
// Sesuai Doc Resmi: POST endpoint -> Parameters via Request Body (NO "?" di URL).
func (c *Client) SubmitOrder(ctx context.Context, reqPayload SubmitOrderRequest) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	// Bangun Params dasar (Type di-hardcode MARKET)
	params := map[string]string{
		"symbol":     reqPayload.Symbol,
		"side":       strings.ToUpper(reqPayload.Side),
		"type":       "MARKET",
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	// Hanya masukkan quantity (untuk SELL) atau quoteOrderQty (untuk BUY)
	if reqPayload.Quantity != "" {
		params["quantity"] = reqPayload.Quantity
	}
	if reqPayload.QuoteOrderQty != "" {
		params["quoteOrderQty"] = reqPayload.QuoteOrderQty
	}

	signingPayload := hmacgo.BuildQuery(params)
	signature := hmacgo.SignSHA256(signingPayload, c.secretKey)

	endpoint := BaseURLPrivate + "/api/v2/order"
	bodyReader := strings.NewReader(signingPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bodyReader)
	if err != nil {
		return OrderResult{}, fmt.Errorf("build submit order request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return OrderResult{}, fmt.Errorf("submit order failed: %w", err)
	}

	var response OrderResult
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("parse submit order JSON: %w", err)
	}

	return response, nil
}
