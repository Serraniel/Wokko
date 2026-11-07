package api

import (
	"net/http"
	"strings"

	"github.com/serraniel/wokko/backend/internal/app"
)

// origin is the address the browser uses. originCheck has already compared it to the host.
func origin(r *http.Request) string {
	if o := r.Header.Get("Origin"); o != "" {
		return o
	}
	return strings.TrimRight(r.Header.Get("Referer"), "/")
}

type passkeyStart struct {
	Flow    string `json:"flow"`
	Options any    `json:"options"`
}

func (s *Server) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) error {
	if !s.loginLimit.Allow(clientKey(r, "passkey")) {
		return errTooMany
	}
	flow, options, err := s.svc.BeginPasskeyLogin(r.Context(), origin(r))
	return reply(w, passkeyStart{flow, options}, err)
}

func (s *Server) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) error {
	u, token, err := s.svc.FinishPasskeyLogin(r.Context(), r.URL.Query().Get("flow"), origin(r), r)
	if err != nil {
		return err
	}
	s.setSession(w, token)
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) listPasskeys(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListPasskeys(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) passkeyAddBegin(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	flow, options, err := s.svc.BeginPasskeyRegistration(r.Context(), v, origin(r))
	return reply(w, passkeyStart{flow, options}, err)
}

func (s *Server) passkeyAddFinish(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	q := r.URL.Query()
	out, err := s.svc.FinishPasskeyRegistration(r.Context(), v, q.Get("flow"), q.Get("name"), origin(r), r)
	return replyCreated(w, out, err)
}

func (s *Server) deletePasskey(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeletePasskey(r.Context(), v, id) })
}

func (s *Server) listTokens(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListAPITokens(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) createToken(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.APITokenInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.CreateAPIToken(r.Context(), v, in)
	return replyCreated(w, out, err)
}

func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.RevokeAPIToken(r.Context(), v, id) })
}

func (s *Server) assistantActivity(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListAssistantActivity(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) oauthRequest(w http.ResponseWriter, r *http.Request, _ app.Viewer) error {
	q := r.URL.Query()
	out, err := s.svc.DescribeOAuthRequest(r.Context(), q.Get("client_id"), q.Get("redirect_uri"), q.Get("scope"))
	return reply(w, out, err)
}

func (s *Server) oauthApprove(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.OAuthApproval
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.ApproveOAuth(r.Context(), v, in)
	return reply(w, out, err)
}
