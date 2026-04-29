// Package setup provides SSL/TLS certificate generation utilities.
// This generates self-signed certificates for local development and testing.
package setup

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultCertDir is the default directory for SSL certificates
	DefaultCertDir = "/etc/armorclaw/ssl"

	// DefaultCertExpiry is the default certificate validity period (1 year)
	DefaultCertExpiry = 365 * 24 * time.Hour

	// DefaultKeySize is the ECDSA curve used (P-256)
)

// SSLConfig configures SSL certificate generation
type SSLConfig struct {
	// ServerName is the CN and SAN for the certificate
	ServerName string

	// OutputDir is where certificates will be written
	OutputDir string

	// Expiry is how long the certificate is valid
	Expiry time.Duration

	// Organization is the certificate organization
	Organization string

	// AdditionalSANs are additional Subject Alternative Names
	AdditionalSANs []string

	// PublicIP is an external/public IP to include in certificate SANs
	PublicIP string
}

// SSLResult contains the generated certificate paths
type SSLResult struct {
	CertPath string
	KeyPath  string
	CertPEM  []byte
	KeyPEM   []byte
}

// GenerateSelfSignedCert generates a self-signed ECDSA certificate.
// This is suitable for local development and testing only.
func GenerateSelfSignedCert(cfg SSLConfig) (*SSLResult, error) {
	if cfg.ServerName == "" {
		return nil, &SetupError{
			Code:  "INS-014",
			Title: "Server name required for SSL certificate",
			Cause: "Cannot generate SSL certificate without a server name.",
			Fix: []string{
				"Provide a server name in the configuration",
				"Or set ARMORCLAW_SERVER_NAME environment variable",
			},
			ExitCode: 1,
		}
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = DefaultCertDir
	}

	if cfg.Expiry == 0 {
		cfg.Expiry = DefaultCertExpiry
	}

	if cfg.Organization == "" {
		cfg.Organization = "ArmorClaw"
	}

	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		return nil, WrapError(err, ErrSSLCertFailed)
	}

	// Generate serial number
	serialNumber, err := cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, WrapError(err, ErrSSLCertFailed)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(cfg.Expiry)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{cfg.Organization},
			CommonName:   cfg.ServerName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              append([]string{cfg.ServerName}, cfg.AdditionalSANs...),
	}

	// Allow localhost and IP addresses for development
	template.DNSNames = append(template.DNSNames, "localhost")
	template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}

	if cfg.PublicIP != "" {
		if ip := net.ParseIP(cfg.PublicIP); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// Generate certificate
	certDER, err := x509.CreateCertificate(cryptorand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, WrapError(err, ErrSSLCertFailed)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, WrapError(err, ErrSSLCertFailed)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	result := &SSLResult{
		CertPath: filepath.Join(cfg.OutputDir, "server.crt"),
		KeyPath:  filepath.Join(cfg.OutputDir, "server.key"),
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
	}

	return result, nil
}

// WriteCertificate writes the certificate and key to disk
func (r *SSLResult) WriteCertificate() error {
	// Create output directory if needed
	outputDir := filepath.Dir(r.CertPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return &SetupError{
			Code:  "INS-003",
			Title: "Cannot create SSL directory",
			Cause: fmt.Sprintf("Failed to create directory %s: %v", outputDir, err),
			Fix: []string{
				"Check parent directory permissions",
				"Run with appropriate privileges",
			},
			ExitCode: 1,
		}
	}

	// Write certificate
	if err := os.WriteFile(r.CertPath, r.CertPEM, 0644); err != nil {
		return &SetupError{
			Code:  "INS-014",
			Title: "Cannot write SSL certificate",
			Cause: fmt.Sprintf("Failed to write certificate to %s: %v", r.CertPath, err),
			Fix: []string{
				"Check directory permissions",
				"Ensure sufficient disk space",
			},
			ExitCode: 1,
		}
	}

	// Write private key with restricted permissions
	if err := os.WriteFile(r.KeyPath, r.KeyPEM, 0600); err != nil {
		return &SetupError{
			Code:  "INS-014",
			Title: "Cannot write SSL private key",
			Cause: fmt.Sprintf("Failed to write key to %s: %v", r.KeyPath, err),
			Fix: []string{
				"Check directory permissions",
				"Ensure sufficient disk space",
			},
			ExitCode: 1,
		}
	}

	return nil
}

// GenerateAndWriteCert is a convenience function that generates and writes a certificate
func GenerateAndWriteCert(cfg SSLConfig) (*SSLResult, error) {
	result, err := GenerateSelfSignedCert(cfg)
	if err != nil {
		return nil, err
	}

	if err := result.WriteCertificate(); err != nil {
		return nil, err
	}

	return result, nil
}

// CertificateExists checks if certificate files already exist
func CertificateExists(certDir string) bool {
	certPath := filepath.Join(certDir, "server.crt")
	keyPath := filepath.Join(certDir, "server.key")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// ValidateCertificate checks if the existing certificate is valid
func ValidateCertificate(certPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return WrapError(err, ErrSSLCertFailed)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return &SetupError{
			Code:  "INS-014",
			Title: "Invalid certificate format",
			Cause: "The certificate file is not in valid PEM format.",
			Fix: []string{
				"Regenerate the certificate",
				"Ensure the certificate file was not corrupted",
			},
			ExitCode: 1,
		}
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return WrapError(err, ErrSSLCertFailed)
	}

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return &SetupError{
			Code:  "INS-014",
			Title: "Certificate not yet valid",
			Cause: "The certificate's NotBefore date is in the future.",
			Fix: []string{
				"Check system clock synchronization",
				"Regenerate the certificate",
			},
			ExitCode: 1,
		}
	}

	if now.After(cert.NotAfter) {
		return &SetupError{
			Code:  "INS-014",
			Title: "Certificate has expired",
			Cause: fmt.Sprintf("Certificate expired on %s", cert.NotAfter.Format(time.RFC3339)),
			Fix: []string{
				"Regenerate the certificate",
				"Use a longer expiry period",
			},
			ExitCode: 1,
		}
	}

	// Warn if expiring soon (within 30 days)
	if time.Until(cert.NotAfter) < 30*24*time.Hour {
		// This is just a warning, not an error
		// In a real implementation, we'd log this
	}

	return nil
}
