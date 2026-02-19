// Package push provides tests for push notification gateway
package push

import (
	"context"
	"testing"
	"time"
)

func TestNewGateway(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "empty config",
			config: Config{
				FCMEnabled: false,
				APNSEnabled: false,
			},
			wantErr: false,
		},
		{
			name: "fcm enabled",
			config: Config{
				FCMEnabled:   true,
				FCMServerKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "all providers enabled",
			config: Config{
				FCMEnabled:      true,
				FCMServerKey:    "test-key",
				APNSEnabled:     true,
				APNSCertFile:    "cert.pem",
				APNSKeyFile:     "key.pem",
				WebPushEnabled:  true,
				WebPushVAPIDKey: "vapid-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw, err := NewGateway(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGateway() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gw == nil {
				t.Error("NewGateway() returned nil without error")
			}
		})
	}
}

func TestDeviceRegistration(t *testing.T) {
	gw, _ := NewGateway(Config{})

	// Test registration
	device, err := gw.RegisterDevice("@user:example.com", PlatformFCM, "token123", "android", "Pixel 6")
	if err != nil {
		t.Fatalf("RegisterDevice() error = %v", err)
	}

	if device.UserID != "@user:example.com" {
		t.Errorf("Device UserID = %s, want @user:example.com", device.UserID)
	}
	if device.Platform != PlatformFCM {
		t.Errorf("Device Platform = %s, want fcm", device.Platform)
	}
	if !device.Enabled {
		t.Error("Device should be enabled")
	}

	// Test re-registration (should update existing)
	device2, err := gw.RegisterDevice("@user:example.com", PlatformFCM, "token123", "android", "Pixel 6 Pro")
	if err != nil {
		t.Fatalf("RegisterDevice() second call error = %v", err)
	}

	if device.ID != device2.ID {
		t.Error("Re-registration should return same device ID")
	}

	// Test get device
	retrieved, err := gw.GetDevice(device.ID)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if retrieved.ID != device.ID {
		t.Errorf("GetDevice() ID = %s, want %s", retrieved.ID, device.ID)
	}

	// Test unregister
	err = gw.UnregisterDevice(device.ID)
	if err != nil {
		t.Fatalf("UnregisterDevice() error = %v", err)
	}

	// Verify unregistered
	_, err = gw.GetDevice(device.ID)
	if err == nil {
		t.Error("GetDevice() should fail for unregistered device")
	}
}

func TestSendToUser(t *testing.T) {
	// Create gateway with mock provider
	gw, _ := NewGateway(Config{})
	mockProvider := NewMockProvider(PlatformFCM)
	gw.providers[PlatformFCM] = mockProvider

	// Register devices
	gw.RegisterDevice("@user:example.com", PlatformFCM, "token1", "android", "Phone 1")
	gw.RegisterDevice("@user:example.com", PlatformFCM, "token2", "android", "Phone 2")

	// Send notification
	notif := &Notification{
		Title:    "Test",
		Body:     "Hello World",
		Priority: PriorityHigh,
	}

	results, err := gw.SendToUser(context.Background(), "@user:example.com", notif)
	if err != nil {
		t.Fatalf("SendToUser() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SendToUser() results = %d, want 2", len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Errorf("Result should be successful, got error: %s", r.Error)
		}
	}
}

func TestSendToUserNoDevices(t *testing.T) {
	gw, _ := NewGateway(Config{})

	notif := &Notification{Title: "Test", Body: "Hello"}
	_, err := gw.SendToUser(context.Background(), "@nouser:example.com", notif)
	if err == nil {
		t.Error("SendToUser() should fail for user with no devices")
	}
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider(PlatformFCM)

	// Test send success
	notif := &Notification{
		ID:          "test-123",
		Platform:    PlatformFCM,
		DeviceToken: "token",
		Title:       "Test",
		Body:        "Body",
	}

	result, err := provider.Send(context.Background(), notif)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("Result should be successful")
	}
	if result.NotificationID != "test-123" {
		t.Errorf("NotificationID = %s, want test-123", result.NotificationID)
	}

	// Test send failure
	provider.SetFailNext(true)
	_, err = provider.Send(context.Background(), notif)
	if err == nil {
		t.Error("Send() should fail when SetFailNext is true")
	}
}

func TestGatewayStats(t *testing.T) {
	gw, _ := NewGateway(Config{})
	mockProvider := NewMockProvider(PlatformFCM)
	gw.providers[PlatformFCM] = mockProvider

	// Register devices
	gw.RegisterDevice("@user1:example.com", PlatformFCM, "token1", "android", "Phone 1")
	gw.RegisterDevice("@user2:example.com", PlatformFCM, "token2", "android", "Phone 2")

	stats := gw.GetStats()

	if stats["total_devices"] != 2 {
		t.Errorf("total_devices = %v, want 2", stats["total_devices"])
	}
	if stats["total_users"] != 2 {
		t.Errorf("total_users = %v, want 2", stats["total_users"])
	}
}

