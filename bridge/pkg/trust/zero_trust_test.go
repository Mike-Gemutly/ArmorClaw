package trust

import (
	"context"
	"testing"
	"time"
)

func TestNewZeroTrustManager(t *testing.T) {
	config := ZeroTrustConfig{
		MinimumTrustLevel: TrustScoreMedium,
	}

	manager := NewZeroTrustManager(config)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.config.MinimumTrustLevel != TrustScoreMedium {
		t.Errorf("Expected medium trust level, got %v", manager.config.MinimumTrustLevel)
	}
}

func TestTrustScoreString(t *testing.T) {
	tests := []struct {
		level    TrustScore
		expected string
	}{
		{TrustScoreUntrusted, "untrusted"},
		{TrustScoreLow, "low"},
		{TrustScoreMedium, "medium"},
		{TrustScoreHigh, "high"},
		{TrustScoreVerified, "verified"},
		{TrustScore(99), "unknown"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestVerifyNewDevice(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{
		MinimumTrustLevel: TrustScoreMedium,
	})

	req := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	result, err := manager.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if result.TrustLevel >= TrustScoreHigh {
		t.Errorf("New device should not have high trust, got %v", result.TrustLevel)
	}

	if result.DeviceID == "" {
		t.Error("Expected device ID to be set")
	}

	if result.SessionID == "" {
		t.Error("Expected session ID to be set")
	}
}

func TestVerifyDeviceVerification(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{
		MinimumTrustLevel: TrustScoreMedium,
	})

	req := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	result, _ := manager.Verify(context.Background(), req)

	err := manager.VerifyDevice(result.DeviceID, "email")
	if err != nil {
		t.Fatalf("Failed to verify device: %v", err)
	}

	result2, err := manager.Verify(context.Background(), &ZeroTrustRequest{
		SessionID: "session-2",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	})
	if err != nil {
		t.Fatalf("Second verification failed: %v", err)
	}

	if result2.TrustLevel < TrustScoreHigh {
		t.Errorf("Verified device should have high trust, got %v", result2.TrustLevel)
	}
}

func TestVerifyLockout(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{
		MinimumTrustLevel:       TrustScoreVerified,
		MaxVerificationAttempts: 3,
		LockoutDuration:         5 * time.Minute,
	})

	req := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	for i := 0; i < 4; i++ {
		result, err := manager.Verify(context.Background(), req)
		if err != nil {
			t.Fatalf("Verification %d failed: %v", i, err)
		}

		if i < 3 && result.Passed {
			t.Errorf("Should not pass with high trust requirement")
		}

		if i >= 3 {
			session, _ := manager.GetTrustedSession(req.SessionID)
			if session != nil && !session.LockedOut {
				t.Error("Expected session to be locked out")
			}
		}
	}
}

func TestRiskScoreCalculation(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	req := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	result, _ := manager.Verify(context.Background(), req)

	if result.RiskScore <= 0 {
		t.Errorf("New device should have risk score > 0, got %d", result.RiskScore)
	}
}

func TestAnomalyDetection(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	req1 := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	manager.Verify(context.Background(), req1)

	req2 := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "10.0.0.1",
		Action:    "access",
		Resource:  "data",
	}

	result, _ := manager.Verify(context.Background(), req2)

	anomalyDetected := false
	for _, flag := range result.AnomalyFlags {
		if flag == "ip_change" {
			anomalyDetected = true
			break
		}
	}

	if !anomalyDetected {
		t.Error("Expected ip_change anomaly to be detected")
	}
}

func TestGetUserTrustedDevices(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	// Create 3 unique devices with completely different fingerprints
	fingerprints := []DeviceFingerprintInput{
		{UserAgent: "Mozilla/5.0 (Windows)", Platform: "windows", CanvasHash: "unique1", ScreenRes: "1920x1080"},
		{UserAgent: "Mozilla/5.0 (Macintosh)", Platform: "macos", CanvasHash: "unique2", ScreenRes: "2560x1440"},
		{UserAgent: "Mozilla/5.0 (Linux)", Platform: "linux", CanvasHash: "unique3", ScreenRes: "3840x2160"},
	}

	for i, fp := range fingerprints {
		manager.Verify(context.Background(), &ZeroTrustRequest{
			SessionID:         "session-" + string(rune('1'+i)),
			UserID:            "user-1",
			DeviceFingerprint: fp,
			IPAddress:         "192.168.1.1",
			Action:            "login",
			Resource:          "session",
		})
	}

	devices := manager.GetUserTrustedDevices("user-1")
	if len(devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(devices))
	}

	devices = manager.GetUserTrustedDevices("user-2")
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices for user-2, got %d", len(devices))
	}
}

