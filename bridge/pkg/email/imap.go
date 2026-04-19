package email

import (
	"context"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

// IMAPFolder represents a mailbox folder on the IMAP server.
type IMAPFolder struct {
	Name       string `json:"name"`
	Delimiter  string `json:"delimiter"`
	Attributes string `json:"attributes,omitempty"`
}

// IMAPMessageSummary is a lightweight envelope for folder message listings.
type IMAPMessageSummary struct {
	UID     int            `json:"uid"`
	Subject string         `json:"subject"`
	From    EmailAddress   `json:"from"`
	To      []EmailAddress `json:"to"`
	Date    time.Time      `json:"date"`
	Flags   []string       `json:"flags,omitempty"`
	Size    int64          `json:"size,omitempty"`
}

// IMAPMessage is the full message including body and attachments.
type IMAPMessage struct {
	UID         int               `json:"uid"`
	Subject     string            `json:"subject"`
	From        EmailAddress      `json:"from"`
	To          []EmailAddress    `json:"to"`
	CC          []EmailAddress    `json:"cc,omitempty"`
	Date        time.Time         `json:"date"`
	Flags       []string          `json:"flags,omitempty"`
	BodyText    string            `json:"body_text,omitempty"`
	BodyHTML    string            `json:"body_html,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// IMAPListOptions controls pagination and filtering for message listings.
type IMAPListOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Unseen   bool   `json:"unseen,omitempty"`
	Since    string `json:"since,omitempty"`
}

// IMAPListResult holds a page of message summaries.
type IMAPListResult struct {
	Messages   []IMAPMessageSummary `json:"messages"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

// IMAPConn abstracts the underlying IMAP connection for mock injection.
type IMAPConn interface {
	ListFolders(ctx context.Context) ([]IMAPFolder, error)
	SelectFolder(ctx context.Context, name string) (total int, err error)
	FetchSummaries(ctx context.Context, seqStart, seqEnd int) ([]IMAPMessageSummary, error)
	FetchMessage(ctx context.Context, uid int) (*IMAPMessage, error)
	StoreFlags(ctx context.Context, uid int, flags []string) error
	MoveMessage(ctx context.Context, uid int, destFolder string) error
	Close() error
}

// IMAPDialer opens an IMAP connection. Defined as a function type so tests
// can inject a mock without importing a real IMAP library.
type IMAPDialer func(ctx context.Context, addr, username, password string) (IMAPConn, error)

// IMAPClientConfig holds constructor parameters for IMAPClient.
type IMAPClientConfig struct {
	Dialer   IMAPDialer
	Log      *logger.Logger
	Addr     string
	Username string
	Password string
}

// IMAPClient provides inbox management operations over IMAP.
// All methods accept context.Context as the first argument.
type IMAPClient struct {
	dialer   IMAPDialer
	log      *logger.Logger
	addr     string
	username string
	password string
}

// NewIMAPClient constructs an IMAPClient. It returns an error if required
// fields are missing.
func NewIMAPClient(cfg IMAPClientConfig) (*IMAPClient, error) {
	if cfg.Dialer == nil {
		return nil, fmt.Errorf("email: IMAPClient requires a Dialer")
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("email: IMAPClient requires Addr")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("email: IMAPClient requires Username")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("email: IMAPClient requires Password")
	}
	return &IMAPClient{
		dialer:   cfg.Dialer,
		log:      cfg.Log,
		addr:     cfg.Addr,
		username: cfg.Username,
		password: cfg.Password,
	}, nil
}

// dial opens a connection via the injected dialer.
func (c *IMAPClient) dial(ctx context.Context) (IMAPConn, error) {
	conn, err := c.dialer(ctx, c.addr, c.username, c.password)
	if err != nil {
		return nil, fmt.Errorf("imap dial: %w", err)
	}
	return conn, nil
}

func (c *IMAPClient) logf() *logger.Logger {
	if c.log != nil {
		return c.log
	}
	l, _ := logger.New(logger.Config{Level: "error", Output: "/dev/null"})
	return l
}

// ListFolders returns all mailbox folders on the server.
func (c *IMAPClient) ListFolders(ctx context.Context) ([]IMAPFolder, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	folders, err := conn.ListFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("imap list folders: %w", err)
	}
	return folders, nil
}

// ListMessages returns a paginated listing of messages in the specified folder.
func (c *IMAPClient) ListMessages(ctx context.Context, folder string, opts IMAPListOptions) (*IMAPListResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 25
	}

	total, err := conn.SelectFolder(ctx, folder)
	if err != nil {
		return nil, fmt.Errorf("imap select folder %q: %w", folder, err)
	}

	totalPages := total / opts.PageSize
	if total%opts.PageSize != 0 {
		totalPages++
	}

	end := total - (opts.Page-1)*opts.PageSize
	start := end - opts.PageSize + 1
	if start < 1 {
		start = 1
	}
	if end < 1 {
		return &IMAPListResult{
			Messages:   []IMAPMessageSummary{},
			Total:      total,
			Page:       opts.Page,
			PageSize:   opts.PageSize,
			TotalPages: totalPages,
		}, nil
	}

	msgs, err := conn.FetchSummaries(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("imap fetch summaries: %w", err)
	}

	return &IMAPListResult{
		Messages:   msgs,
		Total:      total,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ReadMessage fetches the full message (body + attachments) by UID.
func (c *IMAPClient) ReadMessage(ctx context.Context, folder string, uid int) (*IMAPMessage, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.SelectFolder(ctx, folder); err != nil {
		return nil, fmt.Errorf("imap select folder %q: %w", folder, err)
	}

	msg, err := conn.FetchMessage(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("imap fetch uid %d: %w", uid, err)
	}
	return msg, nil
}

// Archive moves a message from srcFolder to the archive destination.
func (c *IMAPClient) Archive(ctx context.Context, srcFolder string, uid int, archiveFolder string) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.SelectFolder(ctx, srcFolder); err != nil {
		return fmt.Errorf("imap select folder %q: %w", srcFolder, err)
	}

	if err := conn.MoveMessage(ctx, uid, archiveFolder); err != nil {
		return fmt.Errorf("imap archive uid %d to %q: %w", uid, archiveFolder, err)
	}

	c.logf().Info("imap_archived", "uid", uid, "src", srcFolder, "dest", archiveFolder)
	return nil
}

// MarkRead sets the \\Seen flag on a message.
func (c *IMAPClient) MarkRead(ctx context.Context, folder string, uid int) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.SelectFolder(ctx, folder); err != nil {
		return fmt.Errorf("imap select folder %q: %w", folder, err)
	}

	if err := conn.StoreFlags(ctx, uid, []string{"\\Seen"}); err != nil {
		return fmt.Errorf("imap mark read uid %d: %w", uid, err)
	}

	c.logf().Info("imap_marked_read", "uid", uid, "folder", folder)
	return nil
}
