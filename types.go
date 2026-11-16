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

// SubmitOrderRequest adalah payload untuk membuat order baru.
// Type: "BUY" atau "SELL".
// Quantity: string presisi (dari FormatQty consumer).
// QuoteOrderQty: string IDR amount (untuk BUY market order).
type SubmitOrderRequest struct {
	Symbol        string `json:"symbol"`
	Type          string `json:"side"` // "BUY" or "SELL"
	Quantity      string `json:"quantity,omitempty"`
	QuoteOrderQty string `json:"quoteOrderQty,omitempty"`
	TimeInForce   string `json:"timeInForce,omitempty"` // e.g., "GTC", "IOC"
}
