package capability

import (
	"encoding/json"
	"reflect"
	"testing"
)

// ============================================================================
// Round-trip JSON tests
// ============================================================================

func TestArtifactRoundTrip(t *testing.T) {
	t.Run("RiskLevel", func(t *testing.T) {
		roundTripStringType[RiskLevel](t, RiskAllow)
	})
	t.Run("RiskClass", func(t *testing.T) {
		roundTripStringType[RiskClass](t, RiskPayment)
	})

	ptrTests := []struct {
		name string
		val  any
	}{
		{
			name: "ActionRequest",
			val: &ActionRequest{
				AgentID: "agent-1",
				TeamID:  "team-1",
				Action:  "browse",
				Params:  map[string]any{"url": "https://example.com", "timeout": float64(30)},
			},
		},
		{
			name: "ActionResponse",
			val: &ActionResponse{
				Allowed:        true,
				Classification: RiskAllow,
				Reason:         "low risk",
				SessionID:      "sess-123",
				RiskClass:      RiskExternalCommunication,
			},
		},
		{
			name: "CapabilitySet",
			val:  CapabilitySet{"browse": true, "email": false},
		},
		{
			name: "SecretRef",
			val: &SecretRef{
				Field:   "card_number",
				Hash:    "sha256:abc",
				Version: 3,
			},
		},
		{
			name: "BrowserIntent",
			val: &BrowserIntent{
				URL:        "https://example.com/form",
				Action:     "fill",
				FormFields: []string{"name", "email"},
			},
		},
		{
			name: "BrowserResult",
			val: &BrowserResult{
				URL:           "https://example.com/result",
				Title:         "Result Page",
				ExtractedData: []string{"data1", "data2"},
				Screenshots:   []string{"shot1.png"},
			},
		},
		{
			name: "DocumentRef",
			val: &DocumentRef{
				CollectionID: "col-42",
				ChunkIDs:     []string{"c1", "c2"},
			},
		},
		{
			name: "ExtractedChunkSet",
			val: &ExtractedChunkSet{
				Chunks:  []string{"chunk-a", "chunk-b"},
				Summary: "two chunks extracted",
			},
		},
		{
			name: "EmailDraft",
			val: &EmailDraft{
				To:          "user@example.com",
				Subject:     "Hello",
				BodyMasked:  "Hi ***",
				Attachments: []string{"file.pdf"},
			},
		},
		{
			name: "ApprovalDecision",
			val: &ApprovalDecision{
				Approved:     true,
				Fields:       []string{"amount", "payee"},
				DeniedFields: []string{"note"},
			},
		},
		{
			name: "WorkflowBlocker",
			val: &WorkflowBlocker{
				Type:       "payment_limit",
				Message:    "daily limit exceeded",
				Suggestion: "retry tomorrow",
			},
		},
		{
			name: "SecretRequestEvent",
			val: &SecretRequestEvent{
				RequestID:      "req-1",
				AgentID:        "agent-1",
				TeamID:         "team-1",
				CredentialName: "api_key",
				TargetDomain:   "api.example.com",
				Reason:         "need to call external API",
				RiskClass:      "credential_use",
			},
		},
		{
			name: "SecretResponseEvent",
			val: &SecretResponseEvent{
				RequestID:   "req-1",
				Approved:    true,
				RespondedBy: "user-1",
			},
		},
	}

	for _, tc := range ptrTests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.val)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			target := newZero(tc.val)
			if err := json.Unmarshal(data, target); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			got := derefIfPtr(target)
			want := derefIfPtr(tc.val)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round-trip mismatch\ngot:  %+v\nwant: %+v", got, tc.val)
			}
		})
	}
}

func roundTripStringType[T ~string](t *testing.T, val T) {
	t.Helper()
	data, err := json.Marshal(val)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got T
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != val {
		t.Errorf("round-trip mismatch: got %q, want %q", got, val)
	}
}

func newZero(v any) any {
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Ptr {
		return reflect.New(rt.Elem()).Interface()
	}
	return reflect.New(rt).Interface()
}

