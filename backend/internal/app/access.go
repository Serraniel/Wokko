package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

const (
	tokenPrefix       = "wokko_"
	oauthCodeLifetime = 5 * time.Minute
	oauthAccessLife   = time.Hour
	oauthRefreshLife  = 60 * 24 * time.Hour
)

func validScope(s string) bool { return s == "read" || s == "write" || s == "delete" }

func toAPIToken(t sqlcgen.ApiToken) APIToken {
	return APIToken{ID: t.ID, Name: t.Name, Scope: t.Scope, IsApp: t.ClientID != "", ExpiresAt: t.ExpiresAt.String, LastUsedAt: t.LastUsedAt.String, CreatedAt: t.CreatedAt}
}

func (s *Service) MCPEnabled(ctx context.Context) (bool, error) {
	return s.settingBool(ctx, "mcp_enabled")
}

func (s *Service) ListAPITokens(ctx context.Context, v Viewer) ([]APIToken, error) {
	rows, err := s.Store.Queries.ListAPITokens(ctx, v.id())
	if err != nil {
		return nil, err
	}
	now := s.stamp(s.Now())
	out := []APIToken{}
	for _, t := range rows {
		if !t.ExpiresAt.Valid || t.ExpiresAt.String > now {
			out = append(out, toAPIToken(t))
		}
	}
	return out, nil
}

func (s *Service) CreateAPIToken(ctx context.Context, v Viewer, in APITokenInput) (NewAPIToken, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return NewAPIToken{}, err
	}
	if !validScope(in.Scope) {
		return NewAPIToken{}, invalid("bad_scope", "Unknown access level")
	}
	secret, err := auth.NewToken()
	if err != nil {
		return NewAPIToken{}, err
	}
	secret = tokenPrefix + secret
	var out NewAPIToken
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		p := sqlcgen.CreateAPITokenParams{UserID: v.id(), Name: name, TokenHash: auth.HashToken(secret), Scope: in.Scope}
		if in.ExpiresInDays > 0 {
			p.ExpiresAt = nullStr(s.stamp(s.Now().AddDate(0, 0, int(in.ExpiresInDays))))
		}
		t, err := q.CreateAPIToken(ctx, p)
		if err != nil {
			return err
		}
		out = NewAPIToken{APIToken: toAPIToken(t), Secret: secret}
		return nil
	})
	return out, err
}

// RevokeAPIToken also ends an app's refresh tokens, so it can't simply get a new token.
func (s *Service) RevokeAPIToken(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		rows, err := q.ListAPITokens(ctx, v.id())
		if err != nil {
			return err
		}
		i := slices.IndexFunc(rows, func(t sqlcgen.ApiToken) bool { return t.ID == id })
		if i < 0 {
			return ErrNotFound
		}
		if c := rows[i].ClientID; c != "" {
			if err := q.DeleteOAuthRefreshForClient(ctx, sqlcgen.DeleteOAuthRefreshForClientParams{UserID: v.id(), ClientID: c}); err != nil {
				return err
			}
			return q.DeleteAPITokensForClient(ctx, sqlcgen.DeleteAPITokensForClientParams{UserID: v.id(), ClientID: c})
		}
		_, err = q.DeleteAPIToken(ctx, sqlcgen.DeleteAPITokenParams{ID: id, UserID: v.id()})
		return err
	})
}

// TokenAccess is who a token acts for and what it may do.
type TokenAccess struct {
	Viewer    Viewer
	Scope     string
	TokenName string
}

