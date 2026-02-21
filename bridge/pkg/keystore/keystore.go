// Package keystore provides encrypted credential storage using SQLCipher.
// The keystore uses hardware-derived master keys for zero-knowledge persistence.
//
// Zero-Touch Reboot Strategy:
// - Entropy collected from machine-specific markers (UUID, MAC, salt)
// - Key derived via PBKDF2-HMAC-SHA512 with persisted salt
// - No password required on reboot
// - Database useless if stolen/moved to different server
package keystore

import (
	"bufio"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"database/sql"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	_ "github.com/mutecomm/go-sqlcipher/v4"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Key derivation parameters
	saltLength       = 32
	pbkdf2Iterations = 256000 // SQLCipher default
	keyLength        = 32

	// SQLCipher parameters
	cipherPageSize      = 4096
	cipherKdfIter       = 256000
	cipherHmacAlg       = "HMAC_SHA512"
	cipherKdfAlgorithm  = "PBKDF2_HMAC_SHA512"
)

var (
	ErrKeyNotFound       = errors.New("key not found")
	ErrInvalidProvider   = errors.New("invalid provider")
	ErrKeyExpired        = errors.New("key has expired")
	ErrDatabaseLocked    = errors.New("database is locked")
	ErrInvalidCredential = errors.New("invalid credential format")
)

// Provider represents an AI service provider
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
	ProviderGoogle     Provider = "google"
	ProviderXAI        Provider = "xai"
)

// Credential represents an encrypted API credential
type Credential struct {
	ID          string   `json:"id"`
	Provider    Provider `json:"provider"`
	Token       string   `json:"token"`        // Encrypted
	DisplayName string   `json:"display_name"`
	CreatedAt   int64    `json:"created_at"`
	ExpiresAt   int64    `json:"expires_at,omitempty"` // Unix timestamp
	Tags        []string `json:"tags,omitempty"`
}