func derefIfPtr(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		return rv.Elem().Interface()
	}
	return v
}

// ============================================================================
// Backward compatibility tests
// ============================================================================

func TestArtifactBackwardCompat(t *testing.T) {
	t.Run("ActionRequest_old_format", func(t *testing.T) {
		raw := `{"agent_id":"a","action":"b"}`
		var ar ActionRequest
		if err := json.Unmarshal([]byte(raw), &ar); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ar.AgentID != "a" || ar.Action != "b" {
			t.Fatalf("got agent_id=%q action=%q", ar.AgentID, ar.Action)
		}
		if ar.TeamID != "" {
			t.Errorf("TeamID should be empty, got %q", ar.TeamID)
		}
		if ar.Params != nil {
			t.Errorf("Params should be nil, got %v", ar.Params)
		}
	})

	t.Run("ActionResponse_old_format", func(t *testing.T) {
		raw := `{"allowed":true,"classification":"DENY"}`
		var ar ActionResponse
		if err := json.Unmarshal([]byte(raw), &ar); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ar.Allowed != true || ar.Classification != RiskDeny {
			t.Fatalf("got allowed=%v classification=%q", ar.Allowed, ar.Classification)
		}
		if ar.Reason != "" || ar.SessionID != "" || ar.RiskClass != "" {
			t.Error("optional fields should be zero-valued")
		}
	})

	t.Run("SecretRef_old_format", func(t *testing.T) {
		raw := `{"field":"api_key"}`
		var sr SecretRef
		if err := json.Unmarshal([]byte(raw), &sr); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if sr.Field != "api_key" {
			t.Fatalf("got field=%q", sr.Field)
		}
		if sr.Hash != "" || sr.Version != 0 {
			t.Error("optional fields should be zero-valued")
		}
	})

	t.Run("BrowserIntent_old_format", func(t *testing.T) {
		raw := `{"url":"https://example.com","action":"navigate"}`
		var bi BrowserIntent
		if err := json.Unmarshal([]byte(raw), &bi); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if bi.FormFields != nil {
			t.Errorf("FormFields should be nil, got %v", bi.FormFields)
		}
	})

	t.Run("BrowserResult_old_format", func(t *testing.T) {
		raw := `{"url":"https://example.com"}`
		var br BrowserResult
		if err := json.Unmarshal([]byte(raw), &br); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if br.Title != "" || br.ExtractedData != nil || br.Screenshots != nil {
			t.Error("optional fields should be zero-valued")
		}
	})

	t.Run("DocumentRef_old_format", func(t *testing.T) {
		raw := `{"collection_id":"col-1"}`
		var dr DocumentRef
		if err := json.Unmarshal([]byte(raw), &dr); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if dr.ChunkIDs != nil {
			t.Errorf("ChunkIDs should be nil, got %v", dr.ChunkIDs)
		}
	})

	t.Run("ExtractedChunkSet_old_format", func(t *testing.T) {
		raw := `{"chunks":["a","b"]}`
		var ecs ExtractedChunkSet
		if err := json.Unmarshal([]byte(raw), &ecs); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ecs.Summary != "" {
			t.Errorf("Summary should be empty, got %q", ecs.Summary)
		}
	})

	t.Run("EmailDraft_old_format", func(t *testing.T) {
		raw := `{"to":"a@b.com","subject":"hi","body_masked":"***"}`
		var ed EmailDraft
		if err := json.Unmarshal([]byte(raw), &ed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ed.Attachments != nil {
			t.Errorf("Attachments should be nil, got %v", ed.Attachments)
		}
	})

	t.Run("ApprovalDecision_old_format", func(t *testing.T) {
		raw := `{"approved":false}`
		var ad ApprovalDecision
		if err := json.Unmarshal([]byte(raw), &ad); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ad.Fields != nil || ad.DeniedFields != nil {
			t.Error("optional fields should be nil")
		}
	})

	t.Run("WorkflowBlocker_old_format", func(t *testing.T) {
		raw := `{"type":"cap","message":"limit reached"}`
		var wb WorkflowBlocker
		if err := json.Unmarshal([]byte(raw), &wb); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if wb.Suggestion != "" {
			t.Errorf("Suggestion should be empty, got %q", wb.Suggestion)
		}
	})

	t.Run("SecretRequestEvent_old_format", func(t *testing.T) {
		raw := `{"request_id":"req-1","agent_id":"agent-1","credential_name":"api_key","target_domain":"x.com","reason":"test","risk_class":"credential_use"}`
		var sre SecretRequestEvent
		if err := json.Unmarshal([]byte(raw), &sre); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if sre.RequestID != "req-1" || sre.AgentID != "agent-1" {
			t.Fatalf("got request_id=%q agent_id=%q", sre.RequestID, sre.AgentID)
		}
		if sre.TeamID != "" {
			t.Errorf("TeamID should be empty, got %q", sre.TeamID)
		}
	})

	t.Run("SecretResponseEvent_old_format", func(t *testing.T) {
		raw := `{"request_id":"req-1","approved":true}`
		var sre SecretResponseEvent
		if err := json.Unmarshal([]byte(raw), &sre); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if sre.RequestID != "req-1" || !sre.Approved {
			t.Fatalf("got request_id=%q approved=%v", sre.RequestID, sre.Approved)
		}
		if sre.RespondedBy != "" {
			t.Errorf("RespondedBy should be empty, got %q", sre.RespondedBy)
		}
	})
}

// ============================================================================
// Validation tests
// ============================================================================

func TestValidate_ValidCases(t *testing.T) {
	valid := []struct {
		name string
		val  interface{ Validate() error }
	}{
		{"ActionRequest", &ActionRequest{AgentID: "a", Action: "b"}},
		{"ActionResponse_ALLOW", &ActionResponse{Classification: RiskAllow}},
		{"ActionResponse_DENY", &ActionResponse{Classification: RiskDeny}},
		{"ActionResponse_DEFER", &ActionResponse{Classification: RiskDefer}},
		{"CapabilitySet", CapabilitySet{"x": true}},
		{"SecretRef", &SecretRef{Field: "f"}},
		{"BrowserIntent", &BrowserIntent{URL: "u", Action: "a"}},
		{"BrowserResult", &BrowserResult{URL: "u"}},
		{"DocumentRef", &DocumentRef{CollectionID: "c"}},
		{"ExtractedChunkSet", &ExtractedChunkSet{Chunks: []string{"x"}}},
		{"EmailDraft", &EmailDraft{To: "t", Subject: "s"}},
		{"ApprovalDecision", &ApprovalDecision{}},
		{"ApprovalDecision_denied", &ApprovalDecision{Approved: false}},
		{"WorkflowBlocker", &WorkflowBlocker{Type: "t", Message: "m"}},
		{"SecretRequestEvent", &SecretRequestEvent{RequestID: "r", AgentID: "a"}},
		{"SecretResponseEvent", &SecretResponseEvent{RequestID: "r"}},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.val.Validate(); err != nil {
				t.Errorf("expected nil, got error: %v", err)
			}
		})
	}
}

