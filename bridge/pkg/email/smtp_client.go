package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const smtpProvider = "smtp"

type SMTPClient struct {
	host     string
	port     string
	username string
	password string
	from     string
	useTLS   bool
}

type SMTPClientConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	UseTLS   bool
}

func NewSMTPClient(cfg SMTPClientConfig) *SMTPClient {
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	if cfg.UseTLS == false && cfg.Port == "587" {
		cfg.UseTLS = true
	}
	return &SMTPClient{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
		useTLS:   cfg.UseTLS,
	}
}

func (s *SMTPClient) Send(ctx context.Context, to, subject, bodyText, bodyHTML string, attachments ...*EmailAttachment) (string, error) {
	if _, err := mail.ParseAddress(to); err != nil {
		return "", fmt.Errorf("invalid recipient: %w", err)
	}

	addr := net.JoinHostPort(s.host, s.port)

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	msg.WriteString("\r\n")
	if bodyHTML != "" {
		msg.WriteString(bodyHTML)
	} else {
		msg.WriteString(bodyText)
	}

	messageID := fmt.Sprintf("<%d.smtp@%s>", time.Now().UnixNano(), s.host)

	if s.useTLS {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return "", fmt.Errorf("tls dial %s: %w", addr, err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return "", fmt.Errorf("smtp client: %w", err)
		}
		defer client.Close()

		if err := client.Hello("armorclaw.local"); err != nil {
			return "", fmt.Errorf("smtp hello: %w", err)
		}

		if s.username != "" {
			auth := smtp.PlainAuth("", s.username, s.password, s.host)
			if err := client.Auth(auth); err != nil {
				return "", fmt.Errorf("smtp auth: %w", err)
			}
		}

		if err := client.Mail(s.from); err != nil {
			return "", fmt.Errorf("smtp mail from: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return "", fmt.Errorf("smtp rcpt to: %w", err)
		}

		wc, err := client.Data()
		if err != nil {
			return "", fmt.Errorf("smtp data: %w", err)
		}
		if _, err := wc.Write([]byte(msg.String())); err != nil {
			return "", fmt.Errorf("smtp write: %w", err)
		}
		if err := wc.Close(); err != nil {
			return "", fmt.Errorf("smtp close data: %w", err)
		}

		client.Quit()
		return messageID, nil
	}

	err := smtp.SendMail(addr, smtp.PlainAuth("", s.username, s.password, s.host), s.from, []string{to}, []byte(msg.String()))
	if err != nil {
		return "", fmt.Errorf("smtp send mail: %w", err)
	}

	return messageID, nil
}

func (s *SMTPClient) Provider() string {
	return smtpProvider
}

var _ EmailSender = (*SMTPClient)(nil)
