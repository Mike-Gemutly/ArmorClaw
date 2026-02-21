package securerandom

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestID(t *testing.T) {
	id, err := ID(16)
	if err != nil {
		t.Fatalf("ID() returned error: %v", err)
	}

	// ID should be hex-encoded (32 characters for 16 bytes)
	if len(id) != 32 {
		t.Errorf("ID(16) returned wrong length: got %d, want 32", len(id))
	}

	// Verify it's valid hex
	_, err = hex.DecodeString(id)
	if err != nil {
		t.Errorf("ID() returned invalid hex: %v", err)
	}
}

func TestMustID(t *testing.T) {
	id := MustID(16)
	if len(id) != 32 {
		t.Errorf("MustID(16) returned wrong length: got %d, want 32", len(id))
	}
}

func TestIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := MustID(16)
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestBytes(t *testing.T) {
	b, err := Bytes(32)
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if len(b) != 32 {
		t.Errorf("Bytes(32) returned wrong length: got %d, want 32", len(b))
	}
}

func TestMustBytes(t *testing.T) {
	b := MustBytes(32)
	if len(b) != 32 {
		t.Errorf("MustBytes(32) returned wrong length: got %d, want 32", len(b))
	}
}

func TestBytesUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		b := MustBytes(16)
		key := hex.EncodeToString(b)
		if seen[key] {
			t.Errorf("Duplicate bytes generated: %s", key)
		}
		seen[key] = true
	}
}

func TestToken(t *testing.T) {
	token, err := Token(24)
	if err != nil {
		t.Fatalf("Token() returned error: %v", err)
	}

	// Token should be URL-safe base64 encoded
	if strings.ContainsAny(token, "+/") {
		t.Errorf("Token() contains non-URL-safe characters: %s", token)
	}
}

func TestMustToken(t *testing.T) {
	token := MustToken(24)
	if len(token) == 0 {
		t.Error("MustToken() returned empty string")
	}
}

func TestTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		token := MustToken(16)
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestFill(t *testing.T) {
	b := make([]byte, 32)
	err := Fill(b)
	if err != nil {
		t.Fatalf("Fill() returned error: %v", err)
	}

	// Verify the slice is filled (not all zeros)
	allZeros := true
	for _, v := range b {
		if v != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Fill() left buffer as all zeros (extremely unlikely)")
	}
}

func TestMustFill(t *testing.T) {
	b := make([]byte, 32)
	MustFill(b)

	// Verify the slice is filled (not all zeros)
	allZeros := true
	for _, v := range b {
		if v != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("MustFill() left buffer as all zeros (extremely unlikely)")
	}
}

func TestChallenge(t *testing.T) {
	challenge, err := Challenge()
	if err != nil {
		t.Fatalf("Challenge() returned error: %v", err)
	}

	if len(challenge) == 0 {
		t.Error("Challenge() returned empty string")
	}
}

func TestMustChallenge(t *testing.T) {
	challenge := MustChallenge()
	if len(challenge) == 0 {
		t.Error("MustChallenge() returned empty string")
	}
}

func TestNonce(t *testing.T) {
	nonce, err := Nonce(12)
	if err != nil {
		t.Fatalf("Nonce() returned error: %v", err)
	}

	if len(nonce) != 12 {
		t.Errorf("Nonce(12) returned wrong length: got %d, want 12", len(nonce))
	}
}

func TestMustNonce(t *testing.T) {
	nonce := MustNonce(12)
	if len(nonce) != 12 {
		t.Errorf("MustNonce(12) returned wrong length: got %d, want 12", len(nonce))
	}
}