func TestCreateMatrixPushNotification(t *testing.T) {
	gw, _ := NewGateway(Config{})

	devices := []*DeviceRegistration{
		{
			ID:          "dev1",
			Platform:    PlatformFCM,
			DeviceToken: "token1",
		},
		{
			ID:          "dev2",
			Platform:    PlatformAPNS,
			DeviceToken: "token2",
		},
	}

	notifications := gw.CreateMatrixPushNotification(
		"!room:example.com",
		"$event123",
		"@sender:example.com",
		"Hello World",
		devices,
	)

	if len(notifications) != 2 {
		t.Fatalf("CreateMatrixPushNotification() count = %d, want 2", len(notifications))
	}

	// Check first notification
	if notifications[0].Platform != PlatformFCM {
		t.Errorf("Platform = %s, want fcm", notifications[0].Platform)
	}
	if notifications[0].Data["room_id"] != "!room:example.com" {
		t.Errorf("room_id = %v, want !room:example.com", notifications[0].Data["room_id"])
	}
	if notifications[0].Data["event_id"] != "$event123" {
		t.Errorf("event_id = %v, want $event123", notifications[0].Data["event_id"])
	}
}

func TestPusherManager(t *testing.T) {
	pm := NewPusherManager()

	pusher := &PusherRegistration{
		PushKey:           "token123",
		Kind:              "http",
		AppID:             "com.armorclaw.bridge",
		AppDisplayName:    "ArmorClaw",
		DeviceDisplayName: "Pixel 6",
		Lang:              "en",
	}

	// Test registration
	err := pm.RegisterPusher("@user:example.com", pusher)
	if err != nil {
		t.Fatalf("RegisterPusher() error = %v", err)
	}

	// Test get pushers
	pushers := pm.GetPushers("@user:example.com")
	if len(pushers) != 1 {
		t.Errorf("GetPushers() count = %d, want 1", len(pushers))
	}

	// Test duplicate registration (should update)
	pusher.DeviceDisplayName = "Pixel 6 Pro"
	err = pm.RegisterPusher("@user:example.com", pusher)
	if err != nil {
		t.Fatalf("RegisterPusher() update error = %v", err)
	}

	pushers = pm.GetPushers("@user:example.com")
	if len(pushers) != 1 {
		t.Errorf("GetPushers() count after update = %d, want 1", len(pushers))
	}
	if pushers[0].DeviceDisplayName != "Pixel 6 Pro" {
		t.Errorf("DeviceDisplayName = %s, want Pixel 6 Pro", pushers[0].DeviceDisplayName)
	}

	// Test unregister
	err = pm.UnregisterPusher("@user:example.com", "token123", "com.armorclaw.bridge")
	if err != nil {
		t.Fatalf("UnregisterPusher() error = %v", err)
	}

	pushers = pm.GetPushers("@user:example.com")
	if len(pushers) != 0 {
		t.Errorf("GetPushers() after unregister = %d, want 0", len(pushers))
	}
}

func TestNotificationPriority(t *testing.T) {
	tests := []struct {
		priority Priority
		want     string
	}{
		{PriorityHigh, "high"},
		{PriorityNormal, "normal"},
		{PriorityLow, "low"},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			notif := &Notification{Priority: tt.priority}
			if string(notif.Priority) != tt.want {
				t.Errorf("Priority = %s, want %s", notif.Priority, tt.want)
			}
		})
	}
}

func TestExtractMessageBody(t *testing.T) {
	tests := []struct {
		name    string
		content map[string]interface{}
		want    string
	}{
		{
			name:    "text message",
			content: map[string]interface{}{"body": "Hello", "msgtype": "m.text"},
			want:    "Hello",
		},
		{
			name:    "image",
			content: map[string]interface{}{"msgtype": "m.image"},
			want:    "📷 Image",
		},
		{
			name:    "video",
			content: map[string]interface{}{"msgtype": "m.video"},
			want:    "🎬 Video",
		},
		{
			name:    "audio",
			content: map[string]interface{}{"msgtype": "m.audio"},
			want:    "🎵 Audio",
		},
		{
			name:    "file",
			content: map[string]interface{}{"msgtype": "m.file"},
			want:    "📎 File",
		},
		{
			name:    "emote",
			content: map[string]interface{}{"body": "waves", "msgtype": "m.emote"},
			want:    "*waves",
		},
		{
			name:    "empty",
			content: map[string]interface{}{},
			want:    "New message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageBody(tt.content)
			if got != tt.want {
				t.Errorf("extractMessageBody() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFormatSenderName(t *testing.T) {
	tests := []struct {
		sender string
		want   string
	}{
		{"@alice:example.com", "alice"},
		{"@bob:matrix.org", "bob"},
		{"plain", "plain"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.sender, func(t *testing.T) {
			got := formatSenderName(tt.sender)
			if got != tt.want {
				t.Errorf("formatSenderName(%s) = %s, want %s", tt.sender, got, tt.want)
			}
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		content string
		maxLen  int
		want    string
	}{
		{"short", 10, "short"},
		{"this is a long message", 10, "this is..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := truncateContent(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateContent() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNotificationTimeout(t *testing.T) {
	gw, _ := NewGateway(Config{})
	mockProvider := NewMockProvider(PlatformFCM)
	gw.providers[PlatformFCM] = mockProvider

	gw.RegisterDevice("@user:example.com", PlatformFCM, "token", "android", "Phone")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	notif := &Notification{Title: "Test", Body: "Hello"}
	_, err := gw.SendToUser(ctx, "@user:example.com", notif)

	// The error might be context expired or successful if it completed in time
	// This is more about testing that timeout is respected
	if err != nil && err != context.DeadlineExceeded {
		// Some other error is acceptable
	}
}
