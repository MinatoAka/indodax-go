package indodax

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MinatoAka/hmacgo"
)

// setupTestServer membuat mock server yang memvalidasi request sesuai pola Indodax.
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	ts := httptest.NewServer(handler)
	
	// Override BaseURLs global untuk test ini saja
	oldPrivate := BaseURLPrivate
	oldPublic := BaseURLPublic
	
	BaseURLPrivate = ts.URL + "/api/v2"
	BaseURLPublic = ts.URL + "/api"
	
	client, err := New("test-api-key", "test-secret-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	return client, ts
}

// cleanupRestore mengembalikan konstanta base URL setelah test selesai.
func cleanupRestore(oldPriv, oldPub string) func() {
	return func() {
		BaseURLPrivate = oldPriv
		BaseURLPublic = oldPub
	}
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
		
		// Build canonical query string (sorted)
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

	client, ts := setupTestServer(t, handler)
	defer ts.Close()
	defer cleanupRestore(BaseURLPrivate, BaseURLPublic)() // Note: In real test, capture originals before override

	// Call method
	res, err := client.Balances(context.Background(), "ethidr")
	if err != nil {
		t.Fatalf("Balances error: %v", err)
	}

	// Assert Results
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
		// 1. Validasi Method & Path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/order") {
			t.Errorf("Expected path ending with /order, got %s", r.URL.Path)
		}
		// CRITICAL: Ensure NO QUERY PARAMS in URL for POST
		if r.URL.RawQuery != "" {
			t.Errorf("POST request should NOT have query parameters. Got: ?%s", r.URL.RawQuery)
		}

		// 2. Read Body
		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr := string(bodyBytes)
		
		// Parse form data manually to verify content
		formValues, _ := url.ParseQuery(bodyStr)
		
		qty := formValues.Get("quantity")
		if qty != expectedQty {
			t.Errorf("Quantity mismatch in body. Got: %s, Want: %s", qty, expectedQty)
		}

		// 3. Validasi Signature Logic (Payload = BODY STRING)
		signature := r.Header.Get("Sign")
		expectedSig := hmacgo.SignSHA256(bodyStr, "test-secret-key")
		
		if signature != expectedSig {
			t.Errorf("Signature mismatch for POST.\nGot:  %s\nWant: %s\nBody: %s", signature, expectedSig, bodyStr)
		}

		// 4. Return Mock Response
		resp := map[string]interface{}{
			"orderId":             123456789,
			"executedQty":         "0.00030391",
			"cummulativeQuoteQty": "10.15",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, ts := setupTestServer(t, handler)
	defer ts.Close()
	defer cleanupRestore(BaseURLPrivate, BaseURLPublic)()

	reqPayload := SubmitOrderRequest{
		Symbol:   "ethidr",
		Type:     "SELL",
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
		// 1. Validasi No Auth Headers
		if r.Header.Get("X-APIKEY") != "" {
			t.Error("Public endpoint should not send X-APIKEY")
		}
		if r.Header.Get("Sign") != "" {
			t.Error("Public endpoint should not send Sign")
		}

		// 2. Return Mock Ticker
		resp := map[string]interface{}{
			"ticker": map[string]string{
				"last": "33570000",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, ts := setupTestServer(t, handler)
	defer ts.Close()
	defer cleanupRestore(BaseURLPrivate, BaseURLPublic)()

	price, err := client.Ticker(context.Background(), "ethidr")
	if err != nil {
		t.Fatalf("Ticker error: %v", err)
	}

	if price != 33570000.0 {
		t.Errorf("Price Mismatch. Got: %f", price)
	}
}

func TestOrderStatus_SigningAndParsing(t *testing.T) {
	expectedOrderID := int64(987654321)
	expectedExecutedQty := "0.00030391"
	expectedCumQuoteQty := "10.15" // ETH price ~33k IDR * 0.0003 qty ≈ 10.15
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 1. Validasi Method & Path
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/order") {
			t.Errorf("Expected path ending with /order, got %s", r.URL.Path)
		}

		// 2. Validasi Headers
		apiKey := r.Header.Get("X-APIKEY")
		signature := r.Header.Get("Sign")
		
		if apiKey != "test-api-key" {
			t.Errorf("Wrong API Key: %s", apiKey)
		}
		
		// 3. Validasi Signature Logic
		// Payload signing HARUS mencakup 'id' parameter sesuai doc resmi
		queryParams := r.URL.Query()
		paramsMap := make(map[string]string)
		for k, v := range queryParams {
			paramsMap[k] = v[0]
		}
		
		// Pastikan 'id' ada di paramsMap sebelum build canonical query
		if _, ok := paramsMap["id"]; !ok {
			t.Error("Missing 'id' parameter in query string for OrderStatus signing")
		}
		
		canonicalQuery := hmacgo.BuildQuery(paramsMap)
		expectedSig := hmacgo.SignSHA256(canonicalQuery, "test-secret-key")
		
		if signature != expectedSig {
			t.Errorf("Signature mismatch.\nGot:  %s\nWant: %s\nCanonical: %s", signature, expectedSig, canonicalQuery)
		}

		// 4. Return Mock Response
		resp := map[string]interface{}{
			"orderId":             expectedOrderID,
			"executedQty":         expectedExecutedQty,
			"cummulativeQuoteQty": expectedCumQuoteQty,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	client, ts := setupTestServer(t, handler)
	defer ts.Close()
	defer cleanupRestore(BaseURLPrivate, BaseURLPublic)()

	res, err := client.OrderStatus(context.Background(), expectedOrderID)
	if err != nil {
		t.Fatalf("OrderStatus error: %v", err)
	}

	if res.OrderID != expectedOrderID {
		t.Errorf("OrderID Mismatch. Got: %d, Want: %d", res.OrderID, expectedOrderID)
	}
	
	// Verifikasi kalkulasi harga derived (cq / eq)
	expectedPrice := 10.15 / 0.00030391 
	if abs(res.Price - expectedPrice) > 0.01 { // tolerance float precision
		t.Errorf("Derived Price Mismatch. Got: %f, Want: %f", res.Price, expectedPrice)
	}
}

func TestPairs_ParsingAndNormalization(t *testing.T) {
	mockPairs := []map[string]interface{}{
		{"id": "ethidr", "trade_min_traded_currency": json.Number("0.0001")},
		{"id": "BTCIDR", "trade_min_traded_currency": json.Number("0.00001")},
		{"id": "invalid_pair", "trade_min_traded_currency": json.Number("0")}, // Should be skipped
	}
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 1. Validasi No Auth Headers
		if r.Header.Get("X-APIKEY") != "" || r.Header.Get("Sign") != "" {
			t.Error("Public endpoint should not send auth headers")
		}

		// 2. Return Mock Array
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPairs)
	}

	client, ts := setupTestServer(t, handler)
	defer ts.Close()
	defer cleanupRestore(BaseURLPrivate, BaseURLPublic)()

	results, err := client.Pairs(context.Background())
	if err != nil {
		t.Fatalf("Pairs error: %v", err)
	}

	// Expect 2 valid entries (invalid_pair skipped because minQty == 0)
	if len(results) != 2 {
		t.Errorf("Expected 2 pairs, got %d", len(results))
	}

	// Check normalization (uppercase IDs)
	foundETH := false
	foundBTC := false
	for _, p := range results {
		if p.ID == "ETHIDR" && p.MinQty == 0.0001 {
			foundETH = true
		}
		if p.ID == "BTCIDR" && p.MinQty == 0.00001 {
			foundBTC = true
		}
	}
	
	if !foundETH {
		t.Error("ETHIDR pair missing or incorrect after normalization")
	}
	if !foundBTC {
		t.Error("BTCIDR pair missing or incorrect after normalization")
	}
}

// Helper untuk absolute difference float
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
