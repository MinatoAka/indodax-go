package indodax

import "encoding/json"

// BalanceResult menyimpan saldo akun.
// AssetRaw TETAP STRING: exchange truth, no float lossy, no buffer/floor/trim.
// Normalisasi qty (FormatQty) sengaja TIDAK masuk SDK — itu concern bot/consumer.
type BalanceResult struct {
	IDR      float64 // Untuk validasi kasar/gate MinQty
	Asset    float64 // Untuk kalkulasi internal non-order
	AssetRaw string  // String mentah dari API, untuk kirim order SELL presisi
}

// OrderResult menyimpan detail eksekusi order.
type OrderResult struct {
	OrderID     int64   `json:"orderId"`
	ExecutedQty float64 `json:"executedQty"`
	CumQuoteQty float64 `json:"cummulativeQuoteQty"`
	Price       float64 // Derived: CumQuoteQty / ExecutedQty (jika > 0)
}

// PairInfo menyimpan metadata pair publik.
type PairInfo struct {
	ID     string  `json:"id"`
	MinQty float64 `json:"trade_min_traded_currency"`
}

// TickerResponse menyimpan data harga pasar terbaru untuk satu pair.
// Field-field ini diambil langsung dari endpoint public /api/ticker/{pair}.
type TickerResponse struct {
	Pair       string      `json:"pair"`
	LastPrice  json.Number `json:"last_price"`   // ← JSON.NUMBER
	LowestAsk  json.Number `json:"lowest_ask"`   // ← JSON.NUMBER
	HighestBid json.Number `json:"highest_bid"`  // ← JSON.NUMBER
	Volume     json.Number `json:"volume"`       // ← JSON.NUMBER
	Value      json.Number `json:"value"`        // ← JSON.NUMBER
}

// SubmitOrderRequest adalah payload untuk membuat order baru.
// Sesuai Doc Resmi: Key wajib adalah 'side', bukan 'type'.
type SubmitOrderRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`          // ← GANTI NAMA FIELD & TAG
	Type          string `json:"type"`          // ← TAMBAH FIELD TYPE (LIMIT/MARKET)
	Price         string `json:"price,omitempty"`     // ← TAMBAH UNTUK LIMIT ORDER
	Quantity      string `json:"quantity,omitempty"`
	QuoteOrderQty string `json:"quoteOrderQty,omitempty"`
	TimeInForce   string `json:"timeInForce,omitempty"`
}


// GetLastPrice mengkonversi LastPrice dari json.Number ke float64.
func (t TickerResponse) GetLastPrice() (float64, error) {
	return t.LastPrice.Float64()
}

// GetLowestAsk mengkonversi LowestAsk dari json.Number ke float64.
func (t TickerResponse) GetLowestAsk() (float64, error) {
	return t.LowestAsk.Float64()
}

// ... similarly for HighestBid, Volume, Value ...
