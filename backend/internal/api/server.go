// Package api is the HTTP layer of the web app: routing, sessions and JSON. All decisions
// about data and permissions live in package app.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/assistant"
	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/kitshn"
	"github.com/serraniel/wokko/backend/internal/text"
	"github.com/serraniel/wokko/backend/internal/web"
)

// SessionCookie is shared with the kitshn layer, which must use this exact name.
const SessionCookie = kitshn.SessionCookie

type Server struct {
	svc          *app.Service
	log          *slog.Logger
	loginLimit   *auth.Limiter
	handleLimit  *auth.Limiter
	importLimit  *auth.Limiter
	secureCookie bool
}

func New(svc *app.Service, log *slog.Logger) *Server {
	return &Server{
		svc:          svc,
		log:          log,
		loginLimit:   auth.NewLimiter(10, 15*time.Minute),
		handleLimit:  auth.NewLimiter(30, time.Hour),
		importLimit:  auth.NewLimiter(60, time.Hour),
		secureCookie: strings.HasPrefix(svc.Cfg.PublicURL, "https://"),
	}
}

type ctxKey int

const viewerKey ctxKey = 0

// Handler returns the complete application, API and frontend included.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	kitshn.New(s.svc, s.log, s.loginLimit, s.secureCookie).Routes(mux)
	assistant.New(s.svc, s.log).Routes(mux)
	mux.Handle("/", web.Handler())
	return s.recoverer(s.securityHeaders(s.originCheck(mux)))
}

