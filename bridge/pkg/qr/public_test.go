package qr

import (
	"strings"
	"testing"
)

func TestToTerminal(t *testing.T) {
	qrResult := &QRResult{DeepLink: "armorclaw://config?d=testdata"}
	result, err := qrResult.ToTerminal()
	if err != nil {
		t.Fatalf("ToTerminal() failed: %v", err)
	}
	if result == "" {
		t.Fatal("ToTerminal() returned empty string")
	}
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "█") && !strings.Contains(line, " ") && !strings.Contains(line, "▄") && !strings.Contains(line, "▀") {
			t.Errorf("ToTerminal() output doesn't look like QR code (no block characters): %s", line)
		}
	}
}

func TestToTerminal_EmptyDeepLink(t *testing.T) {
	qrResult := &QRResult{DeepLink: ""}
	_, err := qrResult.ToTerminal()
	if err == nil {
		t.Fatal("ToTerminal() should fail with empty deep link")
	}
	if !strings.Contains(err.Error(), "deep link is empty") {
		t.Errorf("Expected error to contain 'deep link is empty', got: %v", err)
	}
}

func TestToTerminal_WhitespaceDeepLink(t *testing.T) {
	qrResult := &QRResult{DeepLink: "   \t\n  "}
	_, err := qrResult.ToTerminal()
	if err == nil {
		t.Fatal("ToTerminal() should fail with whitespace-only deep link")
	}
	if !strings.Contains(err.Error(), "deep link is empty") {
		t.Errorf("Expected error to contain 'deep link is empty', got: %v", err)
	}
}
