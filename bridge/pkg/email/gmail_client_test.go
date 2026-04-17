package email

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncodeBase64URL_EmptyString(t *testing.T) {
	result := encodeBase64URL("")
	if result != "" {
		t.Errorf("encodeBase64URL('') = %q, want empty", result)
	}
}

func TestEncodeBase64URL_HelloWorld(t *testing.T) {
	stdResult := base64.RawStdEncoding.EncodeToString([]byte("Hello World"))
	result := encodeBase64URL("Hello World")
	if result != stdResult {
		t.Errorf("encodeBase64URL('Hello World') = %q, want %q", result, stdResult)
	}
}

func TestEncodeBase64URL_URLSafeChars(t *testing.T) {
	result := encodeBase64URL("\xfb\xff\xfe")
	if strings.Contains(result, "+") || strings.Contains(result, "/") {
		t.Errorf("result should use URL-safe chars, got %q", result)
	}
	urlResult := base64.RawURLEncoding.EncodeToString([]byte("\xfb\xff\xfe"))
	if result != urlResult {
		t.Errorf("encodeBase64URL = %q, want %q", result, urlResult)
	}
}

func TestEncodeBase64URL_SingleByte(t *testing.T) {
	input := "A"
	result := encodeBase64URL(input)
	expected := base64.RawURLEncoding.EncodeToString([]byte(input))
	if result != expected {
		t.Errorf("single byte: got %q, want %q", result, expected)
	}
}

func TestEncodeBase64URL_TwoBytes(t *testing.T) {
	input := "AB"
	result := encodeBase64URL(input)
	expected := base64.RawURLEncoding.EncodeToString([]byte(input))
	if result != expected {
		t.Errorf("two bytes: got %q, want %q", result, expected)
	}
}

func TestGmailClient_Provider(t *testing.T) {
	client := &GmailClient{}
	if client.Provider() != "gmail" {
		t.Errorf("Provider() = %q, want gmail", client.Provider())
	}
}

func TestGmailClient_ImplementsEmailSender(t *testing.T) {
	var _ EmailSender = (*GmailClient)(nil)
}

func TestEncodeByte_TableBounds(t *testing.T) {
	encodeByte(0)
	encodeByte(63)
}

func TestEncodeBase64URL_MultiChunk(t *testing.T) {
	input := strings.Repeat("x", 100)
	result := encodeBase64URL(input)
	expected := base64.RawURLEncoding.EncodeToString([]byte(input))
	if result != expected {
		t.Errorf("100-byte input mismatch")
	}
}

func TestEncodeBase64URL_BinaryData(t *testing.T) {
	input := []byte{0x00, 0x01, 0x02, 0xFE, 0xFF}
	result := encodeBase64URL(string(input))
	expected := base64.RawURLEncoding.EncodeToString(input)
	if result != expected {
		t.Errorf("binary data: got %q, want %q", result, expected)
	}
}
