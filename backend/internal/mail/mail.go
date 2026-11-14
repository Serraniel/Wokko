// Package mail sends Wokko's few mails, such as invites and password links, through an SMTP
// server: a mail provider, a sending service or a relay next to Wokko. Wokko doesn't deliver mail
// itself; that needs a server others trust, which a home connection usually isn't.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/config"
)

// Message is one plain-text mail.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender sends mail. Tests swap in their own.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// ErrBadAddress means an address can't be used, for example because it holds a line break.
var ErrBadAddress = errors.New("mail: not a valid address")

// Address checks an email address and returns it plainly, without a name.
func Address(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 254 || strings.ContainsAny(s, "\r\n") {
		return "", ErrBadAddress
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Name != "" || !strings.Contains(a.Address, "@") {
		return "", ErrBadAddress
	}
	return a.Address, nil
}

// SMTP sends through the configured server.
type SMTP struct {
	cfg  config.SMTP
	from *mail.Address
	// Dial is replaced in tests.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
}

// New returns nil when no SMTP server is configured, so mail is simply off.
func New(cfg config.SMTP) (*SMTP, error) {
	if cfg.Host == "" {
		return nil, nil
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM: %w", err)
	}
	d := &net.Dialer{Timeout: 15 * time.Second}
	return &SMTP{cfg: cfg, from: from, Dial: d.DialContext}, nil
}

// Build writes a message with its headers. Every header value is checked or encoded, so text
// from people can't add headers.
func Build(from *mail.Address, m Message, now time.Time) ([]byte, error) {
	to, err := Address(m.To)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return nil, errors.New("mail: subject holds a line break")
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := from.Address[strings.LastIndex(from.Address, "@")+1:]
	var b bytes.Buffer
	header := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	header("From", from.String())
	header("To", to)
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", now.Format(time.RFC1123Z))
	header("Message-ID", "<"+hex.EncodeToString(id)+"@"+domain+">")
	header("MIME-Version", "1.0")
	header("Content-Type", `text/plain; charset="utf-8"`)
	header("Content-Transfer-Encoding", "quoted-printable")
	header("Auto-Submitted", "auto-generated")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(strings.ReplaceAll(m.Body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Send delivers one message to the SMTP server.
func (s *SMTP) Send(ctx context.Context, m Message) error {
	to, err := Address(m.To)
	if err != nil {
		return err
	}
	data, err := Build(s.from, m, time.Now())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	conn, err := s.Dial(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail: connect to %s: %w", addr, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	if s.cfg.TLS == "tls" {
		conn = tls.Client(conn, tlsCfg)
	}
	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mail: %w", err)
	}
	defer c.Close()
	if s.cfg.TLS == "starttls" {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("mail: the server doesn't offer STARTTLS; set SMTP_TLS=tls or, for a relay on a private network, none")
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}
	if s.cfg.Username != "" {
		// PlainAuth refuses to send the password without TLS, except to localhost.
		if err := c.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("mail: login: %w", err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fmt.Errorf("mail: sender refused: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mail: recipient refused: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("mail: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: %w", err)
	}
	return c.Quit()
}
