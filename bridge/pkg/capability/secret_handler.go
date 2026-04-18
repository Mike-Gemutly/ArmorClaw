// Package capability provides the request_secret bridge-local handler.
//
// The handler accepts a JSON payload describing a secret request, delegates
// to an injected SecretRequester function (which calls the HITL flow), and
// returns a SecretRef placeholder on approval — the agent never sees the
// raw secret value.
package capability

import (
	"context"
	"encoding/json"
	"fmt"
)

// SecretRequester is a function type that calls the SecretRequestManager.
// Injected to avoid importing pkg/team from pkg/capability.
type SecretRequester func(ctx context.Context, req SecretRequestParams) (*SecretResult, error)

// SecretRequestParams holds the parameters for a secret request.
type SecretRequestParams struct {
	AgentID        string `json:"agent_id"`
	TeamID         string `json:"team_id,omitempty"`
	CredentialName string `json:"credential_name"`
	TargetDomain   string `json:"target_domain"`
	Reason         string `json:"reason"`
}

// SecretResult holds the response from a secret request.
type SecretResult struct {
	RequestID string `json:"request_id"`
	Approved  bool   `json:"approved"`
	SecretRef string `json:"secret_ref,omitempty"` // placeholder like {{VAULT:field:hash}} when approved
}

// secretHandlerResponse is the JSON structure returned to callers.
type secretHandlerResponse struct {
	RequestID string `json:"request_id"`
	Approved  bool   `json:"approved"`
	SecretRef string `json:"secret_ref,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// NewSecretHandler returns a bridge-local handler for the "request_secret"
// capability. The requester function is injected so this package stays a
// leaf with no dependency on pkg/team.
func NewSecretHandler(requester SecretRequester) func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		var params SecretRequestParams
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("request_secret: invalid input: %w", err)
		}

		if params.AgentID == "" {
			return nil, fmt.Errorf("request_secret: field agent_id is required")
		}
		if params.CredentialName == "" {
			return nil, fmt.Errorf("request_secret: field credential_name is required")
		}

		result, err := requester(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("request_secret: %w", err)
		}

		resp := secretHandlerResponse{
			RequestID: result.RequestID,
			Approved:  result.Approved,
		}

		if result.Approved {
			resp.SecretRef = fmt.Sprintf("{{VAULT:%s:%s}}", params.CredentialName, result.RequestID)
		} else {
			resp.Reason = "secret request denied"
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("request_secret: marshal response: %w", err)
		}
		return raw, nil
	}
}
