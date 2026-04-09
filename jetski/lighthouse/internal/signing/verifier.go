package signing

import (
	"crypto/hmac"
)

func VerifySignature(chartData []byte, signature string, secretKey string) bool {
	expectedSig := SignChart(chartData, secretKey)
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
