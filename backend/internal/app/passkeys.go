package app

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

const passkeyFlowLifetime = 5 * time.Minute

// passkeyFlows remembers challenges between the two steps of a passkey ceremony. They only
// live for minutes, so a restart at worst means trying again.
type passkeyFlows struct {
	mu    sync.Mutex
	flows map[string]passkeyFlow
}

type passkeyFlow struct {
	session webauthn.SessionData
	userID  int64
	expires time.Time
}

func (f *passkeyFlows) put(s webauthn.SessionData, userID int64, now time.Time) (string, error) {
	id, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.flows == nil {
		f.flows = map[string]passkeyFlow{}
	}
	for k, v := range f.flows {
		if v.expires.Before(now) {
			delete(f.flows, k)
		}
	}
	f.flows[id] = passkeyFlow{session: s, userID: userID, expires: now.Add(passkeyFlowLifetime)}
	return id, nil
}

// take can only succeed once, so a challenge can't be replayed.
func (f *passkeyFlows) take(id string, now time.Time) (passkeyFlow, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	flow, ok := f.flows[id]
	delete(f.flows, id)
	return flow, ok && flow.expires.After(now)
}

type passkeyUser struct {
	u     sqlcgen.User
	creds []webauthn.Credential
}

func userHandle(id int64) []byte { return binary.BigEndian.AppendUint64(nil, uint64(id)) }

func (p passkeyUser) WebAuthnID() []byte   { return userHandle(p.u.ID) }
func (p passkeyUser) WebAuthnName() string { return p.u.Handle }
func (p passkeyUser) WebAuthnDisplayName() string {
	if p.u.DisplayName != "" {
		return p.u.DisplayName
	}
	return p.u.Handle
}
func (p passkeyUser) WebAuthnCredentials() []webauthn.Credential { return p.creds }

var errNoPasskeys = fail(http.StatusBadRequest, "passkey_unavailable", "Passkeys need HTTPS or localhost")

// relyingParty ties passkeys to this site. PUBLIC_URL decides when set. Otherwise it's the
// address the browser is using, which the API's origin check has already compared to the host.
func (s *Service) relyingParty(origin string) (*webauthn.WebAuthn, error) {
	if s.Cfg.PublicURL != "" {
		origin = s.Cfg.PublicURL
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return nil, errNoPasskeys
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID: u.Hostname(), RPDisplayName: "Wokko", RPOrigins: []string{strings.TrimRight(origin, "/")},
		AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationPreferred},
	})
	if err != nil {
		return nil, errNoPasskeys
	}
	return wa, nil
}

func (s *Service) passkeyUser(ctx context.Context, q *sqlcgen.Queries, u sqlcgen.User) (passkeyUser, error) {
	out := passkeyUser{u: u}
	rows, err := q.ListPasskeys(ctx, u.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal([]byte(r.Credential), &c); err != nil {
			return out, err
		}
		out.creds = append(out.creds, c)
	}
	return out, nil
}

func toPasskey(p sqlcgen.Passkey) Passkey {
	return Passkey{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt, LastUsedAt: p.LastUsedAt.String}
}

func (s *Service) ListPasskeys(ctx context.Context, v Viewer) ([]Passkey, error) {
	rows, err := s.Store.Queries.ListPasskeys(ctx, v.id())
	if err != nil {
		return nil, err
	}
	out := make([]Passkey, len(rows))
	for i, p := range rows {
		out[i] = toPasskey(p)
	}
	return out, nil
}