func TestValidate_InvalidCases(t *testing.T) {
	tests := []struct {
		name        string
		val         interface{ Validate() error }
		wantErr     bool
		errContains string
	}{
		{
			name:        "ActionRequest_empty_agent_id",
			val:         &ActionRequest{Action: "b"},
			wantErr:     true,
			errContains: "agent_id",
		},
		{
			name:        "ActionRequest_empty_action",
			val:         &ActionRequest{AgentID: "a"},
			wantErr:     true,
			errContains: "action",
		},
		{
			name:        "ActionResponse_invalid_classification",
			val:         &ActionResponse{Classification: RiskLevel("UNKNOWN")},
			wantErr:     true,
			errContains: "classification",
		},
		{
			name:        "ActionResponse_empty_classification",
			val:         &ActionResponse{Classification: ""},
			wantErr:     true,
			errContains: "classification",
		},
		{
			name:        "CapabilitySet_empty",
			val:         CapabilitySet{},
			wantErr:     true,
			errContains: "entry",
		},
		{
			name:        "SecretRef_empty_field",
			val:         &SecretRef{},
			wantErr:     true,
			errContains: "field",
		},
		{
			name:        "BrowserIntent_empty_url",
			val:         &BrowserIntent{Action: "a"},
			wantErr:     true,
			errContains: "url",
		},
		{
			name:        "BrowserIntent_empty_action",
			val:         &BrowserIntent{URL: "u"},
			wantErr:     true,
			errContains: "action",
		},
		{
			name:        "BrowserResult_empty_url",
			val:         &BrowserResult{},
			wantErr:     true,
			errContains: "url",
		},
		{
			name:        "DocumentRef_empty_collection",
			val:         &DocumentRef{},
			wantErr:     true,
			errContains: "collection_id",
		},
		{
			name:        "ExtractedChunkSet_empty",
			val:         &ExtractedChunkSet{},
			wantErr:     true,
			errContains: "chunk",
		},
		{
			name:        "ExtractedChunkSet_nil",
			val:         &ExtractedChunkSet{Chunks: nil},
			wantErr:     true,
			errContains: "chunk",
		},
		{
			name:        "EmailDraft_empty_to",
			val:         &EmailDraft{Subject: "s"},
			wantErr:     true,
			errContains: "to",
		},
		{
			name:        "EmailDraft_empty_subject",
			val:         &EmailDraft{To: "t"},
			wantErr:     true,
			errContains: "subject",
		},
		{
			name:        "WorkflowBlocker_empty_type",
			val:         &WorkflowBlocker{Message: "m"},
			wantErr:     true,
			errContains: "type",
		},
		{
			name:        "WorkflowBlocker_empty_message",
			val:         &WorkflowBlocker{Type: "t"},
			wantErr:     true,
			errContains: "message",
		},
		{
			name:        "SecretRequestEvent_empty_request_id",
			val:         &SecretRequestEvent{AgentID: "a"},
			wantErr:     true,
			errContains: "request_id",
		},
		{
			name:        "SecretRequestEvent_empty_agent_id",
			val:         &SecretRequestEvent{RequestID: "r"},
			wantErr:     true,
			errContains: "agent_id",
		},
		{
			name:        "SecretResponseEvent_empty_request_id",
			val:         &SecretResponseEvent{Approved: true},
			wantErr:     true,
			errContains: "request_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.val.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected nil, got: %v", err)
			}
			if tc.wantErr && err != nil && tc.errContains != "" {
				if !containsString(err.Error(), tc.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
				}
			}
		})
	}
}

