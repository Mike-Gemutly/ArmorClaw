package email

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/armorclaw/bridge/pkg/pii"
)

//=============================================================================
// Email HITL E2E: inbound → PII detection → Matrix event → approval → outbound
//=============================================================================

func TestE2E_HITL_InboundToApproval(t *testing.T) {
	rawEmail := []byte("From: customer@bank.com\r\nTo: agent@armorclaw.com\r\nSubject: Wire Transfer\r\nContent-Type: text/plain\r\n\r\nPlease wire $5000 from account 1234-5678-9012 to SSN 123-45-6789. My phone is 555-123-4567.")

	parsed, err := ParseMIME(rawEmail)
	require.NoError(t, err)
	assert.Equal(t, "customer@bank.com", parsed.From)
	assert.Equal(t, "Wire Transfer", parsed.Subject)
	assert.NotEmpty(t, parsed.BodyText)

	masker := pii.NewMasker()
	masked, fields := masker.MaskPII(parsed.BodyText)
	assert.GreaterOrEqual(t, len(fields), 2, "should detect SSN and phone")
	assert.NotEqual(t, parsed.BodyText, masked, "masked body should differ from original")

	piiTypes := make(map[string]bool)
	for _, f := range fields {
		piiTypes[f.Type] = true
	}
	assert.True(t, piiTypes["ssn"], "should detect SSN")

	tmpDir := t.TempDir()
	storage := NewLocalFSEmailStorage(tmpDir)
	emailID := "e2e_hitl_001"
	require.NoError(t, storage.StoreEmail(emailID, rawEmail))

	emailPath := filepath.Join(tmpDir, "emails", emailID, "raw.eml")
	_, err = os.Stat(emailPath)
	require.NoError(t, err, "stored email should exist")

	var capturedApprovalID string
	var capturedMatrixMsgs []struct {
		roomID    string
		eventType string
		body      string
	}
	var matrixMu sync.Mutex

	approvalMgr := NewEmailApprovalManager(EmailApprovalConfig{
		Timeout: 5 * time.Second,
		SendMatrixMsg: func(roomID, eventType, body string) error {
			matrixMu.Lock()
			capturedMatrixMsgs = append(capturedMatrixMsgs, struct {
				roomID    string
				eventType string
				body      string
			}{roomID, eventType, body})
			matrixMu.Unlock()
			return nil
		},
	})

	piiFieldNames := make([]string, len(fields))
	for i, f := range fields {
		piiFieldNames[i] = f.Type
	}

	outboundReq := &OutboundRequest{
		To:        "customer@bank.com",
		Subject:   "Re: Wire Transfer",
		BodyText:  masked,
		EmailID:   emailID,
		PIIFields: piiFieldNames,
	}

	approvalDone := make(chan *ApprovalDecision, 1)
	go func() {
		decision, err := approvalMgr.RequestApproval(context.Background(), outboundReq)
		if err != nil {
			approvalDone <- &ApprovalDecision{Approved: false}
			return
		}
		approvalDone <- decision
	}()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, approvalMgr.PendingCount(), "should have 1 pending approval")

	matrixMu.Lock()
	require.GreaterOrEqual(t, len(capturedMatrixMsgs), 1, "should send Matrix approval request")
	assert.Contains(t, capturedMatrixMsgs[0].eventType, "email_approval_request")
	matrixMu.Unlock()

	var approvalBody map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(capturedMatrixMsgs[0].body), &approvalBody))
	capturedApprovalID, _ = approvalBody["approval_id"].(string)
	require.NotEmpty(t, capturedApprovalID, "should capture approval_id from Matrix message")

	err = approvalMgr.HandleApprovalResponse(capturedApprovalID, true, "@admin:example.com")
	require.NoError(t, err)

	select {
	case decision := <-approvalDone:
		assert.True(t, decision.Approved, "should be approved")
		assert.Equal(t, "@admin:example.com", decision.ApprovedBy)
		assert.Equal(t, capturedApprovalID, decision.ApprovalID)
		assert.False(t, decision.ApprovedAt.IsZero())
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for approval decision")
	}

	resolutions := make(map[string]string)
	for _, f := range fields {
		resolutions[f.Placeholder] = f.Original
	}
	resolved := masker.ResolvePlaceholders(masked, resolutions)
	assert.Contains(t, resolved, "555-123-4567", "resolved body should contain original phone")
	assert.Contains(t, resolved, "123-45-6789", "resolved body should contain original SSN")
}

func TestE2E_HITL_DeniedApproval(t *testing.T) {
	var capturedBody string
	var bodyMu sync.Mutex

	approvalMgr := NewEmailApprovalManager(EmailApprovalConfig{
		Timeout: 5 * time.Second,
		SendMatrixMsg: func(roomID, eventType, body string) error {
			bodyMu.Lock()
			capturedBody = body
			bodyMu.Unlock()
			return nil
		},
	})

	outboundReq := &OutboundRequest{
		To:        "recipient@example.com",
		Subject:   "Sensitive Data",
		EmailID:   "email_deny_001",
		PIIFields: []string{"ssn"},
	}

	approvalDone := make(chan *ApprovalDecision, 1)
	go func() {
		decision, _ := approvalMgr.RequestApproval(context.Background(), outboundReq)
		approvalDone <- decision
	}()

	time.Sleep(100 * time.Millisecond)

	bodyMu.Lock()
	require.NotEmpty(t, capturedBody, "should have captured Matrix message body")
	bodyMu.Unlock()

	var approvalBody map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &approvalBody))
	approvalID, _ := approvalBody["approval_id"].(string)
	require.NotEmpty(t, approvalID)

	require.NoError(t, approvalMgr.HandleApprovalResponse(approvalID, false, "@admin:example.com"))

	select {
	case decision := <-approvalDone:
		assert.False(t, decision.Approved, "should be denied")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}

func TestE2E_HITL_ApprovalTimeout(t *testing.T) {
	approvalMgr := NewEmailApprovalManager(EmailApprovalConfig{
		Timeout: 500 * time.Millisecond,
	})

	outboundReq := &OutboundRequest{
		To:      "slow@example.com",
		Subject: "Timeout Test",
		EmailID: "email_timeout_001",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	decision, err := approvalMgr.RequestApproval(ctx, outboundReq)
	require.NoError(t, err)
	assert.False(t, decision.Approved, "should not be approved after timeout")
	assert.Empty(t, decision.ApprovedBy, "no approver on timeout")
}
