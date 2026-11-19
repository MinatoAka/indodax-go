package indodax

import (
	"encoding/json"
	"fmt"
)

// APIError merepresentasikan format error standar dari Indodax API v2 dengan tipe yang jelas
type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("indodax api error (code %d): %s", e.Code, e.Msg)
}

// Helper untuk identifikasi cepat di bot trading (switch-case friendly)
func (e *APIError) IsRateLimit() bool           { return e.Code == -1003 }
func (e *APIError) IsInvalidTimestamp() bool     { return e.Code == -1021 }
func (e *APIError) IsInvalidSignature() bool     { return e.Code == -1022 || e.Code == -2014 }
func (e *APIError) IsInsufficientBalance() bool  { return e.Code == -4026 }
func (e *APIError) IsOrderNotFound() bool        { return e.Code == -2013 || e.Code == 1112 }
func (e *APIError) IsQuantityValidationFailed() bool { return e.Code == -1111 }

// checkAPIErrorTyped mengupgrade checkAPIError agar return *APIError
func checkAPIErrorTyped(body []byte) error {
	var errResp APIError
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != 0 {
		return &errResp
	}
	return nil
}
