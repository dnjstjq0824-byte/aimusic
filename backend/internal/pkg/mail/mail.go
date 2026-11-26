// Package mail sends transactional mail over SMTP, or logs it when no
// SMTP host is configured (local/dev behavior).
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
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

// sanitizeHeaderValue strips CR/LF so caller-supplied values (company names
// in subjects, user emails as recipients) cannot inject SMTP headers.
func sanitizeHeaderValue(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, textBody string) error {
	to = sanitizeHeaderValue(to)
	subject = sanitizeHeaderValue(subject)
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
	// Bounded dial: a blackholed relay must never hang the request path.
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()
	needsAuth := strings.TrimSpace(m.Username) != ""
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if needsAuth {
		// Credentials (and token-bearing mail) must never cross plaintext SMTP.
		return fmt.Errorf("smtp server offers no STARTTLS")
	}
	if needsAuth {
		if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(sanitizeHeaderValue(m.From)); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}