func TestRevokeTrustDevice(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	result, _ := manager.Verify(context.Background(), &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	})

	manager.VerifyDevice(result.DeviceID, "email")

	device, _ := manager.GetDeviceData(result.DeviceID)
	if !device.Verified {
		t.Error("Expected device to be verified")
	}

	err := manager.RevokeTrustDevice(result.DeviceID)
	if err != nil {
		t.Fatalf("Failed to revoke device: %v", err)
	}

	device, _ = manager.GetDeviceData(result.DeviceID)
	if device.Verified {
		t.Error("Expected device to be revoked")
	}
	if device.TrustLevel != TrustScoreLow {
		t.Errorf("Expected low trust level after revocation, got %v", device.TrustLevel)
	}
}

func TestInvalidateTrustedSession(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	manager.Verify(context.Background(), &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	})

	_, err := manager.GetTrustedSession("session-1")
	if err != nil {
		t.Fatalf("Session should exist: %v", err)
	}

	err = manager.InvalidateTrustedSession("session-1")
	if err != nil {
		t.Fatalf("Failed to invalidate session: %v", err)
	}

	_, err = manager.GetTrustedSession("session-1")
	if err == nil {
		t.Error("Expected session to be invalidated")
	}
}

func TestCleanupExpired(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{
		DeviceFingerprintTTL: time.Hour,
	})

	manager.Verify(context.Background(), &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	})

	manager.mu.Lock()
	if session, exists := manager.sessions["session-1"]; exists {
		session.ExpiresAt = time.Now().Add(-time.Hour)
	}
	manager.mu.Unlock()

	cleaned := manager.CleanupExpired()
	if cleaned < 1 {
		t.Errorf("Expected at least 1 cleaned item, got %d", cleaned)
	}
}

func TestGetZeroTrustStats(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{
		MinimumTrustLevel: TrustScoreMedium,
	})

	// Create 3 unique devices with completely different fingerprints
	fingerprints := []DeviceFingerprintInput{
		{UserAgent: "Mozilla/5.0 (Windows)", Platform: "windows", CanvasHash: "stats1", ScreenRes: "1920x1080"},
		{UserAgent: "Mozilla/5.0 (Macintosh)", Platform: "macos", CanvasHash: "stats2", ScreenRes: "2560x1440"},
		{UserAgent: "Mozilla/5.0 (Linux)", Platform: "linux", CanvasHash: "stats3", ScreenRes: "3840x2160"},
	}

	for i, fp := range fingerprints {
		manager.Verify(context.Background(), &ZeroTrustRequest{
			SessionID:         "session-" + string(rune('1'+i)),
			UserID:            "user-1",
			DeviceFingerprint: fp,
			IPAddress:         "192.168.1.1",
			Action:            "login",
			Resource:          "session",
		})
	}

	devices := manager.GetUserTrustedDevices("user-1")
	if len(devices) > 0 {
		manager.VerifyDevice(devices[0].ID, "email")
	}

	stats := manager.GetZeroTrustStats()

	if stats["total_devices"].(int) != 3 {
		t.Errorf("Expected 3 devices, got %v", stats["total_devices"])
	}

	if stats["verified_devices"].(int) != 1 {
		t.Errorf("Expected 1 verified device, got %v", stats["verified_devices"])
	}

	if stats["minimum_trust_level"].(string) != "medium" {
		t.Errorf("Expected medium minimum trust, got %v", stats["minimum_trust_level"])
	}
}

func TestCheckPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"invalid", false},
	}

	for _, test := range tests {
		result := CheckPrivateIP(test.ip)
		if result != test.expected {
			t.Errorf("CheckPrivateIP(%s) = %v, expected %v", test.ip, result, test.expected)
		}
	}
}

func TestCustomVerifier(t *testing.T) {
	manager := NewZeroTrustManager(ZeroTrustConfig{})

	manager.AddVerifier(&mockZeroTrustVerifier{
		name:         "test_verifier",
		trustLevel:   TrustScoreHigh,
		riskScore:    10,
		anomalyFlags: []string{"custom_check"},
	})

	req := &ZeroTrustRequest{
		SessionID: "session-1",
		UserID:    "user-1",
		DeviceFingerprint: DeviceFingerprintInput{
			UserAgent: "Mozilla/5.0",
			Platform:  "windows",
		},
		IPAddress: "192.168.1.1",
		Action:    "login",
		Resource:  "session",
	}

	result, err := manager.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	customAnomalyFound := false
	for _, flag := range result.AnomalyFlags {
		if flag == "custom_check" {
			customAnomalyFound = true
			break
		}
	}

	if !customAnomalyFound {
		t.Error("Expected custom verifier anomaly flag")
	}
}

// Mock verifier for testing
type mockZeroTrustVerifier struct {
	name         string
	trustLevel   TrustScore
	riskScore    int
	anomalyFlags []string
}

func (m *mockZeroTrustVerifier) Name() string {
	return m.name
}

func (m *mockZeroTrustVerifier) Verify(ctx context.Context, req *ZeroTrustRequest) (TrustScore, int, []string, error) {
	return m.trustLevel, m.riskScore, m.anomalyFlags, nil
}
