// Package email sends transactional mail (e.g. magic-login links). It exposes a
// small EmailSender interface so handlers can be tested with a mock, an SMTP
// implementation for production, and a logging fallback for dev when SMTP is
// unconfigured (so boot never fails for lack of mail config).
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// EmailSender delivers a single message with both HTML and plaintext bodies.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody, textBody string) error
}

// Config holds SMTP settings sourced from the environment.
type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
	TLS  bool
}

// New returns an SMTPSender when a host is configured, otherwise a LogSender.
func New(cfg Config, logger *slog.Logger) EmailSender {
	if cfg.Host == "" {
		logger.Warn("SMTP_HOST not set; magic-login emails will be logged, not delivered")
		return LogSender{logger: logger}
	}
	port := cfg.Port
	if port == "" {
		port = "587"
	}
	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	return SMTPSender{host: cfg.Host, port: port, user: cfg.User, pass: cfg.Pass, from: from, tls: cfg.TLS}
}

// LogSender logs the email instead of sending it (dev fallback).
type LogSender struct{ logger *slog.Logger }

func (s LogSender) Send(_ context.Context, to, subject, _, _ string) error {
	// Do NOT log the body: it carries the magic link/token. In dev the token is
	// available via the magic-login response instead.
	s.logger.Warn("email not delivered (SMTP unconfigured)", "to", to, "subject", subject)
	return nil
}

// SMTPSender delivers mail over SMTP.
type SMTPSender struct {
	host, port, user, pass, from string
	tls                          bool
}

func (s SMTPSender) auth() smtp.Auth {
	if s.user == "" {
		return nil
	}
	return smtp.PlainAuth("", s.user, s.pass, s.host)
}

func (s SMTPSender) Send(_ context.Context, to, subject, htmlBody, textBody string) error {
	addr := net.JoinHostPort(s.host, s.port)
	msg := buildMIME(s.from, to, subject, htmlBody, textBody)

	if s.tls {
		return s.sendTLS(addr, to, msg)
	}
	// SendMail upgrades to STARTTLS automatically when the server offers it.
	return smtp.SendMail(addr, s.auth(), s.from, []string{to}, msg)
}

// sendTLS handles implicit-TLS servers (e.g. port 465).
func (s SMTPSender) sendTLS(addr, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	if auth := s.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		return err
	}
	return wc.Close()
}

// hdr strips CR/LF so untrusted values can't inject extra email headers.
func hdr(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

// buildMIME assembles a multipart/alternative message (plaintext + HTML).
func buildMIME(from, to, subject, htmlBody, textBody string) []byte {
	const boundary = "blazeaid-boundary-7f3a"
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", hdr(from))
	fmt.Fprintf(&b, "To: %s\r\n", hdr(to))
	fmt.Fprintf(&b, "Subject: %s\r\n", hdr(subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(textBody)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(htmlBody)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}
