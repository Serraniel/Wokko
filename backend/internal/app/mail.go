package app

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/mail"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// MailReady tells whether Wokko can send mail. Links in mail need PUBLIC_URL: built from a
// request's Host header, they could point anywhere.
func (s *Service) MailReady() bool {
	return s.Mail != nil && s.Cfg.PublicURL != ""
}

func (s *Service) link(path string, params url.Values) string {
	return s.Cfg.PublicURL + path + "?" + params.Encode()
}

func (s *Service) siteName(ctx context.Context) string {
	name, err := s.settingString(ctx, "preview_site_name", "")
	if err != nil || name == "" {
		return "Wokko"
	}
	return name
}

// send composes and sends one mail. A failure is logged and returned as a Mailed code, so the
// action itself still succeeds and its link can be copied by hand.
func (s *Service) send(ctx context.Context, kind, locale, to string, vars map[string]string) Mailed {
	if !s.MailReady() || to == "" {
		return Mailed{MailError: "mail_off"}
	}
	if vars == nil {
		vars = map[string]string{}
	}
	vars["site"] = s.siteName(ctx)
	if err := s.Mail.Send(ctx, mail.Compose(kind, locale, to, vars)); err != nil {
		s.Log.Warn("could not send mail", "kind", kind, "error", err)
		if errors.Is(err, mail.ErrBadAddress) {
			return Mailed{MailError: "bad_email"}
		}
		return Mailed{MailError: "mail_failed"}
	}
	return Mailed{Mailed: true}
}

// AdminMailStatus shows admins whether mail works, without the password.
func (s *Service) AdminMailStatus(v Viewer) (MailStatus, error) {
	if !v.isAdmin() {
		return MailStatus{}, ErrForbidden
	}
	c := s.Cfg.SMTP
	return MailStatus{Configured: s.Mail != nil, PublicURL: s.Cfg.PublicURL != "", Ready: s.MailReady(), Host: c.Host, From: c.From}, nil
}

// SendTestMail sends a test mail to the admin's own confirmed address. Unlike other mail, a
// failure here is the answer, so it's returned.
func (s *Service) SendTestMail(ctx context.Context, v Viewer) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	if !s.MailReady() {
		return fail(http.StatusConflict, "mail_off", "Mail isn't set up on this server")
	}
	if v.User.Email == "" || v.User.EmailVerified == 0 {
		return fail(http.StatusConflict, "no_email", "Add and confirm your email address in Settings first")
	}
	if err := s.Mail.Send(ctx, mail.Compose("test", v.User.Locale, v.User.Email, map[string]string{"site": s.siteName(ctx), "name": v.User.DisplayName})); err != nil {
		s.Log.Warn("test mail failed", "error", err)
		return fail(http.StatusBadGateway, "mail_failed", "The mail server didn't accept the mail: "+err.Error())
	}
	return nil
}

// SetEmail stores an address and mails a link to confirm it. An empty address removes it.
func (s *Service) SetEmail(ctx context.Context, v Viewer, address string) (User, error) {
	if v.User == nil {
		return User{}, ErrUnauthorized
	}
	if address == "" {
		err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
			if err := q.DeleteEmailConfirmations(ctx, v.id()); err != nil {
				return err
			}
			return q.SetUserEmail(ctx, sqlcgen.SetUserEmailParams{Email: "", EmailVerified: 0, ID: v.id()})
		})
		if err != nil {
			return User{}, err
		}
		return s.reloadUser(ctx, v.id())
	}
	if !s.MailReady() {
		return User{}, fail(http.StatusConflict, "mail_off", "Mail isn't set up on this server")
	}
	addr, err := mail.Address(address)
	if err != nil {
		return User{}, invalid("bad_email", "This isn't an email address")
	}
	token, err := auth.NewToken()
	if err != nil {
		return User{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.DeleteEmailConfirmations(ctx, v.id()); err != nil {
			return err
		}
		if err := q.SetUserEmail(ctx, sqlcgen.SetUserEmailParams{Email: addr, EmailVerified: 0, ID: v.id()}); err != nil {
			return err
		}
		return q.CreateEmailConfirmation(ctx, sqlcgen.CreateEmailConfirmationParams{TokenHash: auth.HashToken(token), UserID: v.id(), Email: addr, ExpiresAt: s.stamp(s.Now().Add(48 * time.Hour))})
	})
	if err != nil {
		return User{}, err
	}
	if m := s.send(ctx, "confirm", v.User.Locale, addr, map[string]string{"name": v.User.DisplayName, "link": s.link("/confirm-email", url.Values{"token": {token}})}); !m.Mailed {
		return User{}, fail(http.StatusBadGateway, m.MailError, "The confirmation mail couldn't be sent. Please try again later")
	}
	return s.reloadUser(ctx, v.id())
}

