package indodax

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
	Pair       string  `json:"pair"`        // e.g., "ETHIDR"
	LastPrice  float64 `json:"last_price"`  // Harga transaksi terakhir
	LowestAsk  float64 `json:"lowest_ask"`  // Ask terendah saat ini
	HighestBid float64 `json:"highest_bid"` // Bid tertinggi saat ini
	Volume     float64 `json:"volume"`      // Volume 24 jam (base asset)
	Value      float64 `json:"value"`       // Nilai total 24 jam (quote asset/IDR)
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
