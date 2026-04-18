package capability

import (
	"context"
	"strings"
)

// taxonomyEntry pairs a risk class with a default risk level.
type taxonomyEntry struct {
	Class RiskClass
	Level RiskLevel
}

// RiskClassifierImpl maps action strings to risk classifications
// using a hardcoded OWASP-aligned taxonomy.
type RiskClassifierImpl struct {
	taxonomy        map[string]taxonomyEntry
	wildcardPrefixes map[string]taxonomyEntry
}

// Verify RiskClassifierImpl satisfies the local riskClassifier interface.
var _ riskClassifier = (*RiskClassifierImpl)(nil)

// NewRiskClassifier creates the classifier with the default taxonomy.
func NewRiskClassifier() *RiskClassifierImpl {
	return &RiskClassifierImpl{
		taxonomy: map[string]taxonomyEntry{
			// Browser actions
			"browser.browse":     {RiskExternalCommunication, RiskAllow},
			"browser.navigate":   {RiskExternalCommunication, RiskAllow},
			"browser.screenshot": {RiskExternalCommunication, RiskAllow},
			"browser.fill_forms": {RiskExternalCommunication, RiskDefer},
			"browser.submit":     {RiskExternalCommunication, RiskDefer},
			"browser.click":      {RiskExternalCommunication, RiskAllow},
			// Email actions
			"email.send":  {RiskExternalCommunication, RiskDefer},
			"email.draft": {RiskExternalCommunication, RiskAllow},
			"email.read":  {RiskExternalCommunication, RiskAllow},
			// Secret/credential actions
			"secret.access":  {RiskCredentialUse, RiskDefer},
			"secret.request": {RiskCredentialUse, RiskDefer},
			"secret.list":    {RiskCredentialUse, RiskDefer},
			// Payment actions
			"payment.process": {RiskPayment, RiskDefer},
			"payment.refund":  {RiskPayment, RiskDefer},
			"payment.view":    {RiskPayment, RiskAllow},
			// PII actions
			"pii.read":   {RiskIdentityPII, RiskDefer},
			"pii.export": {RiskIdentityPII, RiskDefer},
			"pii.mask":   {RiskIdentityPII, RiskAllow},
			// Document actions
			"doc.query":    {RiskFileExfiltration, RiskAllow},
			"doc.upload":   {RiskFileExfiltration, RiskAllow},
			"doc.delete":   {RiskFileExfiltration, RiskDefer},
			"doc.download": {RiskFileExfiltration, RiskDefer},
		},
		wildcardPrefixes: map[string]taxonomyEntry{
			"browser.": {RiskExternalCommunication, RiskAllow},
			"email.":   {RiskExternalCommunication, RiskDefer},
			"secret.":  {RiskCredentialUse, RiskDefer},
			"payment.": {RiskPayment, RiskDefer},
			"pii.":     {RiskIdentityPII, RiskDefer},
			"doc.":     {RiskFileExfiltration, RiskAllow},
		},
	}
}

// Classify returns the risk class and level for the given action.
// Lookup order: exact match → prefix match → default DENY.
func (rc *RiskClassifierImpl) Classify(_ context.Context, action string, _ map[string]any) (RiskClass, RiskLevel) {
	// Exact match takes priority.
	if entry, ok := rc.taxonomy[action]; ok {
		return entry.Class, entry.Level
	}

	// Prefix match for unlisted actions under known domains.
	for prefix, entry := range rc.wildcardPrefixes {
		if strings.HasPrefix(action, prefix) {
			return entry.Class, entry.Level
		}
	}

	// Unknown action: deny by default.
	return RiskIrreversibleAction, RiskDeny
}
