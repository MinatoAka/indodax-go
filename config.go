package indodax

var (
	// BaseURLPrivate adalah domain untuk endpoint signed (/api/v2/*).
	// Wajib pakai header X-APIKEY & Sign.
	BaseURLPrivate = "https://api.indodax.com"

	// BaseURLPublic adalah domain untuk endpoint tanpa auth (/api/*).
	// Tidak memerlukan kredensial.
	BaseURLPublic = "https://indodax.com"
)