// ConfirmEmail marks an address as confirmed through the link mailed to it.
func (s *Service) ConfirmEmail(ctx context.Context, token string) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		c, err := q.GetEmailConfirmation(ctx, sqlcgen.GetEmailConfirmationParams{TokenHash: auth.HashToken(token), ExpiresAt: s.stamp(s.Now())})
		if isNoRows(err) {
			return fail(http.StatusForbidden, "invalid_link", "This link is invalid or has expired")
		}
		if err != nil {
			return err
		}
		if other, err := q.GetUserByVerifiedEmail(ctx, c.Email); err == nil && other.ID != c.UserID {
			return fail(http.StatusConflict, "email_taken", "Another account already uses this address")
		} else if err != nil && !isNoRows(err) {
			return err
		}
		if err := q.SetUserEmail(ctx, sqlcgen.SetUserEmailParams{Email: c.Email, EmailVerified: 1, ID: c.UserID}); err != nil {
			return err
		}
		return q.DeleteEmailConfirmations(ctx, c.UserID)
	})
}

func (s *Service) reloadUser(ctx context.Context, id int64) (User, error) {
	u, err := s.Store.Queries.GetUser(ctx, id)
	if err != nil {
		return User{}, err
	}
	return toUser(u), nil
}

// ForgotPassword mails a reset link valid for an hour, to an account found by handle or
// confirmed address. It answers the same either way and sends in the background, so it
// reveals nothing about which accounts exist.
func (s *Service) ForgotPassword(ctx context.Context, login string) error {
	if !s.MailReady() {
		return fail(http.StatusNotFound, "mail_off", "Mail isn't set up on this server")
	}
	q := s.Store.Queries
	u, err := q.GetUserByHandle(ctx, login)
	if isNoRows(err) {
		if addr, aerr := mail.Address(login); aerr == nil {
			u, err = q.GetUserByVerifiedEmail(ctx, addr)
		}
	}
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.EmailVerified == 0 || u.Email == "" || u.DisabledAt.Valid {
		return nil
	}
	var token string
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		token, err = s.resetLinkFor(ctx, q, u.ID, time.Hour)
		return err
	})
	if err != nil {
		return err
	}
	go s.send(context.Background(), "reset", u.Locale, u.Email, map[string]string{"name": u.DisplayName, "link": s.link("/reset", url.Values{"token": {token}}), "hours": "1"})
	return nil
}

// MailResetLink mails an admin's reset or setup link to the person's confirmed address.
func (s *Service) MailResetLink(ctx context.Context, v Viewer, userID int64, token string) Mailed {
	u, err := s.Store.Queries.GetUser(ctx, userID)
	if err != nil || u.Email == "" || u.EmailVerified == 0 {
		return Mailed{MailError: "no_email"}
	}
	params := url.Values{"token": {token}}
	kind, hours := "reset", "72"
	if u.NeedsSetup != 0 {
		kind = "setup"
		params.Set("setup", "1")
	}
	return s.send(ctx, kind, u.Locale, u.Email, map[string]string{"name": u.DisplayName, "link": s.link("/reset", params), "hours": hours})
}

func (s *Service) mailInvite(ctx context.Context, v Viewer, inv Invite, in InviteInput) Mailed {
	if in.Email == "" {
		return Mailed{}
	}
	locale := in.Locale
	if locale == "" {
		locale = v.User.Locale
	}
	return s.send(ctx, "invite", locale, in.Email, map[string]string{"link": s.link("/register", url.Values{"invite": {inv.Token}})})
}
