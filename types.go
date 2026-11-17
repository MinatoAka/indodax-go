package indodax

import "encoding/json"

// BalanceResult menyimpan saldo akun.
type BalanceResult struct {
	IDR      float64
	Asset    float64
	AssetRaw string
}

// OrderResult menyimpan SELURUH output detail order dari Indodax API.
// Tipe data string digunakan untuk angka agar presisi tidak hilang saat parsing.
type OrderResult struct {
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
	Side          string `json:"side,omitempty"`
	Type          string `json:"type,omitempty"`
	Price         string `json:"price,omitempty"`     // Kosong jika MARKET order
	StopPrice     string `json:"stopPrice,omitempty"`
	OrigQty       string `json:"origQty,omitempty"`
	ExecutedQty   string `json:"executedQty"`
	Status        string `json:"status,omitempty"`    // Hanya muncul di Get Order
	Time          int64  `json:"time,omitempty"`      // Hanya muncul di Get Order
	FullOrderID   string `json:"fullOrderId,omitempty"`
}

type PairInfo struct {
	ID     string  `json:"id"`
	MinQty float64 `json:"trade_min_traded_currency"`
}

type TickerResponse struct {
	Ticker struct {
		Buy        json.Number `json:"buy"`
		Sell       json.Number `json:"sell"`
		Last       json.Number `json:"last"`
		High       json.Number `json:"high"`
		Low        json.Number `json:"low"`
		ServerTime int64       `json:"server_time"`
	} `json:"ticker"`
}

// SubmitOrderRequest — Payload untuk Market Order.
type SubmitOrderRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`          // BUY atau SELL
	QuoteOrderQty string `json:"quoteOrderQty,omitempty"` // Wajib untuk BUY (Nominal IDR)
	Quantity      string `json:"quantity,omitempty"`      // Wajib untuk SELL (Jumlah Aset)
}

// --- Helper Methods untuk Ticker ---
func (t TickerResponse) GetLastPrice() (float64, error)  { return t.Ticker.Last.Float64() }
func (t TickerResponse) GetHighestBid() (float64, error) { return t.Ticker.Buy.Float64() }
func (t TickerResponse) GetLowestAsk() (float64, error)  { return t.Ticker.Sell.Float64() }
func (t TickerResponse) GetDayHigh() (float64, error)    { return t.Ticker.High.Float64() }
func (t TickerResponse) GetDayLow() (float64, error)     { return t.Ticker.Low.Float64() }
