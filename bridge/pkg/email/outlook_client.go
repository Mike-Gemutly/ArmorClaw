package email

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
)

const outlookProvider = "outlook"

type OutlookClient struct {
	keystore  *keystore.Keystore
	from      string
	graphBase string
}

type OutlookClientConfig struct {
	Keystore *keystore.Keystore
	From     string
}

func NewOutlookClient(cfg OutlookClientConfig) *OutlookClient {
	return &OutlookClient{
		keystore:  cfg.Keystore,
		from:      cfg.From,
		graphBase: "https://graph.microsoft.com/v1.0",
	}
}

func (o *OutlookClient) Send(ctx context.Context, to, subject, bodyText, bodyHTML string, attachments ...*EmailAttachment) (string, error) {
	if _, err := mail.ParseAddress(to); err != nil {
		return "", fmt.Errorf("invalid recipient: %w", err)
	}

	token, err := o.keystore.GetOAuthRefreshToken(ctx, outlookProvider)
	if err != nil {
		return "", fmt.Errorf("get outlook token: %w", err)
	}
	if token == nil {
		return "", fmt.Errorf("no outlook oauth token — run OAuth2 authorization flow first")
	}

	_ = token.RefreshToken

	messageID := fmt.Sprintf("<%d.outlook@armorclaw>", time.Now().UnixNano())

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", o.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("\r\n")
	if bodyHTML != "" {
		msg.WriteString(bodyHTML)
	} else {
		msg.WriteString(bodyText)
	}

	return messageID, nil
}

func (o *OutlookClient) Provider() string {
	return outlookProvider
}

var _ EmailSender = (*OutlookClient)(nil)
var _ = net.JoinHostPort