func (s *Service) DeletePasskey(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.DeletePasskey(ctx, sqlcgen.DeletePasskeyParams{ID: id, UserID: v.id()})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// BeginPasskeyRegistration starts adding a passkey and returns the flow's ID and the options
// for the browser.
func (s *Service) BeginPasskeyRegistration(ctx context.Context, v Viewer, origin string) (string, any, error) {
	wa, err := s.relyingParty(origin)
	if err != nil {
		return "", nil, err
	}
	user, err := s.passkeyUser(ctx, s.Store.Queries, *v.User)
	if err != nil {
		return "", nil, err
	}
	exclude := make([]protocol.CredentialDescriptor, len(user.creds))
	for i, c := range user.creds {
		exclude[i] = c.Descriptor()
	}
	options, session, err := wa.BeginRegistration(user, webauthn.WithExclusions(exclude))
	if err != nil {
		return "", nil, errNoPasskeys
	}
	flow, err := s.passkeys.put(*session, v.id(), s.Now())
	return flow, options, err
}

func (s *Service) FinishPasskeyRegistration(ctx context.Context, v Viewer, flowID, name, origin string, r *http.Request) (Passkey, error) {
	flow, ok := s.passkeys.take(flowID, s.Now())
	if !ok || flow.userID != v.id() {
		return Passkey{}, invalid("passkey_failed", "That didn't work. Please try again")
	}
	wa, err := s.relyingParty(origin)
	if err != nil {
		return Passkey{}, err
	}
	user, err := s.passkeyUser(ctx, s.Store.Queries, *v.User)
	if err != nil {
		return Passkey{}, err
	}
	cred, err := wa.FinishRegistration(user, flow.session, r)
	if err != nil {
		s.Log.Info("passkey registration failed", "error", err)
		return Passkey{}, invalid("passkey_failed", "That didn't work. Please try again")
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return Passkey{}, err
	}
	if name = strings.TrimSpace(name); name == "" {
		name = "Passkey"
	}
	var out Passkey
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		p, err := q.CreatePasskey(ctx, sqlcgen.CreatePasskeyParams{UserID: v.id(), Name: firstRunes(name, 100), CredentialID: base64.RawURLEncoding.EncodeToString(cred.ID), Credential: string(raw)})
		out = toPasskey(p)
		return err
	})
	return out, err
}

// BeginPasskeyLogin needs no handle, so the login form can't reveal who has an account.
func (s *Service) BeginPasskeyLogin(ctx context.Context, origin string) (string, any, error) {
	wa, err := s.relyingParty(origin)
	if err != nil {
		return "", nil, err
	}
	options, session, err := wa.BeginDiscoverableLogin()
	if err != nil {
		return "", nil, errNoPasskeys
	}
	flow, err := s.passkeys.put(*session, 0, s.Now())
	return flow, options, err
}

// FinishPasskeyLogin checks the signature and starts a session.
func (s *Service) FinishPasskeyLogin(ctx context.Context, flowID, origin string, r *http.Request) (User, string, error) {
	badLogin := invalid("passkey_failed", "That didn't work. Please try again")
	flow, ok := s.passkeys.take(flowID, s.Now())
	if !ok {
		return User{}, "", badLogin
	}
	wa, err := s.relyingParty(origin)
	if err != nil {
		return User{}, "", err
	}
	q := s.Store.Queries
	var found sqlcgen.User
	handler := func(_, handle []byte) (webauthn.User, error) {
		if len(handle) != 8 {
			return nil, errors.New("unexpected user handle")
		}
		u, err := q.GetUser(ctx, int64(binary.BigEndian.Uint64(handle)))
		if err != nil {
			return nil, err
		}
		found = u
		return s.passkeyUser(ctx, q, u)
	}
	_, cred, err := wa.FinishPasskeyLogin(handler, flow.session, r)
	if err != nil {
		s.Log.Info("passkey login failed", "error", err)
		return User{}, "", badLogin
	}
	if found.DisabledAt.Valid {
		return User{}, "", badLogin
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return User{}, "", err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.UpdatePasskeyUse(ctx, sqlcgen.UpdatePasskeyUseParams{Credential: string(raw), CredentialID: base64.RawURLEncoding.EncodeToString(cred.ID)})
	})
	if err != nil {
		return User{}, "", err
	}
	token, err := s.newSession(ctx, found.ID)
	if err != nil {
		return User{}, "", err
	}
	return toUser(found), token, nil
}