func (s *Server) routes(m *http.ServeMux) {
	const p = "/api/v1"
	m.HandleFunc("GET "+p+"/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	m.HandleFunc("GET "+p+"/status", s.h(s.status))
	m.HandleFunc("POST "+p+"/setup", s.h(s.setup))
	m.HandleFunc("POST "+p+"/auth/login", s.h(s.login))
	m.HandleFunc("POST "+p+"/auth/register", s.h(s.register))
	m.HandleFunc("POST "+p+"/auth/logout", s.h(s.logout))
	m.HandleFunc("POST "+p+"/auth/reset", s.h(s.resetPassword))

	m.HandleFunc("GET "+p+"/me", s.user(s.me))
	m.HandleFunc("PATCH "+p+"/me", s.user(s.updateMe))
	m.HandleFunc("POST "+p+"/me/password", s.user(s.changePassword))
	m.HandleFunc("GET "+p+"/me/export", s.user(s.export))

	m.HandleFunc("GET "+p+"/recipes", s.user(s.listRecipes))
	m.HandleFunc("POST "+p+"/recipes", s.user(s.createRecipe))
	m.HandleFunc("POST "+p+"/import", s.user(s.importRecipe))
	m.HandleFunc("POST "+p+"/import/file", s.user(s.importFile))
	m.HandleFunc("GET "+p+"/recipes/{id}", s.user(s.getRecipe))
	m.HandleFunc("PUT "+p+"/recipes/{id}", s.user(s.updateRecipe))
	m.HandleFunc("DELETE "+p+"/recipes/{id}", s.user(s.deleteRecipe))
	m.HandleFunc("POST "+p+"/recipes/{id}/restore", s.user(s.restoreRecipe))
	m.HandleFunc("POST "+p+"/recipes/{id}/photo", s.user(s.uploadPhoto))
	m.HandleFunc("DELETE "+p+"/recipes/{id}/photo", s.user(s.deletePhoto))
	m.HandleFunc("GET "+p+"/recipes/{id}/versions", s.user(s.listVersions))
	m.HandleFunc("POST "+p+"/recipes/{id}/versions/{vid}/restore", s.user(s.restoreVersion))
	m.HandleFunc("GET "+p+"/recipes/{id}/sharing", s.user(s.getSharing))
	m.HandleFunc("POST "+p+"/recipes/{id}/sharing/users", s.user(s.shareUser))
	m.HandleFunc("DELETE "+p+"/recipes/{id}/sharing/users/{uid}", s.user(s.unshareUser))
	m.HandleFunc("POST "+p+"/recipes/{id}/sharing/links", s.user(s.createLink))
	m.HandleFunc("DELETE "+p+"/recipes/{id}/sharing/links/{lid}", s.user(s.revokeLink))
	m.HandleFunc("GET "+p+"/recipes/{id}/cooklog", s.user(s.listCookLog))
	m.HandleFunc("POST "+p+"/recipes/{id}/cooklog", s.user(s.addCookLog))
	m.HandleFunc("GET "+p+"/recipes/{id}/books", s.user(s.recipeBooks))
	m.HandleFunc("GET "+p+"/books", s.user(s.listBooks))
	m.HandleFunc("POST "+p+"/books", s.user(s.createBook))
	m.HandleFunc("GET "+p+"/books/{id}", s.user(s.getBook))
	m.HandleFunc("PUT "+p+"/books/{id}", s.user(s.updateBook))
	m.HandleFunc("DELETE "+p+"/books/{id}", s.user(s.deleteBook))
	m.HandleFunc("PUT "+p+"/books/{id}/recipes/{rid}", s.user(s.addToBook))
	m.HandleFunc("DELETE "+p+"/books/{id}/recipes/{rid}", s.user(s.removeFromBook))
	m.HandleFunc("POST "+p+"/auth/passkey/begin", s.h(s.passkeyLoginBegin))
	m.HandleFunc("POST "+p+"/auth/passkey/finish", s.h(s.passkeyLoginFinish))
	m.HandleFunc("GET "+p+"/me/passkeys", s.user(s.listPasskeys))
	m.HandleFunc("POST "+p+"/me/passkeys/begin", s.user(s.passkeyAddBegin))
	m.HandleFunc("POST "+p+"/me/passkeys/finish", s.user(s.passkeyAddFinish))
	m.HandleFunc("DELETE "+p+"/me/passkeys/{id}", s.user(s.deletePasskey))
	m.HandleFunc("GET "+p+"/me/tokens", s.user(s.listTokens))
	m.HandleFunc("POST "+p+"/me/tokens", s.user(s.createToken))
	m.HandleFunc("DELETE "+p+"/me/tokens/{id}", s.user(s.revokeToken))
	m.HandleFunc("GET "+p+"/me/assistant-activity", s.user(s.assistantActivity))
	m.HandleFunc("GET "+p+"/oauth/request", s.user(s.oauthRequest))
	m.HandleFunc("POST "+p+"/oauth/approve", s.user(s.oauthApprove))
	m.HandleFunc("GET "+p+"/me/collection-links", s.user(s.listCollectionLinks))
	m.HandleFunc("POST "+p+"/me/collection-links", s.user(s.createCollectionLink))
	m.HandleFunc("DELETE "+p+"/me/collection-links/{id}", s.user(s.revokeCollectionLink))
	m.HandleFunc("GET "+p+"/trash", s.user(s.listTrash))
	m.HandleFunc("DELETE "+p+"/trash/{id}", s.user(s.purgeRecipe))

	m.HandleFunc("GET "+p+"/foods", s.user(s.listFoods))
	m.HandleFunc("POST "+p+"/foods", s.user(s.createFood))
	m.HandleFunc("POST "+p+"/foods/merge", s.user(s.mergeFoods))
	m.HandleFunc("GET "+p+"/foods/{id}", s.user(s.getFood))
	m.HandleFunc("PUT "+p+"/foods/{id}", s.user(s.updateFood))
	m.HandleFunc("DELETE "+p+"/foods/{id}", s.user(s.deleteFood))
	m.HandleFunc("GET "+p+"/units", s.user(s.listUnits))
	m.HandleFunc("POST "+p+"/units", s.user(s.createUnit))
	m.HandleFunc("POST "+p+"/units/merge", s.user(s.mergeUnits))
	m.HandleFunc("GET "+p+"/units/{id}", s.user(s.getUnit))
	m.HandleFunc("PUT "+p+"/units/{id}", s.user(s.updateUnit))
	m.HandleFunc("DELETE "+p+"/units/{id}", s.user(s.deleteUnit))
	m.HandleFunc("GET "+p+"/keywords", s.user(s.listKeywords))

	m.HandleFunc("GET "+p+"/friends", s.user(s.listFriends))
	m.HandleFunc("POST "+p+"/friends", s.user(s.requestFriend))
	m.HandleFunc("POST "+p+"/friends/{id}/accept", s.user(s.acceptFriend))
	m.HandleFunc("DELETE "+p+"/friends/{id}", s.user(s.removeFriend))

	m.HandleFunc("GET "+p+"/household", s.user(s.getHousehold))
	m.HandleFunc("PATCH "+p+"/household", s.user(s.renameHousehold))
	m.HandleFunc("POST "+p+"/household/invites", s.user(s.createHouseholdInvite))
	m.HandleFunc("DELETE "+p+"/household/invites/{id}", s.user(s.deleteInvite))
	m.HandleFunc("POST "+p+"/household/join", s.user(s.joinHousehold))
	m.HandleFunc("POST "+p+"/household/leave", s.user(s.leaveHousehold))

	m.HandleFunc("GET "+p+"/admin/users", s.user(s.adminListUsers))
	m.HandleFunc("PATCH "+p+"/admin/users/{id}", s.user(s.adminUpdateUser))
	m.HandleFunc("DELETE "+p+"/admin/users/{id}", s.user(s.adminDeleteUser))
	m.HandleFunc("POST "+p+"/admin/users/{id}/reset", s.user(s.adminResetLink))
	m.HandleFunc("GET "+p+"/admin/invites", s.user(s.adminListInvites))
	m.HandleFunc("POST "+p+"/admin/invites", s.user(s.adminCreateInvite))
	m.HandleFunc("DELETE "+p+"/admin/invites/{id}", s.user(s.deleteInvite))
	m.HandleFunc("GET "+p+"/admin/settings", s.user(s.adminGetSettings))
	m.HandleFunc("PUT "+p+"/admin/settings", s.user(s.adminUpdateSettings))
	m.HandleFunc("GET "+p+"/admin/health", s.user(s.adminHealth))
	m.HandleFunc("POST "+p+"/admin/backups", s.user(s.adminBackup))

	s.planningRoutes(m, p)

	m.HandleFunc("GET "+p+"/public/recipes", s.h(s.listPublic))
	m.HandleFunc("GET "+p+"/public/recipes/{id}", s.h(s.getPublic))
	m.HandleFunc("GET "+p+"/public/share/{token}", s.h(s.getShared))
	m.HandleFunc("GET "+p+"/public/collections/{token}", s.h(s.listCollection))
	m.HandleFunc("GET "+p+"/public/collections/{token}/recipes/{id}", s.h(s.getCollectionRecipe))
	m.HandleFunc("GET /media/{file}", s.h(s.serveMedia))
	// Unknown API paths get JSON, never the web app's HTML.
	m.HandleFunc(p+"/", s.h(func(http.ResponseWriter, *http.Request) error { return app.ErrNotFound }))
}

// --- plumbing ---

type handler func(w http.ResponseWriter, r *http.Request) error

func (s *Server) h(fn handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			s.writeError(w, r, err)
		}
	}
}

