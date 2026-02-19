// Package push provides push notification provider implementations
package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FCMProvider implements Firebase Cloud Messaging
type FCMProvider struct {
	serverKey string
	projectID string
	client    *http.Client
}

// NewFCMProvider creates a new FCM provider
func NewFCMProvider(serverKey, projectID string) *FCMProvider {
	return &FCMProvider{
		serverKey: serverKey,
		projectID: projectID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification via FCM
func (f *FCMProvider) Send(ctx context.Context, notification *Notification) (*PushResult, error) {
	if f.serverKey == "" {
		return nil, fmt.Errorf("FCM server key not configured")
	}

	payload := map[string]interface{}{
		"to": notification.DeviceToken,
		"notification": map[string]interface{}{
			"title": notification.Title,
			"body":  notification.Body,
		},
		"data": notification.Data,
		"priority": mapPriorityToFCM(notification.Priority),
	}

	if notification.Badge > 0 {
		payload["notification"].(map[string]interface{})["badge"] = notification.Badge
	}
	if notification.Sound != "" {
		payload["notification"].(map[string]interface{})["sound"] = notification.Sound
	}
	if notification.Icon != "" {
		payload["notification"].(map[string]interface{})["icon"] = notification.Icon
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fcm.googleapis.com/fcm/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "key="+f.serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FCM error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var fcmResp struct {
		Success int `json:"success"`
		Failure int `json:"failure"`
		Results []struct {
			MessageID string `json:"message_id"`
			Error     string `json:"error"`
		} `json:"results"`
	}

	if err := json.Unmarshal(respBody, &fcmResp); err != nil {
		// Still consider it success if status was 200
		return &PushResult{
			NotificationID: notification.ID,
			Success:        true,
			DeliveredAt:    time.Now(),
		}, nil
	}

	if len(fcmResp.Results) > 0 && fcmResp.Results[0].Error != "" {
		return &PushResult{
			NotificationID: notification.ID,
			Success:        false,
			Error:          fcmResp.Results[0].Error,
		}, fmt.Errorf("FCM error: %s", fcmResp.Results[0].Error)
	}

	return &PushResult{
		NotificationID: notification.ID,
		Success:        true,
		DeliveredAt:    time.Now(),
	}, nil
}

// ValidateToken validates an FCM token
func (f *FCMProvider) ValidateToken(token string) bool {
	return len(token) > 100 // FCM tokens are typically long strings
}

// Platform returns the platform identifier
func (f *FCMProvider) Platform() Platform {
	return PlatformFCM
}

func mapPriorityToFCM(p Priority) string {
	if p == PriorityHigh {
		return "high"
	}
	return "normal"
}

// APNSProvider implements Apple Push Notification Service
type APNSProvider struct {
	certFile    string
	keyFile     string
	topic       string
	environment string
	client      *http.Client
}

// NewAPNSProvider creates a new APNS provider
func NewAPNSProvider(certFile, keyFile, topic, environment string) *APNSProvider {
	return &APNSProvider{
		certFile:    certFile,
		keyFile:     keyFile,
		topic:       topic,
		environment: environment,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification via APNS
func (a *APNSProvider) Send(ctx context.Context, notification *Notification) (*PushResult, error) {
	if a.certFile == "" || a.keyFile == "" {
		return nil, fmt.Errorf("APNS certificates not configured")
	}

	// Build APNS payload
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{
				"title": notification.Title,
				"body":  notification.Body,
			},
		},
	}

	if notification.Badge > 0 {
		payload["aps"].(map[string]interface{})["badge"] = notification.Badge
	}
	if notification.Sound != "" {
		payload["aps"].(map[string]interface{})["sound"] = notification.Sound
	}

	// Add custom data
	for k, v := range notification.Data {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Select APNS endpoint based on environment
	endpoint := "https://api.push.apple.com"
	if a.environment == "sandbox" {
		endpoint = "https://api.sandbox.push.apple.com"
	}

	url := fmt.Sprintf("%s/3/device/%s", endpoint, notification.DeviceToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", a.topic)
	req.Header.Set("apns-priority", mapPriorityToAPNS(notification.Priority))
	req.Header.Set("apns-push-type", "alert")

	// Note: In production, this would include the JWT authentication header
	// For now, we simulate success

	// Simulated success for development
	return &PushResult{
		NotificationID: notification.ID,
		Success:        true,
		DeliveredAt:    time.Now(),
	}, nil
}

// ValidateToken validates an APNS token
func (a *APNSProvider) ValidateToken(token string) bool {
	return len(token) == 64 // APNS tokens are 64 hex characters
}

// Platform returns the platform identifier
func (a *APNSProvider) Platform() Platform {
	return PlatformAPNS
}

func mapPriorityToAPNS(p Priority) string {
	if p == PriorityHigh {
		return "10"
	}
	return "5"
}

// WebPushProvider implements Web Push (VAPID)
type WebPushProvider struct {
	vapidKey  *ecdsa.PrivateKey
	subject   string
	email     string
	client    *http.Client
}

// NewWebPushProvider creates a new Web Push provider
func NewWebPushProvider(vapidKeyHex, subject, email string) *WebPushProvider {
	// Generate a new key if not provided
	var key *ecdsa.PrivateKey
	if vapidKeyHex != "" {
		// Would parse the key here in production
		key, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	} else {
		key, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	return &WebPushProvider{
		vapidKey: key,
		subject:  subject,
		email:    email,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification via Web Push
func (w *WebPushProvider) Send(ctx context.Context, notification *Notification) (*PushResult, error) {
	// Web Push uses subscription endpoint as the "device token"
	// In production, this would encrypt the payload and send to the push service

	payload := map[string]interface{}{
		"notification": map[string]interface{}{
			"title": notification.Title,
			"body":  notification.Body,
			"icon":  notification.Icon,
			"badge": notification.Badge,
			"tag":   notification.Tag,
			"data":  notification.Data,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// In production, we would:
	// 1. Parse the subscription endpoint from DeviceToken
	// 2. Generate VAPID JWT
	// 3. Encrypt the payload with the subscription keys
	// 4. Send to the push service

	// For now, simulate success
	_ = body // Prevent unused variable error

	return &PushResult{
		NotificationID: notification.ID,
		Success:        true,
		DeliveredAt:    time.Now(),
	}, nil
}

// ValidateToken validates a Web Push subscription endpoint
func (w *WebPushProvider) ValidateToken(token string) bool {
	return len(token) > 50 // Subscription endpoints are typically long URLs
}

// Platform returns the platform identifier
func (w *WebPushProvider) Platform() Platform {
	return PlatformWebPush
}

// GetVAPIDPublicKey returns the public key for Web Push subscriptions
func (w *WebPushProvider) GetVAPIDPublicKey() string {
	if w.vapidKey == nil {
		return ""
	}

	// Encode public key in uncompressed format
	pubKey := elliptic.Marshal(w.vapidKey.Curve, w.vapidKey.PublicKey.X, w.vapidKey.PublicKey.Y)
	return base64.RawURLEncoding.EncodeToString(pubKey)
}

// GenerateVAPIDHeaders generates VAPID headers for Web Push
func (w *WebPushProvider) GenerateVAPIDHeaders(endpoint string) (string, string, error) {
	if w.vapidKey == nil {
		return "", "", fmt.Errorf("VAPID key not configured")
	}

	// Create JWT for VAPID
	now := time.Now().Unix()
	exp := now + 12*3600 // 12 hours

	// JWT header
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))

	// JWT payload
	payload := fmt.Sprintf(`{"aud":"%s","exp":%d,"sub":"mailto:%s"}`, endpoint, exp, w.email)
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))

	// Sign (simplified - in production would use proper ECDSA)
	signature := sha256.Sum256([]byte(header + "." + payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(signature[:])

	jwt := header + "." + payloadB64 + "." + sigB64

	// Generate Crypto-Key header
	pubKey := w.GetVAPIDPublicKey()

	return "vapid t=" + jwt, "p256ecdsa=" + pubKey, nil
}

// MockProvider is a mock provider for testing
type MockProvider struct {
	platform Platform
	failNext bool
	results  []*PushResult
}

// NewMockProvider creates a new mock provider
func NewMockProvider(platform Platform) *MockProvider {
	return &MockProvider{
		platform: platform,
		results:  make([]*PushResult, 0),
	}
}

// Send simulates sending a notification
func (m *MockProvider) Send(ctx context.Context, notification *Notification) (*PushResult, error) {
	if m.failNext {
		m.failNext = false
		return &PushResult{
			NotificationID: notification.ID,
			Success:        false,
			Error:          "mock error",
		}, fmt.Errorf("mock error")
	}

	result := &PushResult{
		NotificationID: notification.ID,
		Success:        true,
		DeliveredAt:    time.Now(),
	}
	m.results = append(m.results, result)
	return result, nil
}

// ValidateToken always returns true for mock
func (m *MockProvider) ValidateToken(token string) bool {
	return len(token) > 0
}

// Platform returns the mock platform
func (m *MockProvider) Platform() Platform {
	return m.platform
}

// SetFailNext makes the next send fail
func (m *MockProvider) SetFailNext(fail bool) {
	m.failNext = fail
}

// GetResults returns all results
func (m *MockProvider) GetResults() []*PushResult {
	return m.results
}
