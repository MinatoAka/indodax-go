package indodax

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"      // ← Diperlukan untuk parsing query/body di validator
	"strings"
	"testing"

	"github.com/MinatoAka/hmacgo"
)

// setupTestServer membuat mock server dinamis.
// Karena BaseURL sekarang 'var', kita bisa override sebelum create client.
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	ts := httptest.NewServer(handler)
	
	// Simpan nilai asli untuk restore nanti (cleanup)
	oldPriv := BaseURLPrivate
	oldPub := BaseURLPublic
	
	// Override global vars agar Client mengarah ke localhost mock
	BaseURLPrivate = ts.URL + "/api/v2"
	BaseURLPublic = ts.URL + "/api"
	
	client, err := New("test-api-key", "test-secret-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	cleanup := func() {
		BaseURLPrivate = oldPriv
		BaseURLPublic = oldPub
		ts.Close()
	}
	
	return client, cleanup
}

func TestBalances_SigningAndParsing(t *testing.T) {
	expectedIDR := 1500000.0
	expectedAssetRaw := "0.00030391"
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 1. Validasi Method & Path
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/account") {
			t.Errorf("Expected path ending with /account, got %s", r.URL.Path)
		}

		// 2. Validasi Headers
		apiKey := r.Header.Get("X-APIKEY")
		signature := r.Header.Get("Sign")
		
		if apiKey != "test-api-key" {
			t.Errorf("Wrong API Key: %s", apiKey)
		}
		
		// 3. Validasi Signature Logic
		// Reconstruct expected signature from query params
		queryParams := r.URL.Query()
		paramsMap := make(map[string]string)
		for k, v := range queryParams {
			paramsMap[k] = v[0]
		}
		
		canonicalQuery := hmacgo.BuildQuery(paramsMap)
		expectedSig := hmacgo.SignSHA256(canonicalQuery, "test-secret-key")
		
		if signature != expectedSig {
			t.Errorf("Signature mismatch.\nGot:  %s\nWant: %s\nCanonical: %s", signature, expectedSig, canonicalQuery)
		}

		// 4. Return Mock Response
		resp := map[string]interface{}{
			"balances": []map[string]string{
				{"asset": "IDR", "free": "1500000"},
				{"asset": "ETH", "free": expectedAssetRaw},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, cleanup := setupTestServer(t, handler)
	defer cleanup()

	res, err := client.Balances(context.Background(), "ethidr")
	if err != nil {
		t.Fatalf("Balances error: %v", err)
	}

	if res.IDR != expectedIDR {
		t.Errorf("IDR Mismatch. Got: %f, Want: %f", res.IDR, expectedIDR)
	}
	if res.AssetRaw != expectedAssetRaw {
		t.Errorf("AssetRaw Mismatch. Got: %s, Want: %s", res.AssetRaw, expectedAssetRaw)
	}
}

func TestSubmitOrder_PostBodySigning(t *testing.T) {
	expectedQty := "0.00030391"
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/order") {
			t.Errorf("Expected path ending with /order, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("POST request should NOT have query parameters. Got: ?%s", r.URL.RawQuery)
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr := string(bodyBytes)
		
		formValues, _ := url.ParseQuery(bodyStr)
		qty := formValues.Get("quantity")
		if qty != expectedQty {
			t.Errorf("Quantity mismatch in body. Got: %s, Want: %s", qty, expectedQty)
		}

		signature := r.Header.Get("Sign")
		expectedSig := hmacgo.SignSHA256(bodyStr, "test-secret-key")
		
		if signature != expectedSig {
			t.Errorf("Signature mismatch for POST.\nGot:  %s\nWant: %s\nBody: %s", signature, expectedSig, bodyStr)
		}

		resp := map[string]interface{}{
			"orderId":             123456789,
			"executedQty":         "0.00030391",
			"cummulativeQuoteQty": "10.15",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, cleanup := setupTestServer(t, handler)
	defer cleanup()

	reqPayload := SubmitOrderRequest{
		Symbol:   "ethidr",
		Side:     "SELL",
		Quantity: expectedQty,
	}

	res, err := client.SubmitOrder(context.Background(), reqPayload)
	if err != nil {
		t.Fatalf("SubmitOrder error: %v", err)
	}

	if res.OrderID != 123456789 {
		t.Errorf("OrderID Mismatch. Got: %d", res.OrderID)
	}
}

func TestTicker_PublicNoAuth(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-APIKEY") != "" {
			t.Error("Public endpoint should not send X-APIKEY")
		}
		if r.Header.Get("Sign") != "" {
			t.Error("Public endpoint should not send Sign")
		}

		// REAL RESPONSE FORMAT: Nested Object "ticker"
		resp := map[string]interface{}{
			"ticker": map[string]interface{}{
				"buy":         "33569000", // Highest Bid
				"sell":        "33571000", // Lowest Ask
				"last":        "33570000", // Last Traded Price
				"high":        "34157000", // Day High
				"low":         "32901000", // Day Low
				"server_time": 1794876675, // Unix Timestamp
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, cleanup := setupTestServer(t, handler)
	defer cleanup()

	res, err := client.Ticker(context.Background(), "ethidr")
	if err != nil {
		t.Fatalf("Ticker error: %v", err)
	}

	// VALIDASI FIELD BARU (Nested Access)
	
	// 1. Cek Last Price via Helper Method
	lp, errLP := res.GetLastPrice()
	if errLP != nil {
		t.Fatalf("GetLastPrice failed: %v", errLP)
	}
	if lp != 33570000.0 {
		t.Errorf("LastPrice Mismatch. Got: %f, Want: 33570000.0", lp)
	}

	// 2. Cek Highest Bid (Field 'buy')
	hb, errHB := res.GetHighestBid()
	if errHB != nil {
		t.Fatalf("GetHighestBid failed: %v", errHB)
	}
	if hb != 33569000.0 {
		t.Errorf("HighestBid Mismatch. Got: %f, Want: 33569000.0", hb)
	}

	// 3. Cek Lowest Ask (Field 'sell')
	la, errLA := res.GetLowestAsk()
	if errLA != nil {
		t.Fatalf("GetLowestAsk failed: %v", errLA)
	}
	if la != 33571000.0 {
		t.Errorf("LowestAsk Mismatch. Got: %f, Want: 33571000.0", la)
	}

	// 4. Cek Day High & Low (Opsional tapi disarankan buat completeness)
	dh, _ := res.GetDayHigh()
	if dh != 34157000.0 {
		t.Errorf("DayHigh Mismatch. Got: %f, Want: 34157000.0", dh)
	}
	
	dl, _ := res.GetDayLow()
	if dl != 32901000.0 {
		t.Errorf("DayLow Mismatch. Got: %f, Want: 32901000.0", dl)
	}
}

func TestTradeHistory_SigningAndParsing(t *testing.T) {
	expectedSymbol := "ethidr"
	expectedFullOrderID := "ethidr-market-6423"
	expectedPrice := "33570000"
	expectedQty := "0.00030391"
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 1. Validasi Method & Path
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/myTrades") {
			t.Errorf("Expected path ending with /myTrades, got %s", r.URL.Path)
		}

		// 2. Validasi Headers
		apiKey := r.Header.Get("X-APIKEY")
		signature := r.Header.Get("Sign")
		
		if apiKey != "test-api-key" {
			t.Errorf("Wrong API Key: %s", apiKey)
		}
		
		// 3. Validasi Query Params & Signature Logic
		queryParams := r.URL.Query()
		
		// Cek parameter wajib & filter (Symbol harus lowercase sesuai implementasi di private.go)
		if gotSymbol := queryParams.Get("symbol"); gotSymbol != expectedSymbol {
			t.Errorf("Symbol mismatch. Got: %s, Want: %s", gotSymbol, expectedSymbol)
		}
		if gotFullOrderID := queryParams.Get("orderId"); gotFullOrderID != expectedFullOrderID {
			t.Errorf("OrderID mismatch. Got: %s, Want: %s", gotFullOrderID, expectedFullOrderID)
		}

		// Reconstruct signature
		paramsMap := make(map[string]string)
		for k, v := range queryParams {
			paramsMap[k] = v[0]
		}
		
		canonicalQuery := hmacgo.BuildQuery(paramsMap)
		expectedSig := hmacgo.SignSHA256(canonicalQuery, "test-secret-key")
		
		if signature != expectedSig {
			t.Errorf("Signature mismatch.\nGot:  %s\nWant: %s\nCanonical: %s", signature, expectedSig, canonicalQuery)
		}

		// 4. Return Mock Response (sesuai format TAPI v2)
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"tradeId":         "72057594037936570",
					"orderId":         expectedFullOrderID,
					"clientOrderId":   "clientx-1",
					"symbol":          expectedSymbol,
					"price":           expectedPrice,
					"qty":             expectedQty,
					"quoteQty":        "10.20",
					"commission":      "0.01",
					"commissionAsset": "idr",
					"isBuyer":         true,
					"isMaker":         false,
					"time":            1723442692520,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, cleanup := setupTestServer(t, handler)
	defer cleanup()

	req := TradeHistoryRequest{
		Symbol:  "ETHIDR", // Testing case-insensitivity (harusnya diconvert ke lowercase di private.go)
		FullOrderID: expectedFullOrderID,
	}

	trades, err := client.TradeHistory(context.Background(), req)
	if err != nil {
		t.Fatalf("TradeHistory error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	
	// Validasi parsing json.Number (Presisi tetap aman)
	if trade.Price.String() != expectedPrice {
		t.Errorf("Price Mismatch. Got: %s, Want: %s", trade.Price.String(), expectedPrice)
	}
	if trade.Qty.String() != expectedQty {
		t.Errorf("Qty Mismatch. Got: %s, Want: %s", trade.Qty.String(), expectedQty)
	}
	if !trade.IsBuyer {
		t.Errorf("IsBuyer should be true")
	}
	if trade.OrderID != expectedFullOrderID {
		t.Errorf("OrderID Mismatch. Got: %s, Want: %s", trade.OrderID, expectedFullOrderID)
	}
}

func TestTradeHistory_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Indodax sering return 200 OK tapi body-nya JSON error, atau 400 Bad Request.
		// checkAPIError handle keduanya selama body-nya JSON error.
		resp := map[string]interface{}{
			"code": -1121,
			"msg":  "Invalid symbol.",
		}
		json.NewEncoder(w).Encode(resp)
	}

	client, cleanup := setupTestServer(t, handler)
	defer cleanup()

	req := TradeHistoryRequest{
		Symbol: "invalidpair",
	}

	_, err := client.TradeHistory(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Type assertion ke IndodaxError
	apiErr, ok := err.(*IndodaxError)
	if !ok {
		t.Fatalf("Expected *IndodaxError, got %T: %v", err, err)
	}

	if apiErr.Code != -1121 {
		t.Errorf("Error Code Mismatch. Got: %d, Want: -1121", apiErr.Code)
	}
	if apiErr.Msg != "Invalid symbol." {
		t.Errorf("Error Msg Mismatch. Got: %s, Want: Invalid symbol.", apiErr.Msg)
	}
}