// KeyInfo is the public information about a stored key
type KeyInfo struct {
	ID          string   `json:"id"`
	Provider    Provider `json:"provider"`
	DisplayName string   `json:"display_name"`
	CreatedAt   int64    `json:"created_at"`
	ExpiresAt   int64    `json:"expires_at,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Keystore manages encrypted credential storage
type Keystore struct {
	db          *sql.DB
	dbPath      string
	mu          sync.RWMutex
	masterKey   []byte
	salt        []byte
	isOpen      bool
	auditLogger *audit.CriticalOperationLogger
}

// Config holds keystore configuration
type Config struct {
	DBPath    string // Path to the SQLite database file
	MasterKey []byte // Optional master key (if nil, will derive from hardware)
}

// New creates a new Keystore instance
func New(cfg Config) (*Keystore, error) {
	if cfg.DBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.DBPath = filepath.Join(homeDir, ".armorclaw", "keystore.db")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %w", err)
	}

	ks := &Keystore{
		dbPath: cfg.DBPath,
	}

	// Load or generate salt (persists across reboots)
	if err := ks.loadOrGenerateSalt(); err != nil {
		return nil, fmt.Errorf("failed to initialize salt: %w", err)
	}

	// Derive master key from hardware entropy + salt
	if cfg.MasterKey == nil {
		var err error
		cfg.MasterKey, err = ks.deriveHardwareKey()
		if err != nil {
			return nil, fmt.Errorf("failed to derive hardware key: %w", err)
		}
	}

	ks.masterKey = cfg.MasterKey

	return ks, nil
}

// loadOrGenerateSalt loads an existing salt or generates a new one
// The salt persists across reboots to enable zero-touch operation
func (ks *Keystore) loadOrGenerateSalt() error {
	saltPath := ks.dbPath + ".salt"

	// Try to load existing salt
	if data, err := os.ReadFile(saltPath); err == nil {
		ks.salt, err = base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(ks.salt) == saltLength {
			return nil
		}
	}

	// Generate new salt (this happens once on first install)
	ks.salt = make([]byte, saltLength)
	if _, err := io.ReadFull(cryptorand.Reader, ks.salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Save salt for future reboots
	if err := os.WriteFile(saltPath, []byte(base64.StdEncoding.EncodeToString(ks.salt)), 0600); err != nil {
		return fmt.Errorf("failed to save salt: %w", err)
	}

	return nil
}

// deriveHardwareKey derives a master key from hardware-specific entropy
// This provides zero-knowledge persistence - the database cannot be decrypted
// on a different server because the hardware signatures won't match.
func (ks *Keystore) deriveHardwareKey() ([]byte, error) {
	// Collect hardware entropy
	entropy := ks.collectEntropy()

	// Use PBKDF2-HMAC-SHA512 with the persisted salt
	// This matches SQLCipher's default KDF parameters
	key := pbkdf2.Key(entropy, ks.salt, pbkdf2Iterations, keyLength, sha512.New)

	return key, nil
}

// collectEntropy gathers entropy from hardware-specific sources
// This binds the database to the specific VPS instance.
func (ks *Keystore) collectEntropy() []byte {
	var entropyParts []string

	// 1. Machine ID (Linux D-Bus machine-id) - most reliable for VPS
	if id, err := ks.readFile("/etc/machine-id"); err == nil && id != "" {
		entropyParts = append(entropyParts, strings.TrimSpace(id))
	}
	if id, err := ks.readFile("/var/lib/dbus/machine-id"); err == nil && id != "" {
		entropyParts = append(entropyParts, strings.TrimSpace(id))
	}

	// 2. DMI product UUID (SMBIOS) - hardware signature
	if uuid, err := ks.readDMIProductUUID(); err == nil && uuid != "" {
		entropyParts = append(entropyParts, uuid)
	}

	// 3. Primary MAC address - network identity
	if mac, err := ks.getPrimaryMAC(); err == nil && mac != "" {
		entropyParts = append(entropyParts, mac)
	}

	// 4. Hostname - container/VM identifier
	if hostname, err := os.Hostname(); err == nil {
		entropyParts = append(entropyParts, hostname)
	}

	// 5. OS and architecture
	entropyParts = append(entropyParts, runtime.GOOS, runtime.GOARCH)

	// 6. CPU info - additional hardware binding
	if cpuInfo, err := ks.getCPUInfo(); err == nil && cpuInfo != "" {
		entropyParts = append(entropyParts, cpuInfo)
	}

	// Combine all entropy sources
	// The salt ensures uniqueness even if two VPS have similar hardware
	combined := strings.Join(entropyParts, ":")

	return []byte(combined)
}

// readFile reads a file and returns its contents
func (ks *Keystore) readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readDMIProductUUID reads the DMI product UUID from /sys/class/dmi/id/product_uuid
func (ks *Keystore) readDMIProductUUID() (string, error) {
	// Try DMI product UUID first (most reliable for VPS)
	if uuid, err := ks.readFile("/sys/class/dmi/id/product_uuid"); err == nil {
		uuid = strings.TrimSpace(uuid)
		if uuid != "" && uuid != "Not Settable" && uuid != "Not Present" {
			return uuid, nil
		}
	}

	// Fallback to dmidecode command (if available)
	if _, err := exec.LookPath("dmidecode"); err == nil {
		cmd := exec.Command("dmidecode", "-s", "system-uuid")
		output, err := cmd.Output()
		if err == nil {
			uuid := strings.TrimSpace(string(output))
			if uuid != "" && uuid != "Not Settable" && uuid != "Not Present" {
				return uuid, nil
			}
		}
	}

	return "", errors.New("could not read DMI product UUID")
}

// getPrimaryMAC gets the MAC address of the primary network interface
func (ks *Keystore) getPrimaryMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Find the first non-loopback interface with a MAC address
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			if len(iface.HardwareAddr) > 0 {
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	return "", errors.New("no suitable network interface found")
}

// getCPUInfo reads CPU information for additional hardware binding
func (ks *Keystore) getCPUInfo() (string, error) {
	// Try /proc/cpuinfo (Linux)
	if info, err := ks.readFile("/proc/cpuinfo"); err == nil {
		// Extract a few identifying fields
		scanner := bufio.NewScanner(strings.NewReader(info))
		var fields []string
		count := 0
		for scanner.Scan() && count < 3 {
			line := scanner.Text()
			if strings.Contains(line, "model name") || strings.Contains(line, "vendor_id") {
				fields = append(fields, strings.TrimSpace(line))
				count++
			}
		}
		if len(fields) > 0 {
			return strings.Join(fields, ","), nil
		}
	}

	return "", errors.New("could not read CPU info")
}

// SetAuditLogger sets the audit logger for the keystore
func (ks *Keystore) SetAuditLogger(logger *audit.CriticalOperationLogger) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.auditLogger = logger
}

// Open opens and initializes the keystore database with SQLCipher encryption
func (ks *Keystore) Open() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.isOpen {
		return nil
	}

	// SQLCipher connection string with encryption pragmas
	// The key is passed as a hex string
	keyHex := hex.EncodeToString(ks.masterKey)

	dsn := fmt.Sprintf(
		"file:%s?_pragma_key=x'%s'&_pragma_cipher_page_size=%d&_pragma_kdf_iter=%d&_pragma_cipher_hmac_algorithm=%s&_pragma_cipher_kdf_algorithm=%s&_foreign_keys=ON",
		ks.dbPath,
		keyHex,
		cipherPageSize,
		cipherKdfIter,
		cipherHmacAlg,
		cipherKdfAlgorithm,
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection and verify encryption
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize schema
	if err := ks.initSchema(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	ks.db = db
	ks.isOpen = true

	return nil
}

// initSchema creates the database schema if it doesn't exist
func (ks *Keystore) initSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		token_encrypted BLOB NOT NULL,
		nonce BLOB NOT NULL,
		display_name TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER,
		tags TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_provider ON credentials(provider);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON credentials(expires_at);

	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS hardware_binding (
		signature_hash TEXT PRIMARY KEY,
		bound_at INTEGER NOT NULL,
		entropy_sources TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS matrix_refresh_tokens (
		id TEXT PRIMARY KEY,
		token_encrypted BLOB NOT NULL,
		nonce BLOB NOT NULL,
		homeserver_url TEXT NOT NULL,
		user_id TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_profiles (
		id TEXT PRIMARY KEY,
		profile_name TEXT NOT NULL,
		profile_type TEXT NOT NULL DEFAULT 'personal',
		data_encrypted BLOB NOT NULL,
		data_nonce BLOB NOT NULL,
		field_schema TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		last_accessed INTEGER,
		is_default INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_profile_type ON user_profiles(profile_type);
	CREATE INDEX IF NOT EXISTS idx_profile_default ON user_profiles(is_default);

	INSERT OR IGNORE INTO metadata (key, value) VALUES ('version', '1');
	INSERT OR IGNORE INTO metadata (key, value) VALUES ('created_at', ?);
	`

	_, err := db.Exec(query, time.Now().Unix())
	return err
}

