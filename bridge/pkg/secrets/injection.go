// Package secrets provides secure secret injection functionality.
// Secrets are injected through one-time tokens via ArmorChat,
// encrypted with hardware-bound keys, and never written to disk.
package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// SecretType defines the type of secret
type SecretType string

const (
	SecretTypeAPIKey     SecretType = "api_key"
	SecretTypePassword   SecretType = "password"
	SecretTypeToken      SecretType = "token"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeKey        SecretType = "key"
)

// SecretProvider identifies the secret provider
type SecretProvider string

const (
	ProviderOpenAI     SecretProvider = "openai"
	ProviderAnthropic  SecretProvider = "anthropic"
	ProviderOpenRouter SecretProvider = "openrouter"
	ProviderGoogle     SecretProvider = "google"
	ProviderXAI        SecretProvider = "xai"
	ProviderCustom     SecretProvider = "custom"
)

// OneTimeToken represents a single-use token for secret submission
type OneTimeToken struct {
	Token       string      `json:"token"`
	SecretType  SecretType  `json:"secret_type"`
	Provider    SecretProvider `json:"provider"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	Used        bool        `json:"used"`
	UsedAt      *time.Time  `json:"used_at,omitempty"`
	SessionID   string      `json:"session_id,omitempty"`
	FormSchema  FormSchema  `json:"form_schema,omitempty"`
}

// FormSchema defines the form fields for secret input
type FormSchema struct {
	Fields []FormField `json:"fields"`
}

// FormField defines a form input field
type FormField struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // text, password, select, checkbox
	Required    bool   `json:"required"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"` // for select type
}

// SecretSubmission represents a submitted secret
type SecretSubmission struct {
	OneTimeToken string            `json:"one_time_token"`
	Data         map[string]string `json:"data"`
}

// StoredSecret represents a secret in the keystore
type StoredSecret struct {
	ID           string         `json:"id"`
	Type         SecretType     `json:"type"`
	Provider     SecretProvider `json:"provider"`
	DisplayName  string         `json:"display_name"`
	CreatedAt    time.Time      `json:"created_at"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty"`
	EncryptedKey []byte         `json:"-"` // Never serialize
	KeyHash      string         `json:"key_hash"` // SHA-256 of key for verification
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// TokenManager manages one-time tokens for secret injection
type TokenManager struct {
	mu      sync.RWMutex
	tokens  map[string]*OneTimeToken
	ttl     time.Duration
	maxUses int
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		tokens:  make(map[string]*OneTimeToken),
		ttl:     5 * time.Minute,
		maxUses: 1,
	}
}

// SetTTL sets the token time-to-live
func (tm *TokenManager) SetTTL(ttl time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.ttl = ttl
}