func TestValidate_NilCapabilitySet(t *testing.T) {
	var cs CapabilitySet
	if err := cs.Validate(); err != nil {
		t.Errorf("nil CapabilitySet should be valid, got: %v", err)
	}
}

// ============================================================================
// Constant value tests
// ============================================================================

func TestRiskLevelConstants(t *testing.T) {
	tests := []struct {
		name  string
		val   RiskLevel
		want  string
	}{
		{"Allow", RiskAllow, "ALLOW"},
		{"Deny", RiskDeny, "DENY"},
		{"Defer", RiskDefer, "DEFER"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.val) != tc.want {
				t.Errorf("got %q, want %q", tc.val, tc.want)
			}
		})
	}
}

func TestRiskClassConstants(t *testing.T) {
	tests := []struct {
		name string
		val  RiskClass
		want string
	}{
		{"Payment", RiskPayment, "payment"},
		{"IdentityPII", RiskIdentityPII, "identity_pii"},
		{"CredentialUse", RiskCredentialUse, "credential_use"},
		{"ExternalCommunication", RiskExternalCommunication, "external_communication"},
		{"FileExfiltration", RiskFileExfiltration, "file_exfiltration"},
		{"IrreversibleAction", RiskIrreversibleAction, "irreversible_action"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.val) != tc.want {
				t.Errorf("got %q, want %q", tc.val, tc.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