// Close closes the keystore database
func (ks *Keystore) Close() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return nil
	}

	if ks.db != nil {
		if err := ks.db.Close(); err != nil {
			return err
		}
		ks.db = nil
	}

	ks.isOpen = false
	return nil
}

// GetDB returns the underlying database connection for sharing with crypto store
// This allows the crypto store to use the same encrypted database
func (ks *Keystore) GetDB() *sql.DB {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.db
}

// GetDBPath returns the path to the keystore database
func (ks *Keystore) GetDBPath() string {
	return ks.dbPath
}

// Store stores an encrypted credential
func (ks *Keystore) Store(cred Credential) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	// Validate provider
	if !isValidProvider(cred.Provider) {
		return ErrInvalidProvider
	}

	// Validate token format
	if cred.Token == "" {
		return ErrInvalidCredential
	}

	// Encrypt the token using XChaCha20-Poly1305
	encrypted, nonce, err := ks.encrypt([]byte(cred.Token))
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Serialize tags
	tagsJSON := "[]"
	if len(cred.Tags) > 0 {
		if data, err := json.Marshal(cred.Tags); err == nil {
			tagsJSON = string(data)
		}
	}

	// Insert into database
	query := `
	INSERT OR REPLACE INTO credentials
	(id, provider, token_encrypted, nonce, display_name, created_at, expires_at, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = ks.db.Exec(query,
		cred.ID,
		string(cred.Provider),
		encrypted,
		nonce,
		cred.DisplayName,
		cred.CreatedAt,
		cred.ExpiresAt,
		tagsJSON,
	)

	return err
}

// Retrieve retrieves and decrypts a credential
func (ks *Keystore) Retrieve(id string) (*Credential, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, provider, token_encrypted, nonce, display_name, created_at, expires_at, tags
	FROM credentials WHERE id = ?
	`

	row := ks.db.QueryRow(query, id)

	var cred Credential
	var encryptedToken, nonce []byte
	var tagsJSON string

	err := row.Scan(
		&cred.ID,
		&cred.Provider,
		&encryptedToken,
		&nonce,
		&cred.DisplayName,
		&cred.CreatedAt,
		&cred.ExpiresAt,
		&tagsJSON,
	)

	if err == sql.ErrNoRows {
		// Log failed access to audit
		if ks.auditLogger != nil {
			ks.auditLogger.LogKeyAccess(context.Background(), id, "system", "retrieve", false)
		}
		return nil, ErrKeyNotFound
	}
	if err != nil {
		if ks.auditLogger != nil {
			ks.auditLogger.LogKeyAccess(context.Background(), id, "system", "retrieve", false)
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Check expiration
	if cred.ExpiresAt > 0 {
		if time.Now().Unix() > cred.ExpiresAt {
			if ks.auditLogger != nil {
				ks.auditLogger.LogKeyAccess(context.Background(), id, "system", "retrieve", false)
			}
			return nil, ErrKeyExpired
		}
	}

	// Decrypt token using XChaCha20-Poly1305
	token, err := ks.decrypt(encryptedToken, nonce)
	if err != nil {
		if ks.auditLogger != nil {
			ks.auditLogger.LogKeyAccess(context.Background(), id, "system", "retrieve", false)
		}
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	cred.Token = string(token)

	// Parse tags
	if tagsJSON != "[]" {
		json.Unmarshal([]byte(tagsJSON), &cred.Tags)
	}

	// Log successful access to audit
	if ks.auditLogger != nil {
		ks.auditLogger.LogKeyAccess(context.Background(), id, "system", "retrieve", true)
	}

	return &cred, nil
}

// List returns all stored key information (without decrypting tokens)
func (ks *Keystore) List(provider Provider) ([]KeyInfo, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, provider, display_name, created_at, expires_at, tags
	FROM credentials
	`

	args := []interface{}{}
	if provider != "" {
		query += " WHERE provider = ?"
		args = append(args, string(provider))
	}

	rows, err := ks.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var keys []KeyInfo
	for rows.Next() {
		var info KeyInfo
		var tagsJSON string

		err := rows.Scan(
			&info.ID,
			&info.Provider,
			&info.DisplayName,
			&info.CreatedAt,
			&info.ExpiresAt,
			&tagsJSON,
		)
		if err != nil {
			continue
		}

		if tagsJSON != "[]" {
			json.Unmarshal([]byte(tagsJSON), &info.Tags)
		}

		keys = append(keys, info)
	}

	return keys, nil
}

// Delete removes a credential from the keystore
func (ks *Keystore) Delete(id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	_, err := ks.db.Exec("DELETE FROM credentials WHERE id = ?", id)

	// Log deletion to audit
	if ks.auditLogger != nil {
		if err != nil {
			ks.auditLogger.LogKeyDeleted(context.Background(), id, "system")
		} else {
			ks.auditLogger.LogKeyDeleted(context.Background(), id, "system")
		}
	}

	return err
}

// encrypt encrypts data using XChaCha20-Poly1305 AEAD
func (ks *Keystore) encrypt(plaintext []byte) (encrypted, nonce []byte, err error) {
	// Generate a random 24-byte nonce for XChaCha20-Poly1305
	nonce = make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create XChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.NewX(ks.masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Encrypt and authenticate the plaintext
	encrypted = aead.Seal(nil, nonce, plaintext, nil)

	return encrypted, nonce, nil
}

// decrypt decrypts data using XChaCha20-Poly1305 AEAD with tamper detection
func (ks *Keystore) decrypt(encrypted, nonce []byte) ([]byte, error) {
	// Validate inputs
	if len(encrypted) == 0 {
		return nil, fmt.Errorf("cannot decrypt empty data")
	}
	if len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("invalid nonce size: %d (expected %d)", len(nonce), chacha20poly1305.NonceSizeX)
	}

	// Create XChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.NewX(ks.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decrypt and verify the ciphertext
	// AEAD authentication failure indicates potential tampering
	plaintext, err := aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		// Decryption failure - possible tampering or data corruption
		// In production, this should be logged for security monitoring
		return nil, fmt.Errorf("decryption failed (data may be tampered or corrupted): %w", err)
	}

	return plaintext, nil
}

// isValidProvider checks if a provider is valid
func isValidProvider(p Provider) bool {
	switch p {
	case ProviderOpenAI, ProviderAnthropic, ProviderOpenRouter, ProviderGoogle, ProviderXAI:
		return true
	default:
		return false
	}
}

// isRetryableError checks if a database error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for database locked error (SQLite/SQLCipher specific)
	if strings.Contains(err.Error(), "database is locked") {
		return true
	}

	// Check for busy error
	if strings.Contains(err.Error(), "database is busy") {
		return true
	}

	// Check for I/O errors that might be transient
	if strings.Contains(err.Error(), "I/O") || strings.Contains(err.Error(), "timeout") {
		return true
	}

	return false
}

// RetrieveWithRetry retrieves a credential with retry logic for transient database errors
func (ks *Keystore) RetrieveWithRetry(id string, maxAttempts int) (*Credential, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	const baseDelay = 50 * time.Millisecond
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		cred, err := ks.Retrieve(id)
		if err == nil {
			return cred, nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err // Non-retryable error
		}

		// Don't wait after the last attempt
		if attempt < maxAttempts-1 {
			backoff := baseDelay * time.Duration(attempt+1)
			time.Sleep(backoff)
			lastErr = err
		}
	}

	return nil, fmt.Errorf("retrieve failed after %d attempts: %w", maxAttempts, lastErr)
}

// validateSalt checks if the current salt is valid
// Returns false if salt is nil, wrong length, or appears corrupted
func (ks *Keystore) validateSalt() bool {
	if ks.salt == nil {
		return false
	}

	if len(ks.salt) != saltLength {
		return false
	}

	// Check for all-zero salt (corruption indicator)
	allZeros := true
	for _, b := range ks.salt {
		if b != 0 {
			allZeros = false
			break
		}
	}
	return !allZeros
}

// loadOrGenerateSaltWithValidation loads or generates salt with validation
func (ks *Keystore) loadOrGenerateSaltWithValidation() error {
	saltPath := ks.dbPath + ".salt"

	// Try to load existing salt
	if data, err := os.ReadFile(saltPath); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(decoded) == saltLength {
			ks.salt = decoded

			// Validate loaded salt
			if ks.validateSalt() {
				return nil
			}

			// Salt validation failed, regenerate
			// In production, this should be logged as a security concern
			ks.salt = nil
		}
	}

	// Generate new salt
	ks.salt = make([]byte, saltLength)
	if _, err := io.ReadFull(cryptorand.Reader, ks.salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Persist salt
	if err := os.WriteFile(saltPath, []byte(base64.StdEncoding.EncodeToString(ks.salt)), 0600); err != nil {
		return fmt.Errorf("failed to persist salt: %w", err)
	}

	return nil
}

// P1-HIGH-1: MatrixRefreshToken stores encrypted Matrix refresh tokens
// Matrix refresh tokens enable long-lived sessions without requiring re-login
type MatrixRefreshToken struct {
	ID          string   // Unique identifier (e.g., "matrix-refresh-token")
	Token        string   // Encrypted refresh token
	HomeserverURL string // The homeserver this token is for
	UserID       string // The Matrix user ID
	CreatedAt     int64  // Unix timestamp when token was created
}

// StoreMatrixRefreshToken stores an encrypted Matrix refresh token
func (ks *Keystore) StoreMatrixRefreshToken(token MatrixRefreshToken) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	// Validate token data
	if token.ID == "" || token.Token == "" {
		return errors.New("invalid Matrix refresh token: id and token required")
	}
	if token.HomeserverURL == "" {
		return errors.New("invalid Matrix refresh token: homeserver URL required")
	}

	// Encrypt the refresh token using XChaCha20-Poly1305
	encryptedToken, nonce, err := ks.encrypt([]byte(token.Token))
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Insert or replace into database
	query := `
	INSERT OR REPLACE INTO matrix_refresh_tokens
	(id, token_encrypted, nonce, homeserver_url, user_id, created_at)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = ks.db.Exec(query,
		token.ID,
		encryptedToken,
		nonce,
		token.HomeserverURL,
		token.UserID,
		token.CreatedAt,
	)

	return err
}

// RetrieveMatrixRefreshToken retrieves and decrypts a Matrix refresh token
func (ks *Keystore) RetrieveMatrixRefreshToken(id string) (*MatrixRefreshToken, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, token_encrypted, nonce, homeserver_url, user_id, created_at
	FROM matrix_refresh_tokens WHERE id = ?
	`

	row := ks.db.QueryRow(query, id)

	var token MatrixRefreshToken
	var encryptedTokenData, nonce []byte

	err := row.Scan(
		&token.ID,
		&encryptedTokenData,
		&nonce,
		&token.HomeserverURL,
		&token.UserID,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Decrypt the refresh token using XChaCha20-Poly1305
	decryptedToken, err := ks.decrypt(encryptedTokenData, nonce)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	token.Token = string(decryptedToken)

	return &token, nil
}

// DeleteMatrixRefreshToken removes a stored refresh token
func (ks *Keystore) DeleteMatrixRefreshToken(id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	_, err := ks.db.Exec("DELETE FROM matrix_refresh_tokens WHERE id = ?", id)
	return err
}

// UserProfile represents a PII profile for blind fill capability
// This is a minimal interface struct - full definition is in pii/profile.go
type UserProfileData struct {
	ID           string
	ProfileName  string
	ProfileType  string
	Data         []byte // JSON-serialized and encrypted
	FieldSchema  string // JSON-serialized schema
	CreatedAt    int64
	UpdatedAt    int64
	LastAccessed int64
	IsDefault    bool
}

// ErrProfileNotFound is returned when a profile is not found
var ErrProfileNotFound = errors.New("profile not found")

// StoreProfile stores an encrypted user profile
func (ks *Keystore) StoreProfile(id, profileName, profileType string, data []byte, fieldSchema string, isDefault bool) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	// Validate inputs
	if id == "" {
		return errors.New("profile id is required")
	}
	if profileName == "" {
		return errors.New("profile name is required")
	}

	// Encrypt the profile data using XChaCha20-Poly1305
	encrypted, nonce, err := ks.encrypt(data)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	now := time.Now().Unix()

	// If setting as default, clear other defaults of same type
	if isDefault {
		_, _ = ks.db.Exec("UPDATE user_profiles SET is_default = 0 WHERE profile_type = ?", profileType)
	}

	// Insert or replace into database
	query := `
	INSERT OR REPLACE INTO user_profiles
	(id, profile_name, profile_type, data_encrypted, data_nonce, field_schema, created_at, updated_at, last_accessed, is_default)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var createdAt int64
	row := ks.db.QueryRow("SELECT created_at FROM user_profiles WHERE id = ?", id)
	_ = row.Scan(&createdAt) // Ignore error - will use new timestamp if not exists

	if createdAt == 0 {
		createdAt = now
	}

	_, err = ks.db.Exec(query,
		id,
		profileName,
		profileType,
		encrypted,
		nonce,
		fieldSchema,
		createdAt,
		now,
		nil, // last_accessed is NULL initially
		isDefault,
	)

	// Log to audit if available
	if ks.auditLogger != nil {
		ks.auditLogger.LogProfileStored(context.Background(), id, profileType)
	}

	return err
}

// RetrieveProfile retrieves and decrypts a user profile
func (ks *Keystore) RetrieveProfile(id string) (*UserProfileData, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, profile_name, profile_type, data_encrypted, data_nonce, field_schema, created_at, updated_at, last_accessed, is_default
	FROM user_profiles WHERE id = ?
	`

	row := ks.db.QueryRow(query, id)

	var profile UserProfileData
	var encryptedData, nonce []byte
	var lastAccessed sql.NullInt64

	err := row.Scan(
		&profile.ID,
		&profile.ProfileName,
		&profile.ProfileType,
		&encryptedData,
		&nonce,
		&profile.FieldSchema,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&lastAccessed,
		&profile.IsDefault,
	)

	if err == sql.ErrNoRows {
		if ks.auditLogger != nil {
			ks.auditLogger.LogProfileAccess(context.Background(), id, "retrieve", false)
		}
		return nil, ErrProfileNotFound
	}
	if err != nil {
		if ks.auditLogger != nil {
			ks.auditLogger.LogProfileAccess(context.Background(), id, "retrieve", false)
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Decrypt profile data using XChaCha20-Poly1305
	decrypted, err := ks.decrypt(encryptedData, nonce)
	if err != nil {
		if ks.auditLogger != nil {
			ks.auditLogger.LogProfileAccess(context.Background(), id, "retrieve", false)
		}
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	profile.Data = decrypted

	if lastAccessed.Valid {
		profile.LastAccessed = lastAccessed.Int64
	}

	// Update last accessed timestamp asynchronously (don't block on this)
	go func() {
		ks.mu.Lock()
		defer ks.mu.Unlock()
		ks.db.Exec("UPDATE user_profiles SET last_accessed = ? WHERE id = ?", time.Now().Unix(), id)
	}()

	// Log successful access
	if ks.auditLogger != nil {
		ks.auditLogger.LogProfileAccess(context.Background(), id, "retrieve", true)
	}

	return &profile, nil
}

// ListProfiles returns all stored profiles (without decrypting data)
func (ks *Keystore) ListProfiles(profileType string) ([]UserProfileData, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, profile_name, profile_type, field_schema, created_at, updated_at, last_accessed, is_default
	FROM user_profiles
	`

	args := []interface{}{}
	if profileType != "" {
		query += " WHERE profile_type = ?"
		args = append(args, profileType)
	}

	query += " ORDER BY is_default DESC, profile_name ASC"

	rows, err := ks.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var profiles []UserProfileData
	for rows.Next() {
		var profile UserProfileData
		var lastAccessed sql.NullInt64
		var schema sql.NullString

		err := rows.Scan(
			&profile.ID,
			&profile.ProfileName,
			&profile.ProfileType,
			&schema,
			&profile.CreatedAt,
			&profile.UpdatedAt,
			&lastAccessed,
			&profile.IsDefault,
		)
		if err != nil {
			continue
		}

		if schema.Valid {
			profile.FieldSchema = schema.String
		}
		if lastAccessed.Valid {
			profile.LastAccessed = lastAccessed.Int64
		}

		// Note: Data is not included (not decrypted for list operation)
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// DeleteProfile removes a profile from the keystore
func (ks *Keystore) DeleteProfile(id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	result, err := ks.db.Exec("DELETE FROM user_profiles WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrProfileNotFound
	}

	// Log deletion to audit
	if ks.auditLogger != nil {
		ks.auditLogger.LogProfileDeleted(context.Background(), id)
	}

	return nil
}

// GetDefaultProfile returns the default profile for a given type
func (ks *Keystore) GetDefaultProfile(profileType string) (*UserProfileData, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.isOpen {
		return nil, errors.New("keystore is not open")
	}

	query := `
	SELECT id, profile_name, profile_type, data_encrypted, data_nonce, field_schema, created_at, updated_at, last_accessed, is_default
	FROM user_profiles WHERE profile_type = ? AND is_default = 1
	LIMIT 1
	`

	row := ks.db.QueryRow(query, profileType)

	var profile UserProfileData
	var encryptedData, nonce []byte
	var lastAccessed sql.NullInt64

	err := row.Scan(
		&profile.ID,
		&profile.ProfileName,
		&profile.ProfileType,
		&encryptedData,
		&nonce,
		&profile.FieldSchema,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&lastAccessed,
		&profile.IsDefault,
	)

	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Decrypt profile data
	decrypted, err := ks.decrypt(encryptedData, nonce)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	profile.Data = decrypted

	if lastAccessed.Valid {
		profile.LastAccessed = lastAccessed.Int64
	}

	return &profile, nil
}

// SetDefaultProfile sets a profile as the default for its type
func (ks *Keystore) SetDefaultProfile(id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.isOpen {
		return errors.New("keystore is not open")
	}

	// Get the profile type first
	var profileType string
	row := ks.db.QueryRow("SELECT profile_type FROM user_profiles WHERE id = ?", id)
	err := row.Scan(&profileType)
	if err == sql.ErrNoRows {
		return ErrProfileNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get profile type: %w", err)
	}

	// Clear other defaults of same type
	_, err = ks.db.Exec("UPDATE user_profiles SET is_default = 0 WHERE profile_type = ?", profileType)
	if err != nil {
		return fmt.Errorf("failed to clear defaults: %w", err)
	}

	// Set new default
	_, err = ks.db.Exec("UPDATE user_profiles SET is_default = 1 WHERE id = ?", id)
	return err
}

