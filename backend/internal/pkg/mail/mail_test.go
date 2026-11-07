package mail

import (
	"context"
	"strings"
	"testing"
)

func TestSMTPMailerRejectsBadRecipient(t *testing.T) {
	m := NewSMTPMailer("smtp.example.com", 587, "u", "p", "", "")
	for _, to := range []string{"", "not-an-email"} {
		if err := m.Send(context.Background(), to, "s", "b"); err == nil {
			t.Fatalf("recipient %q accepted", to)
		}
	}
}

func TestLogMailerAcceptsAnything(t *testing.T) {
	if err := (LogMailer{}).Send(context.Background(), "u@example.com", "s", strings.Repeat("b", 10)); err != nil {
		t.Fatalf("log mailer: %v", err)
	}
}
