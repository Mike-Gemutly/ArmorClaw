package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func SignChart(chartData []byte, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(chartData)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