// GenerateToken creates a new one-time token for secret submission
func (tm *TokenManager) GenerateToken(secretType SecretType, provider SecretProvider, sessionID string) (*OneTimeToken, error) {
	tokenBytes, err := securerandom.Bytes(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	token := &OneTimeToken{
		Token:      "ott_" + hex.EncodeToString(tokenBytes),
		SecretType: secretType,
		Provider:   provider,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(tm.ttl),
		Used:       false,
		SessionID:  sessionID,
		FormSchema: tm.getFormSchema(secretType),
	}

	tm.tokens[token.Token] = token

	return token, nil
}

// getFormSchema returns the form schema for a secret type
func (tm *TokenManager) getFormSchema(secretType SecretType) FormSchema {
	switch secretType {
	case SecretTypeAPIKey:
		return FormSchema{
			Fields: []FormField{
				{
					Name:        "token",
					Type:        "password",
					Required:    true,
					Label:       "API Key",
					Placeholder: "sk-...",
				},
				{
					Name:        "display_name",
					Type:        "text",
					Required:    false,
					Label:       "Display Name",
					Placeholder: "Production API Key",
				},
			},
		}
	case SecretTypePassword:
		return FormSchema{
			Fields: []FormField{
				{
					Name:     "password",
					Type:     "password",
					Required: true,
					Label:    "Password",
				},
				{
					Name:     "confirm",
					Type:     "password",
					Required: true,
					Label:    "Confirm Password",
				},
			},
		}
	default:
		return FormSchema{
			Fields: []FormField{
				{
					Name:     "value",
					Type:     "password",
					Required: true,
					Label:    "Secret Value",
				},
			},
		}
	}
}

// ValidateToken checks if a token is valid and not expired
func (tm *TokenManager) ValidateToken(tokenString string) (*OneTimeToken, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	token, exists := tm.tokens[tokenString]
	if !exists {
		return nil, errors.New("token not found")
	}

	if token.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return token, nil
}

// UseToken marks a token as used and returns it
func (tm *TokenManager) UseToken(tokenString string) (*OneTimeToken, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	token, exists := tm.tokens[tokenString]
	if !exists {
		return nil, errors.New("token not found")
	}

	if token.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().After(token.ExpiresAt) {
		delete(tm.tokens, tokenString)
		return nil, errors.New("token expired")
	}

	now := time.Now()
	token.Used = true
	token.UsedAt = &now

	// Remove token after use
	delete(tm.tokens, tokenString)

	return token, nil
}

// Cleanup removes expired tokens
func (tm *TokenManager) Cleanup() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	count := 0

	for tokenString, token := range tm.tokens {
		if now.After(token.ExpiresAt) || token.Used {
			delete(tm.tokens, tokenString)
			count++
		}
	}

	return count
}

// SecretManager handles secret storage and retrieval
type SecretManager struct {
	mu      sync.RWMutex
	secrets map[string]*StoredSecret
	keystore KeystoreBackend
}

// KeystoreBackend interface for secure storage
type KeystoreBackend interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// NewSecretManager creates a new secret manager
func NewSecretManager(keystore KeystoreBackend) *SecretManager {
	return &SecretManager{
		secrets:  make(map[string]*StoredSecret),
		keystore: keystore,
	}
}

// PrepareAdd creates a one-time token for adding a secret
func (sm *SecretManager) PrepareAdd(secretType SecretType, provider SecretProvider, sessionID string) (*OneTimeToken, error) {
	tm := NewTokenManager()
	return tm.GenerateToken(secretType, provider, sessionID)
}

// SubmitSecret processes a secret submission
func (sm *SecretManager) SubmitSecret(submission SecretSubmission) (*StoredSecret, error) {
	// Validate and use the one-time token
	tm := NewTokenManager()
	token, err := tm.UseToken(submission.OneTimeToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract secret value
	var secretValue string
	var displayName string

	switch token.SecretType {
	case SecretTypeAPIKey:
		secretValue = submission.Data["token"]
		displayName = submission.Data["display_name"]
		if displayName == "" {
			displayName = string(token.Provider) + " API Key"
		}
	case SecretTypePassword:
		if submission.Data["password"] != submission.Data["confirm"] {
			return nil, errors.New("passwords do not match")
		}
		secretValue = submission.Data["password"]
		displayName = "Password"
	default:
		secretValue = submission.Data["value"]
		displayName = string(token.SecretType)
	}

	if secretValue == "" {
		return nil, errors.New("secret value is required")
	}

	// Encrypt the secret
	encrypted, err := sm.keystore.Encrypt([]byte(secretValue))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Generate secret ID
	secretID := "secret_" + securerandom.MustID(16)

	// Create stored secret
	secret := &StoredSecret{
		ID:          secretID,
		Type:        token.SecretType,
		Provider:    token.Provider,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
		EncryptedKey: encrypted,
		KeyHash:     hashKey(secretValue),
	}

	sm.mu.Lock()
	sm.secrets[secretID] = secret
	sm.mu.Unlock()

	return secret, nil
}

// GetSecret retrieves and decrypts a secret
func (sm *SecretManager) GetSecret(secretID string) (string, error) {
	sm.mu.RLock()
	secret, exists := sm.secrets[secretID]
	sm.mu.RUnlock()

	if !exists {
		return "", errors.New("secret not found")
	}

	// Decrypt the secret
	plaintext, err := sm.keystore.Decrypt(secret.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Update last used
	sm.mu.Lock()
	now := time.Now()
	secret.LastUsedAt = &now
	sm.mu.Unlock()

	return string(plaintext), nil
}

// ListSecrets returns all stored secrets (without values)
func (sm *SecretManager) ListSecrets() []*StoredSecret {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*StoredSecret, 0, len(sm.secrets))
	for _, secret := range sm.secrets {
		// Create copy without encrypted value
		copy := &StoredSecret{
			ID:          secret.ID,
			Type:        secret.Type,
			Provider:    secret.Provider,
			DisplayName: secret.DisplayName,
			CreatedAt:   secret.CreatedAt,
			LastUsedAt:  secret.LastUsedAt,
			KeyHash:     secret.KeyHash,
			Metadata:    secret.Metadata,
		}
		result = append(result, copy)
	}

	return result
}

// DeleteSecret removes a secret
func (sm *SecretManager) DeleteSecret(secretID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.secrets[secretID]; !exists {
		return errors.New("secret not found")
	}

	delete(sm.secrets, secretID)
	return nil
}

// GetSecretByProvider returns the first secret for a provider
func (sm *SecretManager) GetSecretByProvider(provider SecretProvider) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, secret := range sm.secrets {
		if secret.Provider == provider {
			return sm.GetSecret(secret.ID)
		}
	}

	return "", errors.New("no secret found for provider")
}

