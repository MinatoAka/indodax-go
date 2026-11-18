# indodax-go

[![Go Version](https://img.shields.io/badge/Go-1.27.3-00ADD8?logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/MinatoAka/indodax-go.svg)](https://pkg.go.dev/github.com/MinatoAka/indodax-go)
[![Go Tests](https://github.com/MinatoAka/indodax-go/actions/workflows/go.yml/badge.svg)](https://github.com/MinatoAka/indodax-go/actions/workflows/go.yml)
[![CodeQL](https://github.com/MinatoAka/indodax-go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/MinatoAka/indodax-go/actions/workflows/github-code-scanning/codeql)
![Release](https://img.shields.io/github/v/release/MinatoAka/indodax-go?include_prereleases&label=release)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Support via Saweria](https://img.shields.io/badge/Saweria-Support%20me-orange)](https://saweria.co/AKaAturDompet)

***Standalone Library Go untuk Indodax [Official Trade API v2](https://github.com/btcid/indodax-official-api-docs)***

Dirancang khusus untuk kebutuhan bot trading: presisi tinggi, modular, fokus pada eksekusi **Market Order**, serta pemisahan jelas antara endpoint Public & Private.

> ⚠️ **Catatan Penting:**  
> Proyek ini adalah implementasi pihak ketiga (*third-party*) yang tidak berafiliasi dengan Indodax.
> Pastikan Anda memahami dokumentasi resmi sebelum digunakan dalam produksi.

## ✨ Fitur Utama

- **Pemisahan Domain Fisik:** Base URL terpisah (`api.indodax.com` untuk private/signed, `indodax.com` untuk public/no-auth) untuk mencegah error routing.
- **Presisi Aset Maksimal:** Saldo aset dikembalikan sebagai string mentah (`AssetRaw`) tanpa konversi float *lossy* di layer SDK, sehingga mencegah error validasi presisi (`-1111`).
- **Fokus Market Order:** Mendukung penuh spesifikasi TAPI v2 untuk eksekusi instan (`quoteOrderQty` untuk `BUY` dan `quantity` untuk `SELL`).
- **Signing Otomatis & Deterministik:** Integrasi penuh dengan `hmacgo` untuk HMAC-SHA256 sesuai protokol resmi TAPI v2 (query string untuk GET, form-encoded body untuk POST).
- **Context Support Penuh:** Semua method menerima `context.Context` untuk manajemen timeout & cancellation.
- **Zero Coupling Config:** Tidak bergantung pada struct config bot tertentu. Cukup inisialisasi client sekali dengan kredensial.

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

    "[github.com/MinatoAka/indodax-go](https://github.com/MinatoAka/indodax-go)"
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
    fmt.Printf("Saldo IDR                  : Rp %.2f\n", balances.IDR)
    fmt.Printf("Saldo ETH (Mentah/String)  : %s\n", balances.AssetRaw)
    fmt.Printf("Saldo ETH (Float utk gate) : %.8f\n", balances.Asset)
}
```

### 4. Kirim Order BUY/SELL (Private - Signed POST)

Sesuai dokumentasi TAPI v2, `type` otomatis diset ke `MARKET`. Gunakan `QuoteOrderQty` (nominal IDR) untuk `BUY`, atau `Quantity` (jumlah aset) untuk `SELL`.  

- Market BUY (Nominal IDR)
  Gunakan helper `PlaceMarketBuy`. Ini mencegah salah input parameter (seperti tukar-balik Quantity vs QuoteOrderQty).

```go
ctx := context.Background()

// Beli ETH senilai Rp 100.000
buyRes, err := client.PlaceMarketBuy(ctx, "ETHIDR", 100000.00)
if err != nil {
    log.Printf("Error submit buy: %v", err)
} else {
    // Validasi sukses instan: OrderID > 0
    if buyRes.OrderID > 0 {
        fmt.Printf("[✓] BUY Success | ID: %d | Qty Filled: %s\n", 
            buyRes.OrderID, buyRes.ExecutedQty)
    }
}
```

- Market SELL (Jumlah Koin)
  Gunakan helper `PlaceMarketSell`. Pastikan menggunakan saldo riil dari API `Balances()` untuk menghindari rejection karena floating point error.

```go
// Asumsi sudah panggil balances sebelumnya
balances, _ := client.Balances(ctx, "ETHIDR")

// Jual seluruh saldo ETH yang dimiliki
sellRes, err := client.PlaceMarketSell(ctx, "ETHIDR", balances.Asset)
if err != nil {
    log.Printf("Error submit sell: %v", err)
} else {
    if sellRes.OrderID > 0 {
        fmt.Printf("[✓] SELL Success | ID: %d | Qty Sold: %s\n", 
            sellRes.OrderID, sellRes.ExecutedQty)
    }
}
```

## 🛠️ Arsitektur & Design Choices

| Komponen | Implementasi | Alasan Teknis |
| :--- | :--- | :--- |
| **Base URLs** | Global vars di `private.go` & `public.go` | Memudahkan override untuk testing/mock server tanpa refactor besar. |
| **Signing Payload** | Sorted Query String `GET` / Form Body `POST` | Sesuai protokol ketat Indodax TAPI v2. Salah satu saja = reject silent. |
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

## 📦 Data Types (Response)

SDK ini mengembalikan data mentah dari API dengan presisi tinggi. Berikut adalah struct utama yang akan Anda terima:

### 1. `OrderResult` (`BUY` & `SELL`)
Menyimpan detail transaksi. Angka desimal disimpan sebagai `string` untuk mencegah *floating-point loss*.

| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `Symbol` | `string` | Trading pair symbol. |
| `OrderID` | `int64` | Unique identifier for the order. |
| `Side` | `string` | Order side (`BUY` or `SELL`). |
| `Type` | `string` | Order type (`MARKET`). |
| `OrigQty` | `string` | Original order quantity. |
| `ExecutedQty` | `string` | Quantity that had already been executed before cancellation. |
| `FullOrderID` | `string` | System-generated order reference (`ethidr-market-6423`) |

### 2. `BalanceResult`
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `IDR` | `float64` | Saldo IDR |
| `Asset` | `float64` | Saldo aset/coin |
| `AssetRaw` | `string` | Saldo aset **mentah** dari API. |

### 3. `TickerResponse` (dari `Ticker`)
Data pasar *real-time*. Menggunakan `json.Number` untuk mencegah *error parsing* tipe data.
> Akses nilainya menggunakan *helper methods*:
* `.GetLastPrice()`,
* `.GetHighestBid()`,
* `.GetLowestAsk()`,
* `.GetDayHigh()`,
* `.GetDayLow()`.

### 4. `PairInfo`
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `ID` | `string` | Nama pair (contoh: `ETHIDR`). |
| `MinQty` | `float64` | Batas minimum kuantitas yang boleh ditransaksikan. |


## 🤝 Support & Exchange

Silakan mendaftar melalui tautan resmi berikut:

👉 **[Daftar Akun Indodax](https://indodax.com/deeplink?page=register&id=AkaKen)**

Jika library ini membantu project bot trading anda, bisa juga **[Traktir ☕](https://saweria.co/AKaAturDompet)**.

*(Tautan di atas adalah link referral standar untuk pendaftaran akun baru di platform Indodax.)*