// AuthenticateToken checks a Bearer token. A disabled owner's tokens stop working at once.
func (s *Service) AuthenticateToken(ctx context.Context, raw string) (TokenAccess, error) {
	if raw == "" {
		return TokenAccess{}, ErrUnauthorized
	}
	q := s.Store.Queries
	t, err := q.GetAPITokenByHash(ctx, auth.HashToken(raw))
	if isNoRows(err) {
		return TokenAccess{}, ErrUnauthorized
	}
	if err != nil {
		return TokenAccess{}, err
	}
	now := s.Now()
	if t.ExpiresAt.Valid && t.ExpiresAt.String <= s.stamp(now) {
		return TokenAccess{}, ErrUnauthorized
	}
	u, err := q.GetUser(ctx, t.UserID)
	if err != nil {
		return TokenAccess{}, err
	}
	if u.DisabledAt.Valid {
		return TokenAccess{}, ErrUnauthorized
	}
	// Once an hour is exact enough for "last used", and saves a write on every call.
	if !t.LastUsedAt.Valid || t.LastUsedAt.String < s.stamp(now.Add(-time.Hour)) {
		touch := func(q *sqlcgen.Queries) error {
			return q.TouchAPIToken(ctx, sqlcgen.TouchAPITokenParams{LastUsedAt: nullStr(s.stamp(now)), ID: t.ID})
		}
		if err := s.Store.Tx(ctx, touch); err != nil {
			s.Log.Warn("could not note token use", "error", err) // the token itself is still valid
		}
	}
	return TokenAccess{Viewer: Viewer{User: &u, Locale: u.Locale}, Scope: t.Scope, TokenName: t.Name}, nil
}

func (s *Service) LogAssistantActivity(ctx context.Context, a TokenAccess, tool, target string) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.AddAssistantActivity(ctx, sqlcgen.AddAssistantActivityParams{UserID: a.Viewer.id(), TokenName: a.TokenName, Tool: tool, Target: firstRunes(target, 200)}); err != nil {
			return err
		}
		newest, err := q.MaxAssistantActivityID(ctx, a.Viewer.id())
		if err != nil {
			return err
		}
		return q.DeleteOldAssistantActivity(ctx, sqlcgen.DeleteOldAssistantActivityParams{UserID: a.Viewer.id(), ID: newest - 1000})
	})
	if err != nil {
		s.Log.Warn("could not record assistant activity", "error", err) // the change itself is saved
	}
}

func (s *Service) ListAssistantActivity(ctx context.Context, v Viewer) ([]AssistantActivity, error) {
	rows, err := s.Store.Queries.ListAssistantActivity(ctx, v.id())
	if err != nil {
		return nil, err
	}
	out := make([]AssistantActivity, len(rows))
	for i, r := range rows {
		out[i] = AssistantActivity{ID: r.ID, TokenName: r.TokenName, Tool: r.Tool, Target: r.Target, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// --- OAuth, for assistants that connect by themselves ---

func oauthError(code, msg string) *Error { return fail(http.StatusBadRequest, code, msg) }

// validRedirect allows https, and plain http only back to the same computer, which is how
// local apps receive their code. Anything else could hand the code to someone else.
func validRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.User != nil || u.Hostname() == "" {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && isLoopback(u.Hostname()))
}

func isLoopback(host string) bool { return host == "localhost" || host == "127.0.0.1" || host == "::1" }

// redirectMatches compares exactly, except that local apps may pick any port (RFC 8252).
func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	a, err1 := url.Parse(registered)
	b, err2 := url.Parse(requested)
	if err1 != nil || err2 != nil || a.Scheme != "http" || !isLoopback(a.Hostname()) {
		return false
	}
	return b.Scheme == "http" && a.Hostname() == b.Hostname() && a.Path == b.Path && a.RawQuery == b.RawQuery
}

// RegisterOAuthClient lets an app introduce itself, which MCP clients do automatically. It
// grants nothing by itself: a person still has to approve the app.
func (s *Service) RegisterOAuthClient(ctx context.Context, name string, redirects []string) (string, error) {
	if len(redirects) == 0 || len(redirects) > 10 {
		return "", oauthError("invalid_redirect_uri", "A redirect address is needed")
	}
	for _, r := range redirects {
		if !validRedirect(r) {
			return "", oauthError("invalid_redirect_uri", "This redirect address isn't allowed")
		}
	}
	id, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(redirects)
	if err != nil {
		return "", err
	}
	if name = strings.TrimSpace(name); name == "" {
		name = "Unnamed app"
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.CreateOAuthClient(ctx, sqlcgen.CreateOAuthClientParams{ClientID: id, Name: firstRunes(name, 100), RedirectUris: string(raw)})
	})
	return id, err
}

