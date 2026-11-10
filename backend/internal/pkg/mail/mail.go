// Package mail sends transactional mail over SMTP, or logs it when no
// SMTP host is configured (local/dev behavior).
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// Mailer sends one transactional message.
type Mailer interface {
	Send(ctx context.Context, to, subject, textBody string) error
}

// LogMailer records mail via structured logs instead of sending.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, textBody string) error {
	slog.Info("mail.send", "to", to, "subject", subject, "body_chars", len(textBody))
	return nil
}

// SMTPMailer delivers mail via SMTP with optional authentication.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

// NewSMTPMailer builds an SMTP mailer; From falls back to a no-reply address.
func NewSMTPMailer(host string, port int, username, password, from, fromName string) *SMTPMailer {
	if strings.TrimSpace(from) == "" {
		from = "no-reply@avibemusic.com"
	}
	return &SMTPMailer{Host: host, Port: port, Username: username, Password: password, From: from, FromName: fromName}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, textBody string) error {
	if strings.TrimSpace(to) == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("invalid recipient")
	}
	from := m.From
	if strings.TrimSpace(m.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", m.FromName, m.From)
	}
	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + textBody
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	var auth smtp.Auth
	if strings.TrimSpace(m.Username) != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
}
