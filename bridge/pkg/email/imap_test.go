package email

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockConn struct {
	folders      []IMAPFolder
	foldersErr   error
	selectTotal  int
	selectErr    error
	summaries    []IMAPMessageSummary
	summariesErr error
	message      *IMAPMessage
	messageErr   error
	storeErr     error
	moveErr      error
	closed       bool
}

func (m *mockConn) ListFolders(_ context.Context) ([]IMAPFolder, error) {
	return m.folders, m.foldersErr
}

func (m *mockConn) SelectFolder(_ context.Context, _ string) (int, error) {
	return m.selectTotal, m.selectErr
}

func (m *mockConn) FetchSummaries(_ context.Context, _, _ int) ([]IMAPMessageSummary, error) {
	return m.summaries, m.summariesErr
}

func (m *mockConn) FetchMessage(_ context.Context, _ int) (*IMAPMessage, error) {
	return m.message, m.messageErr
}

func (m *mockConn) StoreFlags(_ context.Context, _ int, _ []string) error {
	return m.storeErr
}

func (m *mockConn) MoveMessage(_ context.Context, _ int, _ string) error {
	return m.moveErr
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func stubDialer(conn *mockConn) IMAPDialer {
	return func(_ context.Context, _, _, _ string) (IMAPConn, error) {
		return conn, nil
	}
}

func dialErr(err error) IMAPDialer {
	return func(_ context.Context, _, _, _ string) (IMAPConn, error) {
		return nil, err
	}
}

func validCfg(dialer IMAPDialer) IMAPClientConfig {
	return IMAPClientConfig{
		Dialer:   dialer,
		Addr:     "imap.example.com:993",
		Username: "user@example.com",
		Password: "s3cret",
	}
}

func TestNewIMAPClient_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  IMAPClientConfig
		want string
	}{
		{"no dialer", IMAPClientConfig{Addr: "a", Username: "u", Password: "p"}, "Dialer"},
		{"no addr", IMAPClientConfig{Dialer: func(_ context.Context, _, _, _ string) (IMAPConn, error) { return nil, nil }, Username: "u", Password: "p"}, "Addr"},
		{"no username", IMAPClientConfig{Dialer: func(_ context.Context, _, _, _ string) (IMAPConn, error) { return nil, nil }, Addr: "a", Password: "p"}, "Username"},
		{"no password", IMAPClientConfig{Dialer: func(_ context.Context, _, _, _ string) (IMAPConn, error) { return nil, nil }, Addr: "a", Username: "u"}, "Password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewIMAPClient(tt.cfg)
			if err == nil {
				t.Fatalf("expected error containing %q", tt.want)
			}
		})
	}
}

func TestListFolders_Success(t *testing.T) {
	mc := &mockConn{
		folders: []IMAPFolder{
			{Name: "INBOX", Delimiter: "/"},
			{Name: "Archive", Delimiter: "/"},
		},
	}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	folders, err := client.ListFolders(context.Background())
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}
	if folders[0].Name != "INBOX" {
		t.Errorf("first folder = %q, want INBOX", folders[0].Name)
	}
	if !mc.closed {
		t.Error("connection was not closed")
	}
}

func TestListFolders_DialError(t *testing.T) {
	client, _ := NewIMAPClient(validCfg(dialErr(errors.New("connection refused"))))

	_, err := client.ListFolders(context.Background())
	if err == nil {
		t.Fatal("expected error from dial failure")
	}
}

func TestListFolders_ListError(t *testing.T) {
	mc := &mockConn{foldersErr: errors.New("LIST failed")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	_, err := client.ListFolders(context.Background())
	if err == nil {
		t.Fatal("expected LIST error")
	}
}

func TestListMessages_Pagination(t *testing.T) {
	summaries := []IMAPMessageSummary{
		{UID: 3, Subject: "Third"},
		{UID: 2, Subject: "Second"},
		{UID: 1, Subject: "First"},
	}
	mc := &mockConn{
		selectTotal: 30,
		summaries:   summaries,
	}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	result, err := client.ListMessages(context.Background(), "INBOX", IMAPListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if result.Total != 30 {
		t.Errorf("Total = %d, want 30", result.Total)
	}
	if result.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", result.TotalPages)
	}
	if result.Page != 1 {
		t.Errorf("Page = %d, want 1", result.Page)
	}
	if len(result.Messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(result.Messages))
	}
}

func TestListMessages_Defaults(t *testing.T) {
	mc := &mockConn{selectTotal: 5, summaries: []IMAPMessageSummary{{UID: 1}}}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	result, err := client.ListMessages(context.Background(), "INBOX", IMAPListOptions{})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("default Page = %d, want 1", result.Page)
	}
	if result.PageSize != 25 {
		t.Errorf("default PageSize = %d, want 25", result.PageSize)
	}
}

func TestListMessages_EmptyFolder(t *testing.T) {
	mc := &mockConn{selectTotal: 0}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	result, err := client.ListMessages(context.Background(), "INBOX", IMAPListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(result.Messages))
	}
}

