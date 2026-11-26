package mail

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
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

// fakeSMTP serves one minimal dialogue: greeting, EHLO (optionally without
// STARTTLS), MAIL/RCPT/DATA. It records whether DATA completed.
type fakeSMTP struct {
	advertiseTLS bool
	gotData      bool
}

func (f *fakeSMTP) serve(t *testing.T, ln net.Listener) {
	t.Helper()
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	fmt.Fprintf(rw, "220 fake ESMTP\r\n")
	_ = rw.Flush()
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}
		up := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(up, "EHLO"), strings.HasPrefix(up, "HELO"):
			fmt.Fprintf(rw, "250-fake\r\n")
			if f.advertiseTLS {
				fmt.Fprintf(rw, "250-STARTTLS\r\n")
			}
			fmt.Fprintf(rw, "250 OK\r\n")
		case strings.HasPrefix(up, "MAIL FROM"), strings.HasPrefix(up, "RCPT TO"):
			fmt.Fprintf(rw, "250 OK\r\n")
		case strings.HasPrefix(up, "DATA"):
			fmt.Fprintf(rw, "354 go ahead\r\n")
			_ = rw.Flush()
			for {
				dl, err := rw.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
			}
			f.gotData = true
			fmt.Fprintf(rw, "250 queued\r\n")
		case strings.HasPrefix(up, "QUIT"), strings.HasPrefix(up, "RSET"):
			fmt.Fprintf(rw, "221 bye\r\n")
			_ = rw.Flush()
			return
		default:
			fmt.Fprintf(rw, "502 unimplemented\r\n")
		}
		_ = rw.Flush()
	}
}

func TestSMTPMailerNoAuthDialogue(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	f := &fakeSMTP{}
	go f.serve(t, ln)
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	var portN int
	fmt.Sscanf(port, "%d", &portN)
	m := NewSMTPMailer(host, portN, "", "", "no-reply@avibemusic.com", "")
	if err := m.Send(context.Background(), "u@example.com", "hi", "body"); err != nil {
		t.Fatalf("no-auth send: %v", err)
	}
	if !f.gotData {
		t.Fatalf("server never received DATA")
	}
}

func TestSMTPMailerAuthRequiresSTARTTLS(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	f := &fakeSMTP{}
	go f.serve(t, ln)
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	var portN int
	fmt.Sscanf(port, "%d", &portN)
	m := NewSMTPMailer(host, portN, "user", "pass", "no-reply@avibemusic.com", "")
	if err := m.Send(context.Background(), "u@example.com", "hi", "body"); err == nil ||
		!strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("auth over plaintext must be refused, got %v", err)
	}
	if f.gotData {
		t.Fatalf("refused send must not reach DATA")
	}
}

// P1: caller-supplied header values (company names in subjects, user emails)
// must not be able to inject SMTP headers via CR/LF.

func TestSanitizeHeaderValue(t *testing.T) {
	in := "[AVIBE] business inquiry: Acme\r\nBcc: evil@x.example"
	got := sanitizeHeaderValue(in)
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("CR/LF survived: %q", got)
	}
	want := "[AVIBE] business inquiry: Acme  Bcc: evil@x.example"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if plain := "Verify your email"; sanitizeHeaderValue(plain) != plain {
		t.Fatalf("plain subject mangled: %q", sanitizeHeaderValue(plain))
	}
}