func (s *Service) oauthClient(ctx context.Context, q *sqlcgen.Queries, clientID, redirect string) (sqlcgen.OauthClient, error) {
	c, err := q.GetOAuthClient(ctx, clientID)
	if isNoRows(err) {
		return c, oauthError("invalid_client", "This app isn't known")
	}
	if err != nil {
		return c, err
	}
	var allowed []string
	if err := json.Unmarshal([]byte(c.RedirectUris), &allowed); err != nil {
		return c, err
	}
	if !slices.ContainsFunc(allowed, func(r string) bool { return redirectMatches(r, redirect) }) {
		return c, oauthError("invalid_redirect_uri", "This redirect address isn't allowed")
	}
	return c, nil
}

// normalScope turns OAuth's scope text into one of Wokko's access levels.
func normalScope(raw string) string {
	best := "read"
	for _, part := range strings.Fields(raw) {
		if part == "delete" || (part == "write" && best != "delete") {
			best = part
		}
	}
	return best
}

// DescribeOAuthRequest gives the consent page what to show.
func (s *Service) DescribeOAuthRequest(ctx context.Context, clientID, redirect, scope string) (OAuthRequest, error) {
	c, err := s.oauthClient(ctx, s.Store.Queries, clientID, redirect)
	if err != nil {
		return OAuthRequest{}, err
	}
	u, err := url.Parse(redirect)
	if err != nil {
		return OAuthRequest{}, oauthError("invalid_redirect_uri", "This redirect address isn't allowed")
	}
	return OAuthRequest{ClientName: c.Name, RedirectHost: u.Hostname(), Scope: normalScope(scope)}, nil
}

// ApproveOAuth is the person's yes. PKCE is required, so a stolen code alone is worthless.
func (s *Service) ApproveOAuth(ctx context.Context, v Viewer, in OAuthApproval) (OAuthRedirect, error) {
	if in.CodeChallenge == "" {
		return OAuthRedirect{}, oauthError("invalid_request", "This app didn't use a secure sign-in")
	}
	code, err := auth.NewToken()
	if err != nil {
		return OAuthRedirect{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.oauthClient(ctx, q, in.ClientID, in.RedirectURI); err != nil {
			return err
		}
		if err := q.DeleteExpiredOAuthCodes(ctx, s.stamp(s.Now())); err != nil {
			return err
		}
		return q.CreateOAuthCode(ctx, sqlcgen.CreateOAuthCodeParams{CodeHash: auth.HashToken(code), ClientID: in.ClientID, UserID: v.id(), RedirectUri: in.RedirectURI, Challenge: in.CodeChallenge, Scope: normalScope(in.Scope), ExpiresAt: s.stamp(s.Now().Add(oauthCodeLifetime))})
	})
	if err != nil {
		return OAuthRedirect{}, err
	}
	u, err := url.Parse(in.RedirectURI)
	if err != nil {
		return OAuthRedirect{}, oauthError("invalid_redirect_uri", "This redirect address isn't allowed")
	}
	query := u.Query()
	query.Set("code", code)
	if in.State != "" {
		query.Set("state", in.State)
	}
	u.RawQuery = query.Encode()
	return OAuthRedirect{RedirectURL: u.String()}, nil
}

