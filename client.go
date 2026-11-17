package indodax

import (
	"strconv"
	"strings"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MinatoAka/myhttp"
)

// Client adalah wrapper utama untuk interaksi Indodax API.
// Ia memegang kredensial dan transport instance.
type Client struct {
	apiKey    string
	secretKey string
	http      *myhttp.Client // Reuse myhttp sebagai indirect dependency
}

// Option adalah fungsi modifier untuk konfigurasi Client saat inisialisasi.
type Option func(*clientConfig)

// clientConfig menyimpan opsi internal sebelum dibangun jadi myhttp.Client.
type clientConfig struct {
	retries       int
	retryWait     time.Duration
	rateLimit     int
	timeout       time.Duration
	userAgent     string
}

// WithRetry mengatur retry policy.
func WithRetry(retries int, wait time.Duration) Option {
	return func(c *clientConfig) {
		c.retries = retries
		c.retryWait = wait
	}
}

// WithRateLimit mengatur rate limit request per detik.
func WithRateLimit(limit int) Option {
	return func(c *clientConfig) {
		c.rateLimit = limit
	}
}

// WithTimeout mengatur timeout HTTP client.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithUserAgent mengatur custom User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *clientConfig) {
		c.userAgent = ua
	}
}

// New membuat instance Client baru dengan kredensial dan opsi transport.
// Jika apiKey/secretKey kosong, method Private akan gagal runtime (expected behavior).
func New(apiKey, secretKey string, opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		retries:   3,
		retryWait: 1 * time.Second,
		rateLimit: 5,
		timeout:   15 * time.Second,
		userAgent: "indodax-go/1.0",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Bangun myhttp.Client dengan opsi yang disamakan
	myOpts := []myhttp.Option{
		myhttp.WithRetry(cfg.retries, cfg.retryWait),
		myhttp.WithRateLimit(cfg.rateLimit),
		myhttp.WithTimeout(cfg.timeout),
		myhttp.WithUserAgent(cfg.userAgent),
	}

	httpClient := myhttp.New(myOpts...)

	if httpClient == nil {
		return nil, fmt.Errorf("gagal inisialisasi myhttp client")
	}

	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		http:      httpClient,
	}, nil
}

// doRequest adalah helper internal untuk mengeksekusi HTTP request via myhttp.
// Ini memastikan semua response body dibatasi (LimitReader) dan closed properly.
func (c *Client) doRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	req = req.WithContext(ctx)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transport error: %w", err)
	}
	defer resp.Body.Close()

	// Batas 2MB untuk mencegah memory exhaustion (DDoS protection)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// --- HELPER MARKET ORDER (Mencegah salah parameter Buy vs Sell) ---

// PlaceMarketBuy melakukan pembelian instan menggunakan nominal IDR (quoteOrderQty).
func (c *Client) PlaceMarketBuy(ctx context.Context, pair string, idrAmount float64) (OrderResult, error) {
	return c.SubmitOrder(ctx, SubmitOrderRequest{
		Symbol:        strings.ToUpper(pair),
		Side:          "BUY",
		QuoteOrderQty: strconv.FormatFloat(idrAmount, 'f', 2, 64),
	})
}

// PlaceMarketSell melakukan penjualan instan menggunakan jumlah koin (quantity).
func (c *Client) PlaceMarketSell(ctx context.Context, pair string, coinAmountRaw string) (OrderResult, error) {
	return c.SubmitOrder(ctx, SubmitOrderRequest{
		Symbol:   strings.ToUpper(pair),
		Side:     "SELL",
		Quantity: coinAmountRaw, // Langsung string mentah
	})
}