func TestListMessages_PageBeyondTotal(t *testing.T) {
	mc := &mockConn{selectTotal: 5}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	result, err := client.ListMessages(context.Background(), "INBOX", IMAPListOptions{Page: 10, PageSize: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected empty for page beyond total, got %d", len(result.Messages))
	}
}

func TestListMessages_SelectError(t *testing.T) {
	mc := &mockConn{selectErr: errors.New("no such mailbox")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	_, err := client.ListMessages(context.Background(), "MISSING", IMAPListOptions{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("expected select error")
	}
}

func TestListMessages_FetchError(t *testing.T) {
	mc := &mockConn{selectTotal: 5, summariesErr: errors.New("fetch failed")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	_, err := client.ListMessages(context.Background(), "INBOX", IMAPListOptions{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestReadMessage_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	msg := &IMAPMessage{
		UID:      42,
		Subject:  "Test Subject",
		From:     EmailAddress{Address: "alice@example.com", Name: "Alice"},
		To:       []EmailAddress{{Address: "bob@example.com"}},
		Date:     now,
		BodyText: "Hello, World!",
	}
	mc := &mockConn{selectTotal: 1, message: msg}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	got, err := client.ReadMessage(context.Background(), "INBOX", 42)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got.UID != 42 {
		t.Errorf("UID = %d, want 42", got.UID)
	}
	if got.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Test Subject")
	}
	if got.BodyText != "Hello, World!" {
		t.Errorf("BodyText = %q, want %q", got.BodyText, "Hello, World!")
	}
}

func TestReadMessage_SelectError(t *testing.T) {
	mc := &mockConn{selectErr: errors.New("mailbox not found")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	_, err := client.ReadMessage(context.Background(), "MISSING", 1)
	if err == nil {
		t.Fatal("expected select error")
	}
}

func TestReadMessage_FetchError(t *testing.T) {
	mc := &mockConn{selectTotal: 1, messageErr: errors.New("no such message")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	_, err := client.ReadMessage(context.Background(), "INBOX", 9999)
	if err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestArchive_Success(t *testing.T) {
	mc := &mockConn{selectTotal: 10}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	if err := client.Archive(context.Background(), "INBOX", 5, "Archive"); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !mc.closed {
		t.Error("connection was not closed")
	}
}

func TestArchive_SelectError(t *testing.T) {
	mc := &mockConn{selectErr: errors.New("not found")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	err := client.Archive(context.Background(), "INBOX", 5, "Archive")
	if err == nil {
		t.Fatal("expected select error")
	}
}

func TestArchive_MoveError(t *testing.T) {
	mc := &mockConn{selectTotal: 10, moveErr: errors.New("cannot move")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	err := client.Archive(context.Background(), "INBOX", 5, "Archive")
	if err == nil {
		t.Fatal("expected move error")
	}
}

func TestMarkRead_Success(t *testing.T) {
	mc := &mockConn{selectTotal: 10}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	if err := client.MarkRead(context.Background(), "INBOX", 7); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if !mc.closed {
		t.Error("connection was not closed")
	}
}

func TestMarkRead_StoreError(t *testing.T) {
	mc := &mockConn{selectTotal: 10, storeErr: errors.New("STORE failed")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	err := client.MarkRead(context.Background(), "INBOX", 7)
	if err == nil {
		t.Fatal("expected store error")
	}
}

func TestMarkRead_SelectError(t *testing.T) {
	mc := &mockConn{selectErr: errors.New("mailbox not found")}
	client, _ := NewIMAPClient(validCfg(stubDialer(mc)))

	err := client.MarkRead(context.Background(), "MISSING", 1)
	if err == nil {
		t.Fatal("expected select error")
	}
}

func TestDialError_AllMethods(t *testing.T) {
	client, _ := NewIMAPClient(validCfg(dialErr(errors.New("auth failed"))))

	ctx := context.Background()

	if _, err := client.ListFolders(ctx); err == nil {
		t.Error("ListFolders should fail on dial error")
	}
	if _, err := client.ListMessages(ctx, "INBOX", IMAPListOptions{}); err == nil {
		t.Error("ListMessages should fail on dial error")
	}
	if _, err := client.ReadMessage(ctx, "INBOX", 1); err == nil {
		t.Error("ReadMessage should fail on dial error")
	}
	if err := client.Archive(ctx, "INBOX", 1, "Archive"); err == nil {
		t.Error("Archive should fail on dial error")
	}
	if err := client.MarkRead(ctx, "INBOX", 1); err == nil {
		t.Error("MarkRead should fail on dial error")
	}
}

func TestNewIMAPClient_Success(t *testing.T) {
	cfg := validCfg(stubDialer(&mockConn{}))
	client, err := NewIMAPClient(cfg)
	if err != nil {
		t.Fatalf("NewIMAPClient: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.addr != cfg.Addr {
		t.Errorf("addr = %q, want %q", client.addr, cfg.Addr)
	}
	if client.username != cfg.Username {
		t.Errorf("username = %q, want %q", client.username, cfg.Username)
	}
}