// OAuthTokens is the token endpoint's answer.
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (s *Service) issueOAuthTokens(ctx context.Context, q *sqlcgen.Queries, clientID string, userID int64, scope string) (OAuthTokens, error) {
	c, err := q.GetOAuthClient(ctx, clientID)
	if err != nil {
		return OAuthTokens{}, err
	}
	access, err := auth.NewToken()
	if err != nil {
		return OAuthTokens{}, err
	}
	refresh, err := auth.NewToken()
	if err != nil {
		return OAuthTokens{}, err
	}
	access, refresh = tokenPrefix+access, tokenPrefix+refresh
	now := s.Now()
	if err := q.DeleteExpiredAPITokens(ctx, nullStr(s.stamp(now))); err != nil {
		return OAuthTokens{}, err
	}
	// Each app shows once in settings, however often its short-lived token renews.
	if err := q.DeleteAPITokensForClient(ctx, sqlcgen.DeleteAPITokensForClientParams{UserID: userID, ClientID: clientID}); err != nil {
		return OAuthTokens{}, err
	}
	if _, err := q.CreateAPIToken(ctx, sqlcgen.CreateAPITokenParams{UserID: userID, Name: c.Name, TokenHash: auth.HashToken(access), Scope: scope, ClientID: clientID, ExpiresAt: nullStr(s.stamp(now.Add(oauthAccessLife)))}); err != nil {
		return OAuthTokens{}, err
	}
	if err := q.CreateOAuthRefresh(ctx, sqlcgen.CreateOAuthRefreshParams{TokenHash: auth.HashToken(refresh), ClientID: clientID, UserID: userID, Scope: scope, ExpiresAt: s.stamp(now.Add(oauthRefreshLife))}); err != nil {
		return OAuthTokens{}, err
	}
	return OAuthTokens{AccessToken: access, TokenType: "Bearer", ExpiresIn: int64(oauthAccessLife.Seconds()), RefreshToken: refresh, Scope: scope}, nil
}

func pkceMatches(challenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

// ExchangeOAuthCode trades a code for tokens. A code works once.
func (s *Service) ExchangeOAuthCode(ctx context.Context, clientID, code, redirect, verifier string) (OAuthTokens, error) {
	var out OAuthTokens
	bad := oauthError("invalid_grant", "This code is invalid or has expired")
	var denied *Error
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		hash := auth.HashToken(code)
		row, err := q.GetOAuthCode(ctx, hash)
		if isNoRows(err) {
			denied = bad
			return nil
		}
		if err != nil {
			return err
		}
		// Whatever happens next, this code is spent.
		if err := q.DeleteOAuthCode(ctx, hash); err != nil {
			return err
		}
		if row.ExpiresAt <= s.stamp(s.Now()) || row.ClientID != clientID || !redirectMatches(row.RedirectUri, redirect) || !pkceMatches(row.Challenge, verifier) {
			denied = bad
			return nil
		}
		out, err = s.issueOAuthTokens(ctx, q, clientID, row.UserID, row.Scope)
		return err
	})
	if err != nil {
		return OAuthTokens{}, err
	}
	if denied != nil {
		return OAuthTokens{}, denied
	}
	return out, nil
}

// RefreshOAuth replaces a refresh token with a new one.
func (s *Service) RefreshOAuth(ctx context.Context, clientID, refresh string) (OAuthTokens, error) {
	var out OAuthTokens
	bad := oauthError("invalid_grant", "This token is invalid or has expired")
	var denied *Error
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		hash := auth.HashToken(refresh)
		row, err := q.GetOAuthRefresh(ctx, hash)
		if isNoRows(err) {
			denied = bad
			return nil
		}
		if err != nil {
			return err
		}
		if err := q.DeleteOAuthRefresh(ctx, hash); err != nil {
			return err
		}
		if row.ExpiresAt <= s.stamp(s.Now()) || row.ClientID != clientID {
			denied = bad
			return nil
		}
		u, err := q.GetUser(ctx, row.UserID)
		if err != nil {
			return err
		}
		if u.DisabledAt.Valid {
			denied = bad
			return nil
		}
		out, err = s.issueOAuthTokens(ctx, q, clientID, row.UserID, row.Scope)
		return err
	})
	if err != nil {
		return OAuthTokens{}, err
	}
	if denied != nil {
		return OAuthTokens{}, denied
	}
	return out, nil
}
