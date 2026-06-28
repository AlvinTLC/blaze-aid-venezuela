package email

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestNew_FallbackToLogWhenNoHost(t *testing.T) {
	s := New(Config{}, testLogger())
	if _, ok := s.(LogSender); !ok {
		t.Fatalf("expected LogSender when SMTP_HOST empty, got %T", s)
	}
}

func TestNew_SMTPWhenHostSet(t *testing.T) {
	s := New(Config{Host: "smtp.example.com", User: "u", Pass: "p"}, testLogger())
	sender, ok := s.(SMTPSender)
	if !ok {
		t.Fatalf("expected SMTPSender, got %T", s)
	}
	if sender.port != "587" {
		t.Fatalf("expected default port 587, got %q", sender.port)
	}
	if sender.from != "u" {
		t.Fatalf("expected From to default to user, got %q", sender.from)
	}
}

func TestBuildMIME(t *testing.T) {
	msg := string(buildMIME("from@x", "to@y", "Hi", "<b>html</b>", "plain"))
	for _, want := range []string{
		"From: from@x", "To: to@y", "Subject: Hi",
		"multipart/alternative", "text/plain", "text/html", "plain", "<b>html</b>",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("MIME missing %q in:\n%s", want, msg)
		}
	}
}
