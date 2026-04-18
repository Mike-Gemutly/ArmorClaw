package capability

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

// SecretStorer stores an approved secret value and returns a reference.
// Injected to avoid keystore import. Returns the storage reference hash.
type SecretStorer func(ctx context.Context, credentialName, value string) (secretRef string, err error)

// SecretApprovalConfig holds constructor parameters.
type SecretApprovalConfig struct {
	// SecretStorer is optional — if nil, generates ref but doesn't persist.
	SecretStorer SecretStorer
}

// SecretApprovalPolicy determines the approval path for secret requests
// based on risk classification. Credential names are classified into risk
// categories: payment/identity secrets always require human approval (DENY),
// generic API keys may auto-approve (ALLOW), and everything else defaults
// to DENY (conservative).
type SecretApprovalPolicy struct {
	classifier   RiskClassifierImpl
	secretStorer SecretStorer
}

// NewSecretApprovalPolicy creates a new policy with the given configuration.
func NewSecretApprovalPolicy(cfg SecretApprovalConfig) *SecretApprovalPolicy {
	return &SecretApprovalPolicy{
		classifier:   *NewRiskClassifier(),
		secretStorer: cfg.SecretStorer,
	}
}

// ClassifyRisk returns the risk level for a credential based on its name.
// Payment and identity secrets → DENY (requires explicit approval).
// Generic API keys/tokens → ALLOW (auto-approve per policy).
// Unknown → DENY (conservative default).
func (p *SecretApprovalPolicy) ClassifyRisk(credentialName string) RiskLevel {
	lower := strings.ToLower(credentialName)

	// Payment secrets: always require approval.
	if strings.Contains(lower, "payment") ||
		strings.Contains(lower, "credit") ||
		strings.Contains(lower, "card") {
		return RiskDeny
	}

	// Identity/PII secrets: always require approval.
	if strings.Contains(lower, "ssn") ||
		strings.Contains(lower, "passport") ||
		strings.Contains(lower, "id_") {
		return RiskDeny
	}

	// Generic API keys/tokens: may auto-approve.
	if strings.Contains(lower, "api_key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "key") {
		return RiskAllow
	}

	// Conservative default: require approval.
	return RiskDeny
}

// ShouldAutoApprove returns true for generic API keys, false for
// payment/identity secrets and unknowns.
func (p *SecretApprovalPolicy) ShouldAutoApprove(credentialName string) bool {
	return p.ClassifyRisk(credentialName) == RiskAllow
}

// StoreApprovedSecret persists an approved secret and returns a vault
// reference placeholder. If secretStorer is nil, generates a hash-based
// reference without persisting. The returned reference format is
// {{VAULT:credentialName:hashPrefix}} — the agent never sees the raw value.
func (p *SecretApprovalPolicy) StoreApprovedSecret(ctx context.Context, credentialName, value string) (string, error) {
	if p.secretStorer != nil {
		return p.secretStorer(ctx, credentialName, value)
	}

	// No storer configured: generate hash-based reference.
	hash := sha256.Sum256([]byte(value))
	hashHex := fmt.Sprintf("%x", hash)
	ref := fmt.Sprintf("{{VAULT:%s:%s}}", credentialName, hashHex[:16])
	return ref, nil
}
