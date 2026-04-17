package email

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/armorclaw/bridge/pkg/keystore"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const gmailProvider = "gmail"

type GmailClient struct {
	keystore *keystore.Keystore
	from     string
}

type GmailClientConfig struct {
	Keystore *keystore.Keystore
	From     string
}

func NewGmailClient(cfg GmailClientConfig) *GmailClient {
	return &GmailClient{
		keystore: cfg.Keystore,
		from:     cfg.From,
	}
}

func (g *GmailClient) Send(ctx context.Context, to, subject, bodyText, bodyHTML string, attachments ...*EmailAttachment) (string, error) {
	if _, err := mail.ParseAddress(to); err != nil {
		return "", fmt.Errorf("invalid recipient: %w", err)
	}

	token, err := g.getOAuthToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get oauth token: %w", err)
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		RefreshToken: token.RefreshToken,
		TokenType:    "Bearer",
	})

	svc, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return "", fmt.Errorf("create gmail service: %w", err)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", g.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", subject))
	if bodyHTML != "" {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("\r\n")
	if bodyHTML != "" {
		msg.WriteString(bodyHTML)
	} else {
		msg.WriteString(bodyText)
	}

	gmsg := &gmail.Message{
		Raw: encodeBase64URL(msg.String()),
	}

	result, err := svc.Users.Messages.Send("me", gmsg).Do()
	if err != nil {
		return "", fmt.Errorf("gmail send: %w", err)
	}

	return result.Id, nil
}

func (g *GmailClient) Provider() string {
	return gmailProvider
}

func (g *GmailClient) getOAuthToken(ctx context.Context) (*keystore.OAuthTokenRecord, error) {
	rec, err := g.keystore.GetOAuthRefreshToken(ctx, gmailProvider)
	if err != nil {
		return nil, fmt.Errorf("get oauth token from keystore: %w", err)
	}
	if rec == nil {
		return nil, fmt.Errorf("no gmail oauth token found — run OAuth2 authorization flow first")
	}
	return rec, nil
}

func (g *GmailClient) createOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailSendScope},
	}
}

func encodeBase64URL(s string) string {
	raw := []byte(s)
	var encoded strings.Builder
	b64 := make([]byte, 4)
	for i := 0; i < len(raw); i += 3 {
		n := len(raw) - i
		if n > 3 {
			n = 3
		}
		val := uint32(raw[i]) << 16
		if n > 1 {
			val |= uint32(raw[i+1]) << 8
		}
		if n > 2 {
			val |= uint32(raw[i+2])
		}
		b64[0] = encodeByte((val >> 18) & 0x3F)
		b64[1] = encodeByte((val >> 12) & 0x3F)
		b64[2] = '='
		b64[3] = '='
		if n > 1 {
			b64[2] = encodeByte((val >> 6) & 0x3F)
		}
		if n > 2 {
			b64[3] = encodeByte(val & 0x3F)
		}
		encoded.Write(b64[:4])
	}
	return encoded.String()
}

func encodeByte(b uint32) byte {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	return table[b]
}

var _ EmailSender = (*GmailClient)(nil)
var _ = http.DefaultClient
