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

// IndodaxError merepresentasikan format error standar dari Indodax API v2
type IndodaxError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// checkAPIError adalah helper untuk mendeteksi jika response sebenarnya adalah error
func checkAPIError(body []byte) error {
	var errResp IndodaxError
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != 0 {
		return fmt.Errorf("indodax api error (code %d): %s", errResp.Code, errResp.Msg)
	}
	return nil
}

// setPrivateHeaders adalah helper untuk set header auth (eliminasi duplikasi)
// isPost = true akan menambah Content-Type untuk POST request
func (c *Client) setPrivateHeaders(req *http.Request, signature string, isPost bool) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-APIKEY", c.apiKey)
	req.Header.Set("Sign", signature)
	if isPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

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

	c.setPrivateHeaders(req, signature, false)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return BalanceResult{}, fmt.Errorf("balances fetch failed (pair: %s): %w", pair, err)
	}

	// Cek apakah ini sebenarnya response error dari Indodax
	if err := checkAPIError(body); err != nil {
		return BalanceResult{}, err
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

	c.setPrivateHeaders(req, signature, true)

	body, err := c.doRequest(ctx, req)
	if err != nil {
		return OrderResult{}, fmt.Errorf("submit order failed (symbol: %s, side: %s): %w", 
			reqPayload.Symbol, reqPayload.Side, err)
	}

	// Cek apakah ini sebenarnya response error dari Indodax
	if err := checkAPIError(body); err != nil {
		return OrderResult{}, err
	}

	var response OrderResult
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("parse submit order JSON: %w", err)
	}

	return response, nil
}
