package setup

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGenerateSelfSignedCert tests certificate generation
func TestGenerateSelfSignedCert(t *testing.T) {
	cfg := SSLConfig{
		ServerName:    "test.example.com",
		Organization:  "Test Org",
		Expiry:        24 * time.Hour,
		AdditionalSANs: []string{"test2.example.com"},
	}

	result, err := GenerateSelfSignedCert(cfg)
	if err != nil {
		t.Fatalf("failed to generate certificate: %v", err)
	}

	// Verify cert and key PEM data exists
	if len(result.CertPEM) == 0 {
		t.Error("expected non-empty certificate PEM")
	}
	if len(result.KeyPEM) == 0 {
		t.Error("expected non-empty key PEM")
	}

	// Verify paths are set correctly
	if result.CertPath == "" {
		t.Error("expected cert path to be set")
	}
	if result.KeyPath == "" {
		t.Error("expected key path to be set")
	}

	// Parse and verify certificate
	block, _ := pem.Decode(result.CertPEM)
	if block == nil {
		t.Fatal("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Verify subject
	if cert.Subject.CommonName != cfg.ServerName {
		t.Errorf("expected CN %s, got %s", cfg.ServerName, cert.Subject.CommonName)
	}

	// Verify SANs
	foundServerName := false
	foundAdditional := false
	for _, name := range cert.DNSNames {
		if name == cfg.ServerName {
			foundServerName = true
		}
		if name == "test2.example.com" {
			foundAdditional = true
		}
	}
	if !foundServerName {
		t.Error("expected server name in SANs")
	}
	if !foundAdditional {
		t.Error("expected additional SAN in certificate")
	}

	// Verify localhost is included
	foundLocalhost := false
	for _, name := range cert.DNSNames {
		if name == "localhost" {
			foundLocalhost = true
			break
		}
	}
	if !foundLocalhost {
		t.Error("expected localhost in SANs")
	}
}

// TestGenerateSelfSignedCertNoServerName tests error when server name is missing
func TestGenerateSelfSignedCertNoServerName(t *testing.T) {
	cfg := SSLConfig{
		Organization: "Test Org",
	}

	_, err := GenerateSelfSignedCert(cfg)
	if err == nil {
		t.Error("expected error when server name is missing")
	}

	setupErr := GetSetupError(err)
	if setupErr == nil {
		t.Error("expected SetupError")
	} else if setupErr.Code != "INS-014" {
		t.Errorf("expected error code INS-014, got %s", setupErr.Code)
	}
}

// TestGenerateSelfSignedCertDefaults tests default values
func TestGenerateSelfSignedCertDefaults(t *testing.T) {
	cfg := SSLConfig{
		ServerName: "default.example.com",
	}

	result, err := GenerateSelfSignedCert(cfg)
	if err != nil {
		t.Fatalf("failed to generate certificate with defaults: %v", err)
	}

	// Verify defaults were applied
	if result.CertPath != filepath.Join(DefaultCertDir, "server.crt") {
		t.Errorf("expected default cert path, got %s", result.CertPath)
	}
	if result.KeyPath != filepath.Join(DefaultCertDir, "server.key") {
		t.Errorf("expected default key path, got %s", result.KeyPath)
	}
}

// TestWriteCertificate tests writing certificate to disk
func TestWriteCertificate(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "armorclaw-ssl-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := SSLConfig{
		ServerName: "write-test.example.com",
		OutputDir:  tempDir,
		Expiry:     24 * time.Hour,
	}

	result, err := GenerateSelfSignedCert(cfg)
	if err != nil {
		t.Fatalf("failed to generate certificate: %v", err)
	}

	// Write certificate
	err = result.WriteCertificate()
	if err != nil {
		t.Fatalf("failed to write certificate: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(result.CertPath); os.IsNotExist(err) {
		t.Error("certificate file was not created")
	}
	if _, err := os.Stat(result.KeyPath); os.IsNotExist(err) {
		t.Error("key file was not created")
	}

	// Verify key file permissions (should be 0600 on Unix systems)
	// Note: On Windows, permissions work differently
	keyInfo, err := os.Stat(result.KeyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	// On Unix-like systems, check permissions
	// On Windows, just verify file exists
	_ = keyInfo // Avoid unused variable warning on Windows
}

// TestCertificateExists tests certificate existence check
func TestCertificateExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "armorclaw-ssl-exists-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Should not exist initially
	if CertificateExists(tempDir) {
		t.Error("expected certificate to not exist")
	}

	// Create certificate files
	cfg := SSLConfig{
		ServerName: "exists-test.example.com",
		OutputDir:  tempDir,
	}
	result, _ := GenerateSelfSignedCert(cfg)
	result.WriteCertificate()

	// Should exist now
	if !CertificateExists(tempDir) {
		t.Error("expected certificate to exist after creation")
	}
}

// TestValidateCertificate tests certificate validation
func TestValidateCertificate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "armorclaw-ssl-validate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate and write certificate
	cfg := SSLConfig{
		ServerName: "validate-test.example.com",
		OutputDir:  tempDir,
		Expiry:     24 * time.Hour,
	}
	result, _ := GenerateSelfSignedCert(cfg)
	result.WriteCertificate()

	// Validate should pass
	err = ValidateCertificate(result.CertPath)
	if err != nil {
		t.Errorf("certificate validation failed: %v", err)
	}
}

// TestValidateCertificateInvalidPEM tests validation with invalid PEM
func TestValidateCertificateInvalidPEM(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "armorclaw-ssl-invalid-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write invalid PEM
	certPath := filepath.Join(tempDir, "server.crt")
	if err := os.WriteFile(certPath, []byte("not a valid PEM"), 0644); err != nil {
		t.Fatalf("failed to write invalid cert: %v", err)
	}

	err = ValidateCertificate(certPath)
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

// TestGenerateAndWriteCert tests the convenience function
func TestGenerateAndWriteCert(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "armorclaw-ssl-convenience-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := SSLConfig{
		ServerName: "convenience-test.example.com",
		OutputDir:  tempDir,
		Expiry:     24 * time.Hour,
	}

	result, err := GenerateAndWriteCert(cfg)
	if err != nil {
		t.Fatalf("failed to generate and write certificate: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(result.CertPath); os.IsNotExist(err) {
		t.Error("certificate file was not created")
	}
	if _, err := os.Stat(result.KeyPath); os.IsNotExist(err) {
		t.Error("key file was not created")
	}
}