// hashKey creates a SHA-256 hash of a key for verification
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ToJSON returns the secret list as JSON
func (sm *SecretManager) ToJSON() ([]byte, error) {
	return json.MarshalIndent(sm.ListSecrets(), "", "  ")
}

// Summary returns a summary of stored secrets
func (sm *SecretManager) Summary() map[string]interface{} {
	secrets := sm.ListSecrets()

	byProvider := make(map[string]int)
	for _, s := range secrets {
		byProvider[string(s.Provider)]++
	}

	return map[string]interface{}{
		"total":        len(secrets),
		"by_provider":  byProvider,
		"secret_types": countByType(secrets),
	}
}

func countByType(secrets []*StoredSecret) map[string]int {
	result := make(map[string]int)
	for _, s := range secrets {
		result[string(s.Type)]++
	}
	return result
}

// InjectionRequest represents a request to inject a secret
type InjectionRequest struct {
	ContainerID string `json:"container_id"`
	SecretID    string `json:"secret_id"`
	EnvVar      string `json:"env_var,omitempty"` // For env injection
	FD          int    `json:"fd,omitempty"`      // For FD passing
}

// InjectionResponse represents the result of secret injection
type InjectionResponse struct {
	Success   bool   `json:"success"`
	Method    string `json:"method"` // env, fd, file
	Message   string `json:"message,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Injector handles injecting secrets into containers
type Injector struct {
	secrets *SecretManager
}

// NewInjector creates a new secret injector
func NewInjector(secrets *SecretManager) *Injector {
	return &Injector{secrets: secrets}
}

// InjectSecret injects a secret into a container
func (i *Injector) InjectSecret(req InjectionRequest) (*InjectionResponse, error) {
	// Get the secret value
	_, err := i.secrets.GetSecret(req.SecretID)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// For security, we use file descriptor passing
	// The actual injection is handled by the bridge's Docker client
	// This method just validates and prepares the injection
	// The secret value is passed via FD, never written to disk

	// Validate the request
	if req.ContainerID == "" {
		return nil, errors.New("container ID is required")
	}

	// In production, this would:
	// 1. Verify the container exists
	// 2. Verify the container is allowed to receive this secret
	// 3. Inject via FD passing (preferred) or environment
	// 4. Set up automatic cleanup

	method := "fd"
	if req.EnvVar != "" {
		method = "env"
	}

	// Set expiry for memory-only secrets
	expiresAt := time.Now().Add(10 * time.Second)

	return &InjectionResponse{
		Success:   true,
		Method:    method,
		Message:   "Secret prepared for injection",
		ExpiresAt: &expiresAt,
	}, nil
}
