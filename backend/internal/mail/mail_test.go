package mail

import (
	"bufio"
	"context"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/serraniel/wokko/backend/internal/config"
)

// fakeServer speaks just enough SMTP to receive one message.
func fakeServer(t *testing.T) (string, <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		defer ln.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		say("220 fake")
		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case strings.HasPrefix(cmd, "DATA"):
				say("354 go on")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					data.WriteString(l)
				}
				got <- data.String()
				say("250 queued")
			case strings.HasPrefix(cmd, "QUIT"):
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	return ln.Addr().String(), got
}

func TestSendsThroughSMTP(t *testing.T) {
	addr, got := fakeServer(t)
	host, port, _ := net.SplitHostPort(addr)
	s, err := New(config.SMTP{Host: host, Port: atoi(port), From: "Wokko <wokko@example.com>", TLS: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send(context.Background(), Compose("reset", "de", "oma@example.com", map[string]string{"site": "Wokko", "name": "Oma", "link": "https://w.example/reset?token=abc", "hours": "1"})); err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-got:
		msg, err := mail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
		body, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
		if msg.Header.Get("To") != "oma@example.com" || subject != "Neues Passwort für Wokko" || !strings.Contains(string(body), "Hallo Oma") || !strings.Contains(string(body), "https://w.example/reset?token=abc") {
			t.Fatalf("message:\n%s", raw)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing arrived")
	}
}

func TestHeadersCantBeInjected(t *testing.T) {
	from := &mail.Address{Name: "Wokko", Address: "wokko@example.com"}
	for _, to := range []string{"a@example.com\r\nBcc: x@example.com", "Someone <a@example.com>", "not an address", ""} {
		if _, err := Build(from, Message{To: to, Subject: "Hi", Body: "x"}, time.Now()); err == nil {
			t.Errorf("accepted %q", to)
		}
	}
	if _, err := Build(from, Message{To: "a@example.com", Subject: "Hi\r\nBcc: x@example.com", Body: "x"}, time.Now()); err == nil {
		t.Error("a line break in the subject was accepted")
	}
	// Composed subjects never carry line breaks, whatever is filled in.
	if m := Compose("test", "en", "a@example.com", map[string]string{"site": "Evil\r\nBcc: x@example.com"}); strings.ContainsAny(m.Subject, "\r\n") {
		t.Errorf("subject: %q", m.Subject)
	}
}

func TestEveryTextExistsInEveryLanguage(t *testing.T) {
	for lang, kinds := range texts {
		for _, k := range Kinds() {
			if kinds[k][0] == "" || kinds[k][1] == "" {
				t.Errorf("%s has no %s mail", lang, k)
			}
		}
	}
	if m := Compose("invite", "fr", "a@example.com", map[string]string{"site": "Wokko", "link": "L"}); !strings.Contains(m.Body, "invited") {
		t.Errorf("fallback to English: %q", m.Body)
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}