type userHandler func(w http.ResponseWriter, r *http.Request, v app.Viewer) error

// user requires a logged-in, non-disabled user.
func (s *Server) user(fn userHandler) http.HandlerFunc {
	return s.h(func(w http.ResponseWriter, r *http.Request) error {
		u, err := s.svc.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			return err
		}
		locale := u.Locale
		if q := r.URL.Query().Get("lang"); text.ValidLocale(q) {
			locale = q
		}
		return fn(w, r, app.Viewer{User: u, Locale: locale})
	})
}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(SessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *app.Error
	switch {
	case errors.As(err, &ae):
		writeJSON(w, ae.Status, map[string]any{"error": ae})
	case errors.Is(err, context.Canceled):
		// The client went away. There is nobody left to answer.
	default:
		s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "server_error", "message": "Something went wrong on the server. Your data was not changed"}})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the status is already sent; a broken connection can't be reported
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	if err := dec.Decode(v); err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "bad_request", Message: "The request could not be read"}
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, app.ErrNotFound
	}
	return id, nil
}

func queryInt(r *http.Request, name string) int64 {
	n, err := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (s *Server) anonLocale(r *http.Request) string {
	if q := r.URL.Query().Get("lang"); text.ValidLocale(q) {
		return q
	}
	return text.PickLocale(r.Header.Get("Accept-Language"), "en")
}

func (s *Server) setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, kitshn.NewSessionCookie(token, s.secureCookie))
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer") // share links must not leak through Referer
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// originCheck blocks cross-site form posts and scripts. kitshn sends no Origin header.
func (s *Server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// MCP and OAuth's token calls carry their own credentials instead of cookies, so a
		// cross-site page gains nothing from them.
		machine := r.URL.Path == "/mcp" || r.URL.Path == "/oauth/token" || r.URL.Path == "/oauth/register"
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !machine {
			if o := r.Header.Get("Origin"); o != "" {
				u, err := url.Parse(o)
				if err != nil || u.Host != r.Host {
					writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "bad_origin", "message": "Cross-site request blocked"}})
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic while handling request", "method", r.Method, "path", r.URL.Path, "panic", rec)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "server_error", "message": "Something went wrong on the server"}})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func clientKey(r *http.Request, handle string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host + "|" + strings.ToLower(handle)
}

var errTooMany = &app.Error{Status: http.StatusTooManyRequests, Code: "too_many_attempts", Message: "Too many attempts. Please wait a while and try again"}
