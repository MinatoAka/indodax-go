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
		if r.Header.Get("X-APIKEY") != "" {
			t.Error("Public endpoint should not send X-APIKEY")
		}
		if r.Header.Get("Sign") != "" {
			t.Error("Public endpoint should not send Sign")
		}

		resp := map[string]interface{}{
			"pair":       "ETHIDR",
			"last_price": "33570000", // String sesuai format JSON Indodax biasanya
			"lowest_ask": "33571000",
			"highest_bid":"33569000",
			"volume":     "120.5",
			"value":      "4045282500",
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

    // Convert dulu baru compare
    lp, _ := res.LastPrice.Float64()
    if lp != 33570000.0 {
        t.Errorf("LastPrice Mismatch. Got: %f, Want: 33570000.0", lp)
    }

    hb, _ := res.HighestBid.Float64()
    if hb != 33569000.0 {
        t.Errorf("HighestBid Mismatch. Got: %f, Want: 33569000.0", hb)
    }
}
