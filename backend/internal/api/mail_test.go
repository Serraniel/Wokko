package api_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/mail"
)

// outbox stands in for the SMTP server and keeps what was sent.
type outbox struct {
	mu   sync.Mutex
	sent []mail.Message
	fail error
}

func (o *outbox) Send(_ context.Context, m mail.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fail != nil {
		return o.fail
	}
	o.sent = append(o.sent, m)
	return nil
}

// wait returns the next mail to an address, waiting a little for mail sent in the background.
func (o *outbox) wait(t *testing.T, to string) mail.Message {
	t.Helper()
	for i := 0; i < 100; i++ {
		o.mu.Lock()
		for j, m := range o.sent {
			if m.To == to {
				o.sent = append(o.sent[:j], o.sent[j+1:]...)
				o.mu.Unlock()
				return m
			}
		}
		o.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no mail to %s", to)
	return mail.Message{}
}

func (o *outbox) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.sent)
}

// linkIn finds the Wokko link in a mail and returns its query.
func linkIn(t *testing.T, m mail.Message) url.Values {
	t.Helper()
	for _, f := range strings.Fields(m.Body) {
		if strings.HasPrefix(f, "http://wokko.test/") {
			u, err := url.Parse(f)
			if err != nil {
				t.Fatal(err)
			}
			return u.Query()
		}
	}
	t.Fatalf("no link in %q", m.Body)
	return nil
}

func TestMailForAccountsInvitesAndForgottenPasswords(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	box := &outbox{}
	e.svc.Mail = box

	var st app.Status
	e.anon().ok("GET", v1+"/status", nil, 200).decode(t, &st)
	if !st.MailEnabled {
		t.Fatal("mail should be on")
	}
	// An address is used only once confirmed through the mailed link.
	var me app.User
	admin.ok("PUT", v1+"/me/email", map[string]string{"email": "admin@example.com"}, 200).decode(t, &me)
	if me.EmailVerified || me.Email != "admin@example.com" {
		t.Fatalf("unconfirmed: %+v", me)
	}
	admin.wantCode("POST", v1+"/admin/mail/test", nil, 409, "no_email")
	confirm := box.wait(t, "admin@example.com")
	e.anon().ok("POST", v1+"/auth/confirm-email", map[string]string{"token": linkIn(t, confirm).Get("token")}, 200)
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	if !me.EmailVerified {
		t.Fatal("not confirmed")
	}
	admin.ok("POST", v1+"/admin/mail/test", nil, 200)
	if m := box.wait(t, "admin@example.com"); !strings.Contains(m.Subject, "Test") {
		t.Fatalf("test mail: %+v", m)
	}

	// An account created with an address gets its setup link by mail, in its language.
	var made app.NewAccount
	admin.ok("POST", v1+"/admin/users", app.NewAccountInput{Handle: "oma", DisplayName: "Oma", Locale: "de", Email: "oma@example.com"}, 201).decode(t, &made)
	if !made.Mailed.Mailed || !made.User.EmailVerified {
		t.Fatalf("made: %+v", made)
	}
	setup := box.wait(t, "oma@example.com")
	q := linkIn(t, setup)
	if q.Get("setup") != "1" || q.Get("token") != made.Token || !strings.Contains(setup.Body, "Hallo Oma") {
		t.Fatalf("setup mail: %+v", setup)
	}
	e.anon().ok("POST", v1+"/auth/reset", map[string]string{"token": q.Get("token"), "password": "omas-secret"}, 200)

	// Invites go to an address the admin types.
	var inv app.Invite
	admin.ok("POST", v1+"/admin/invites", app.InviteInput{MaxUses: 1, Email: "neighbour@example.com", Locale: "en"}, 201).decode(t, &inv)
	if !inv.Mailed.Mailed || linkIn(t, box.wait(t, "neighbour@example.com")).Get("invite") != inv.Token {
		t.Fatalf("invite: %+v", inv)
	}

	// Forgot password: a link for an hour, and the same answer for accounts that don't exist.
	e.anon().ok("POST", v1+"/auth/forgot", map[string]string{"login": "oma"}, 200)
	reset := box.wait(t, "oma@example.com")
	e.anon().ok("POST", v1+"/auth/forgot", map[string]string{"login": "nobody-here"}, 200)
	e.anon().ok("POST", v1+"/auth/forgot", map[string]string{"login": "nobody@example.com"}, 200)
	time.Sleep(50 * time.Millisecond)
	if box.count() != 0 {
		t.Fatalf("mail for an unknown account: %d", box.count())
	}
	real := e.svc.Now
	e.svc.Now = func() time.Time { return real().Add(2 * time.Hour) }
	e.anon().wantCode("POST", v1+"/auth/reset", map[string]string{"token": linkIn(t, reset).Get("token"), "password": "too-late-pw"}, 403, "invalid_reset")
	e.svc.Now = real
	e.anon().ok("POST", v1+"/auth/forgot", map[string]string{"login": "OMA@example.com"}, 200)
	e.anon().ok("POST", v1+"/auth/reset", map[string]string{"token": linkIn(t, box.wait(t, "oma@example.com")).Get("token"), "password": "new-oma-pw"}, 200)

	// An admin's reset link can go out by mail too.
	var users []app.User
	admin.ok("GET", v1+"/admin/users", nil, 200).decode(t, &users)
	var oma app.User
	for _, u := range users {
		if u.Handle == "oma" {
			oma = u
		}
	}
	var res struct {
		Token  string `json:"token"`
		Mailed bool   `json:"mailed"`
	}
	admin.ok("POST", path("/admin/users/%d/reset?send=1", oma.ID), nil, 201).decode(t, &res)
	if !res.Mailed || linkIn(t, box.wait(t, "oma@example.com")).Get("token") != res.Token {
		t.Fatalf("reset by mail: %+v", res)
	}

	// One confirmed address belongs to one account.
	other := e.newUser(admin, "other")
	other.ok("PUT", v1+"/me/email", map[string]string{"email": "oma@example.com"}, 200)
	e.anon().wantCode("POST", v1+"/auth/confirm-email", map[string]string{"token": linkIn(t, box.wait(t, "oma@example.com")).Get("token")}, 409, "email_taken")
	other.wantCode("PUT", v1+"/me/email", map[string]string{"email": "not an address\r\nBcc: x@example.com"}, 400, "bad_email")

	// A failing mail server doesn't fail the action: the link is still there to copy.
	box.fail = errors.New("connection refused")
	admin.ok("POST", v1+"/admin/invites", app.InviteInput{MaxUses: 1, Email: "x@example.com"}, 201).decode(t, &inv)
	if inv.Mailed.Mailed || inv.MailError != "mail_failed" || inv.Token == "" {
		t.Fatalf("failed send: %+v", inv)
	}

	// Without mail, the options are gone.
	e.svc.Mail = nil
	e.anon().wantCode("POST", v1+"/auth/forgot", map[string]string{"login": "oma"}, 404, "mail_off")
	e.anon().ok("GET", v1+"/status", nil, 200).decode(t, &st)
	if st.MailEnabled {
		t.Fatal("mail should be off")
	}
}
