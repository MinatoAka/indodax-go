# indodax-go

[![Go Version](https://img.shields.io/badge/Go-1.27.3-00ADD8?logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/MinatoAka/indodax-go.svg)](https://pkg.go.dev/github.com/MinatoAka/indodax-go)
[![Go Tests](https://github.com/MinatoAka/indodax-go/actions/workflows/go.yml/badge.svg)](https://github.com/MinatoAka/indodax-go/actions/workflows/go.yml)
[![CodeQL](https://github.com/MinatoAka/indodax-go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/MinatoAka/indodax-go/actions/workflows/github-code-scanning/codeql)
![Release](https://img.shields.io/github/v/release/MinatoAka/indodax-go?include_prereleases&label=release)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Support via Saweria](https://img.shields.io/badge/Saweria-Support%20me-orange)](https://saweria.co/AKaAturDompet)

***Standalone Library Go untuk Indodax [Official Trade API v2](https://github.com/btcid/indodax-official-api-docs)***[span_3](start_span)[span_3](end_span)

Dirancang khusus untuk kebutuhan bot trading: presisi tinggi, modular, fokus pada eksekusi **Market Order**, serta pemisahan jelas antara endpoint Public & Private[span_4](start_span)[span_4](end_span)[span_5](start_span)[span_5](end_span).

> ⚠️ **Catatan Penting:**  
> Proyek ini adalah implementasi pihak ketiga (*third-party*) yang tidak berafiliasi dengan Indodax[span_6](start_span)[span_6](end_span).  
> Pastikan Anda memahami dokumentasi resmi sebelum digunakan dalam produksi[span_7](start_span)[span_7](end_span).

## ✨ Fitur Utama

- **Pemisahan Domain Fisik:** Base URL terpisah (`api.indodax.com` untuk private/signed, `indodax.com` untuk public/no-auth) untuk mencegah error routing[span_8](start_span)[span_8](end_span)[span_9](start_span)[span_9](end_span)[span_10](start_span)[span_10](end_span).
- **Presisi Aset Maksimal:** Saldo aset dikembalikan sebagai string mentah (`AssetRaw`) tanpa konversi float *lossy* di layer SDK, sehingga mencegah error validasi presisi (`-1111`)[span_11](start_span)[span_11](end_span)[span_12](start_span)[span_12](end_span)[span_13](start_span)[span_13](end_span)[span_14](start_span)[span_14](end_span).
- **Fokus Market Order:** Mendukung penuh spesifikasi TAPI v2 untuk eksekusi instan (`quoteOrderQty` untuk `BUY` dan `quantity` untuk `SELL`)[span_15](start_span)[span_15](end_span)[span_16](start_span)[span_16](end_span)[span_17](start_span)[span_17](end_span).
- **Signing Otomatis & Deterministik:** Integrasi penuh dengan `hmacgo` untuk HMAC-SHA256 sesuai protokol resmi TAPI v2 (query string untuk GET, form-encoded body untuk POST)[span_18](start_span)[span_18](end_span)[span_19](start_span)[span_19](end_span)[span_20](start_span)[span_20](end_span).
- **Context Support Penuh:** Semua method menerima `context.Context` untuk manajemen timeout & cancellation[span_21](start_span)[span_21](end_span)[span_22](start_span)[span_22](end_span)[span_23](start_span)[span_23](end_span)[span_24](start_span)[span_24](end_span).
- **Zero Coupling Config:** Tidak bergantung pada struct config bot tertentu[span_25](start_span)[span_25](end_span). Cukup inisialisasi client sekali dengan kredensial[span_26](start_span)[span_26](end_span)[span_27](start_span)[span_27](end_span).

## 🚀 Instalasi

```bash
go get [github.com/MinatoAka/indodax-go@main](https://github.com/MinatoAka/indodax-go@main)
```

## ⚡ Quick Start

### 1. Inisialisasi Client

Simpan kredensial di environment variable atau loader config. Jangan hardcode!

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/MinatoAka/indodax-go"
)

func main() {
    apiKey := os.Getenv("INDODAX_API_KEY")
    secretKey := os.Getenv("INDODAX_SECRET_KEY")

    if apiKey == "" || secretKey == "" {
        log.Fatal("Set INDODAX_API_KEY & INDODAX_SECRET_KEY dulu!")
    }

    // Opsi default sudah optimal (retry 3x, rate limit 5 req/s, timeout 15s)
    client, err := indodax.New(apiKey, secretKey)
    if err != nil {
        log.Fatalf("Gagal bikin client: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // ... lanjut ke contoh usage di bawah
}
```

### 2. Ambil Data Live Market Lengkap (Public - No Auth)

Menggunakan struct `TickerResponse` dengan tipe `json.Number` untuk menjaga presisi angka dari JSON. Gunakan helper method `.Float64()` untuk akses nilai desimal.

```go
ticker, err := client.Ticker(ctx, "ethidr")
if err != nil {
    log.Printf("Error ticker: %v", err)
} else {
    // Akses data via Helper Method (menghandle konversi json.Number ke float64 secara aman)
    lastPrice, _ := ticker.GetLastPrice()
    bestBid, _   := ticker.GetHighestBid()
    bestAsk, _   := ticker.GetLowestAsk()
    
    fmt.Printf("Harga Terakhir : Rp %.2f\n", lastPrice)
    fmt.Printf("Best Bid       : Rp %.2f\n", bestBid)
    fmt.Printf("Best Ask       : Rp %.2f\n", bestAsk)
}
```

### 3. Cek Saldo (Private - Signed GET)

> **⚠️ CATATAN KRUSIAL:**
> Gunakan field `AssetRaw` (string) untuk mengirim order SELL.
> Jangan pernah konversi ke float sebelum kirim, karena bisa kehilangan presisi digit terakhir yang menyebabkan rejection `-1111 Quantity validation failed`.

```go
balances, err := client.Balances(ctx, "ethidr")
if err != nil {
    log.Printf("Error balances: %v", err)
} else {
    fmt.Printf("Saldo IDR: Rp %.2f\n", balances.IDR)
    fmt.Printf("Saldo ETH (Mentah/String): %s\n", balances.AssetRaw)
    fmt.Printf("Saldo ETH (Float utk gate MinQty): %.8f\n", balances.Asset)
}
```

### 4. Kirim Order SELL Market (Private - Signed POST)

Payload signing adalah BODY string, bukan query string. URL endpoint bersih tanpa `?`.

```go
orderReq := indodax.SubmitOrderRequest{
    Symbol:   "ETHIDR",           // Uppercase sesuai doc
    Side:     "SELL",             // BUY or SELL
    Type:     "MARKET",           // LIMIT or MARKET
    Quantity: balances.AssetRaw,  // ← WAJIB pakai string mentah dari Balances()
}

result, err := client.SubmitOrder(ctx, orderReq)
if err != nil {
    log.Printf("Error submit order: %v", err)
} else {
    fmt.Printf("Order Placed! ID: %d, Executed Qty: %.8f\n", result.OrderID, result.ExecutedQty)
}
```

### 5. Cek Status Order (Private - Signed GET)

Doc resmi mewajibkan parameter `symbol` selain `orderId`. Pastikan keduanya dikirim.

```go
// Note: Signature method OrderStatus membutuhkan pair sebagai argumen pertama
status, err := client.OrderStatus(ctx, "ETHIDR", result.OrderID)
if err != nil {
    log.Printf("Error order status: %v", err)
} else {
    fmt.Printf("Status Order: Filled? Qty=%.8f, Avg Price=%.2f\n", status.ExecutedQty, status.Price)
}
```

## 🛠️ Arsitektur & Design Choices

| Komponen | Implementasi | Alasan Teknis |
| :--- | :--- | :--- |
| **Base URLs** | Global vars di `config.go` | Memudahkan override untuk testing/mock server tanpa refactor besar. |
| **Signing Payload** | Sorted Query String (GET) / Form Body (POST) | Sesuai protokol ketat Indodax TAPI v2. Salah satu saja = reject silent. |
| **Response Parsing** | Strict JSON Unmarshal + LimitReader (2MB) | Mencegah memory exhaustion dari response abnormal/DDoS ringan. |
| **Error Handling** | Wrapped errors (`%w`) | Mempertahankan stack trace debugging sampai ke akar masalah transport/logic. |

## 🧪 Testing

Unit tests menggunakan `httptest.Server` lokal (mock server). Tidak butuh koneksi internet asli. Validasi meliputi:
1. Logika Signature (GET vs POST).
2. Presisi String Aset (`AssetRaw`).
3. Separation of Concerns (Public tidak bocor header auth).

```bash
go test -v ./...
```

## 🤝 Support & Exchange

Silakan mendaftar melalui tautan resmi berikut:

👉 **[Daftar Akun Indodax](https://indodax.com/deeplink?page=register&id=AkaKen)**

Jika library ini membantu project bot trading anda, bisa juga **[Traktir ☕](https://saweria.co/AKaAturDompet)**.

*(Tautan di atas adalah link referral standar untuk pendaftaran akun baru di platform Indodax.)*
