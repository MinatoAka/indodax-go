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

// OrderStatus mengambil detail order berdasarkan OrderID DAN Symbol Pair.
// Sesuai Doc Resmi: GET /api/v2/order membutuhkan parameter 'symbol' (mandatory).
func (c *Client) OrderStatus(ctx context.Context, pair string, orderID int64) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	params := map[string]string{
		"symbol":     strings.ToUpper(pair),           // ← WAJIB ADA
		"id":         strconv.FormatInt(orderID, 10),  // ← Doc sebut 'orderId', tapi query param biasanya 'id' atau 'orderId'. Cek doc lagi.
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	// NOTE: Di doc table tertulis "orderId", tapi di contoh legacy sering pakai "id". 
	// Untuk safety, coba pakai key "orderId" dulu. Kalau reject, ganti ke "id".
	// Berdasarkan pola Indodax V2 terbaru, kemungkinan besar key-nya "orderId".
	delete(params, "id") 
	params["orderId"] = strconv.FormatInt(orderID, 10)

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
// Sesuai Doc Resmi: POST endpoint -> Parameters via Request Body (NO "?" di URL).
func (c *Client) SubmitOrder(ctx context.Context, reqPayload SubmitOrderRequest) (OrderResult, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return OrderResult{}, fmt.Errorf("missing credentials for private endpoint")
	}

	// 1. Bangun Params Map — SESUAI DOC RESMI
	params := map[string]string{
		"symbol":     reqPayload.Symbol,
		"side":       strings.ToUpper(reqPayload.Side),   // ← KEY 'side', VALUE UPPERCASE
		"type":       strings.ToUpper(reqPayload.Type),   // ← KEY 'type' (LIMIT/MARKET)
		"timestamp":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"recvWindow": "5000",
	}

	// Conditional Fields
	if reqPayload.Price != "" {
		params["price"] = reqPayload.Price
	}
	if reqPayload.Quantity != "" {
		params["quantity"] = reqPayload.Quantity
	}
	if reqPayload.QuoteOrderQty != "" {
		params["quoteOrderQty"] = reqPayload.QuoteOrderQty
	}
	if reqPayload.TimeInForce != "" {
		params["timeInForce"] = reqPayload.TimeInForce
	}

	// 2. Signing Payload = Body String (sorted keys via hmacgo.BuildQuery)
	signingPayload := hmacgo.BuildQuery(params)
	signature := hmacgo.SignSHA256(signingPayload, c.secretKey)

	// 3. POST Request — URL BERSIH TANPA '?'
	endpoint := BaseURLPrivate + "/api/v2/order"
	bodyReader := strings.NewReader(signingPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bodyReader)
	if err != nil {
		return OrderResult{}, fmt.Errorf("build submit order request: %w", err)
	}

	// 4. Headers Wajib
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)

	// 5. Execute & Parse
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
