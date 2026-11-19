package indodax

import (
	"encoding/json"
	"fmt"
)

// BalanceResult menyimpan saldo akun.
type BalanceResult struct {
	IDR      float64
	Asset    float64
	AssetRaw string
}

// SubmitOrderRequest — Payload untuk Market Order.
type SubmitOrderRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`          // BUY atau SELL
	QuoteOrderQty string `json:"quoteOrderQty,omitempty"` // Wajib untuk BUY (Nominal IDR)
	Quantity      string `json:"quantity,omitempty"`      // Wajib untuk SELL (Jumlah Aset)
}

// OrderResult menyimpan SELURUH output detail order dari Indodax API.
// Tipe data string digunakan untuk angka agar presisi tidak hilang saat parsing.
type OrderResult struct {
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
	Side          string `json:"side,omitempty"`
	Type          string `json:"type,omitempty"`
	OrigQty       string `json:"origQty,omitempty"`
	ExecutedQty   string `json:"executedQty"`
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

// --- Helper Methods untuk Ticker ---
func (t TickerResponse) GetLastPrice() (float64, error)  { return t.Ticker.Last.Float64() }
func (t TickerResponse) GetHighestBid() (float64, error) { return t.Ticker.Buy.Float64() }
func (t TickerResponse) GetLowestAsk() (float64, error)  { return t.Ticker.Sell.Float64() }
func (t TickerResponse) GetDayHigh() (float64, error)    { return t.Ticker.High.Float64() }
func (t TickerResponse) GetDayLow() (float64, error)     { return t.Ticker.Low.Float64() }

// TradeHistoryRequest adalah parameter untuk GET /api/v2/myTrades.
type TradeHistoryRequest struct {
	Symbol        string // Mandatory (e.g., btcidr)
	OrderID       string // Optional, isi dengan FullOrderID dari SubmitOrder (e.g., ethidr-market-6423)
	ClientOrderID string // Optional
	StartTime     int64  // Optional, Unix milliseconds
	EndTime       int64  // Optional, Unix milliseconds
	Limit         int    // Optional, min 10, max 1000, default 500
	Sort          string // Optional, "asc" atau "desc", default "desc"
}

// Trade merepresentasikan satu baris trade fill dari GET /api/v2/myTrades.
// Menggunakan json.Number untuk menjaga presisi desimal asli dari API.
type Trade struct {
	TradeID         string      `json:"tradeId"`
	OrderID         string      `json:"orderId"`
	ClientOrderID   string      `json:"clientOrderId"`
	Symbol          string      `json:"symbol"`
	Price           json.Number `json:"price"`
	Qty             json.Number `json:"qty"`
	QuoteQty        json.Number `json:"quoteQty"`
	Commission      json.Number `json:"commission"`
	CommissionAsset string      `json:"commissionAsset"`
	IsBuyer         bool        `json:"isBuyer"`
	IsMaker         bool        `json:"isMaker"`
	Time            int64       `json:"time"`
}

// tradeHistoryResponse adalah wrapper response dari endpoint myTrades.
type tradeHistoryResponse struct {
	Data []Trade `json:"data"`
}

// Error report API
type IndodaxError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *IndodaxError) Error() string {
	return fmt.Sprintf("indodax api error (code %d): %s", e.Code, e.Msg)
}

func (e *IndodaxError) IsRateLimit() bool           { return e.Code == -1003 }
func (e *IndodaxError) IsInvalidCredentials() bool  { return e.Code == -1002 }
func (e *IndodaxError) IsInvalidTimestamp() bool    { return e.Code == -1021 }
func (e *IndodaxError) IsInvalidSignature() bool    { return e.Code == -1022 }
func (e *IndodaxError) IsInsufficientBalance() bool { return e.Code == -4026 }
func (e *IndodaxError) IsOrderNotFound() bool       { return e.Code == -2013 || e.Code == 1112 }
func (e *IndodaxError) IsMarketSuspended() bool     { return e.Code == -1016 }
func (e *IndodaxError) IsOrderRejected() bool       { return e.Code == -2010 }
