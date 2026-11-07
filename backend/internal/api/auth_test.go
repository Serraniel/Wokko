package api_test

import (
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

func TestSetupOnlyWorksOnce(t *testing.T) {
	e := newEnv(t)
	var st app.Status
	e.anon().ok("GET", v1+"/status", nil, 200).decode(t, &st)
	if st.SetupComplete {
		t.Fatal("new instance must need setup")
	}
	e.anon().wantCode("POST", v1+"/setup", map[string]string{"handle": "x", "password": "password123"}, 400, "bad_handle")
	e.anon().wantCode("POST", v1+"/setup", map[string]string{"handle": "admin", "password": "short"}, 400, "bad_password")
	admin := e.admin()
	e.anon().wantCode("POST", v1+"/setup", map[string]string{"handle": "second", "password": "password123"}, 409, "already_setup")
	var me app.User
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	if me.Role != "admin" {
		t.Fatalf("first user must be admin, got %q", me.Role)
	}
}

func TestLoginLogoutAndRateLimit(t *testing.T) {
	e := newEnv(t)
	e.admin()
	c := e.anon()
	c.wantCode("GET", v1+"/me", nil, 401, "unauthorized")
	c.wantCode("POST", v1+"/auth/login", map[string]string{"handle": "admin", "password": "wrong-password"}, 401, "bad_login")
	c.wantCode("POST", v1+"/auth/login", map[string]string{"handle": "nobody", "password": "wrong-password"}, 401, "bad_login")
	c.ok("POST", v1+"/auth/login", map[string]string{"handle": "ADMIN", "password": "password123"}, 200)
	c.ok("GET", v1+"/me", nil, 200)
	c.ok("POST", v1+"/auth/logout", nil, 200)
	c.wantCode("GET", v1+"/me", nil, 401, "unauthorized")

	for range 10 {
		c.do("POST", v1+"/auth/login", map[string]string{"handle": "target", "password": "wrong-password"})
	}
	c.wantCode("POST", v1+"/auth/login", map[string]string{"handle": "target", "password": "wrong-password"}, 429, "too_many_attempts")
}

func TestSessionCookieIsHttpOnly(t *testing.T) {
	e := newEnv(t)
	e.admin()
	r := e.anon().ok("POST", v1+"/auth/login", map[string]string{"handle": "admin", "password": "password123"}, 200)
	got := r.header.Get("Set-Cookie")
	for _, want := range []string{"sessionid=", "HttpOnly", "SameSite=Lax"} {
		if !contains(got, want) {
			t.Fatalf("cookie %q is missing %q", got, want)
		}
	}
}

func TestCrossSiteRequestsAreBlocked(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	req := mustReq(t, "POST", e.srv.URL+v1+"/recipes", `{"name":"evil"}`)
	req.Header.Set("Origin", "https://evil.example")
	if r := c.send(req); r.status != 403 || r.code(t) != "bad_origin" {
		t.Fatalf("got %d: %s", r.status, r.body)
	}
}

func TestRegistrationNeedsInviteUnlessOpen(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	in := map[string]string{"handle": "newbie", "password": "password123"}
	e.anon().wantCode("POST", v1+"/auth/register", in, 403, "invite_required")

	// The admin toggle takes effect, unlike in the old app.
	admin.ok("PUT", v1+"/admin/settings", app.Settings{OpenRegistration: true, AllowPublic: true}, 200)
	e.anon().ok("POST", v1+"/auth/register", in, 201)
	e.anon().wantCode("POST", v1+"/auth/register", in, 409, "handle_taken")
	admin.ok("PUT", v1+"/admin/settings", app.Settings{OpenRegistration: false, AllowPublic: true}, 200)
	var st app.Settings
	admin.ok("GET", v1+"/admin/settings", nil, 200).decode(t, &st)
	if st.OpenRegistration {
		t.Fatal("setting was not saved")
	}

	var inv app.Invite
	admin.ok("POST", v1+"/admin/invites", map[string]any{"max_uses": 1}, 201).decode(t, &inv)
	e.anon().wantCode("POST", v1+"/auth/register", map[string]string{"handle": "bad-token", "password": "password123", "invite": "nope"}, 403, "invalid_invite")
	e.anon().ok("POST", v1+"/auth/register", map[string]string{"handle": "invited", "password": "password123", "invite": inv.Token}, 201)
	e.anon().wantCode("POST", v1+"/auth/register", map[string]string{"handle": "again", "password": "password123", "invite": inv.Token}, 403, "invalid_invite")

	var list []app.Invite
	admin.ok("GET", v1+"/admin/invites", nil, 200).decode(t, &list)
	if len(list) != 1 || list[0].UseCount != 1 || list[0].Token != "" {
		t.Fatalf("invite list wrong: %+v", list)
	}
	admin.ok("DELETE", path("/admin/invites/%d", inv.ID), nil, 200)
}

func TestDisabledUserIsLockedOutImmediately(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "alice")
	var me app.User
	u.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	admin.ok("PATCH", path("/admin/users/%d", me.ID), map[string]bool{"disabled": true}, 200)
	u.wantCode("GET", v1+"/me", nil, 401, "unauthorized")
	e.anon().wantCode("POST", v1+"/auth/login", map[string]string{"handle": "alice", "password": "password123"}, 401, "bad_login")
	admin.ok("PATCH", path("/admin/users/%d", me.ID), map[string]bool{"disabled": false}, 200)
	e.anon().ok("POST", v1+"/auth/login", map[string]string{"handle": "alice", "password": "password123"}, 200)
}

func TestLastAdminIsProtected(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	var me app.User
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	admin.wantCode("PATCH", path("/admin/users/%d", me.ID), map[string]bool{"disabled": true}, 409, "last_admin")
	admin.wantCode("PATCH", path("/admin/users/%d", me.ID), map[string]string{"role": "user"}, 409, "last_admin")
	admin.wantCode("DELETE", path("/admin/users/%d", me.ID), nil, 409, "cannot_delete_self")
}

func TestProfileAndPassword(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	var me app.User
	c.ok("PATCH", v1+"/me", map[string]string{"display_name": "Chef", "locale": "fr"}, 200).decode(t, &me)
	if me.DisplayName != "Chef" || me.Locale != "fr" {
		t.Fatalf("profile not updated: %+v", me)
	}
	c.wantCode("PATCH", v1+"/me", map[string]string{"locale": "not a locale"}, 400, "bad_locale")
	c.wantCode("POST", v1+"/me/password", map[string]string{"current": "wrong", "next": "new-password-1"}, 401, "bad_login")
	other := e.anon()
	other.ok("POST", v1+"/auth/login", map[string]string{"handle": "admin", "password": "password123"}, 200)
	c.ok("POST", v1+"/me/password", map[string]string{"current": "password123", "next": "new-password-1"}, 200)
	c.ok("GET", v1+"/me", nil, 200) // this device stays logged in
	other.wantCode("GET", v1+"/me", nil, 401, "unauthorized")
	e.anon().ok("POST", v1+"/auth/login", map[string]string{"handle": "admin", "password": "new-password-1"}, 200)
}

func TestAdminResetLinkRecoversAccountWithoutEmail(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "forgetful")
	var me app.User
	u.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	u.wantCode("POST", path("/admin/users/%d/reset", me.ID), nil, 403, "forbidden")
	var res struct {
		Token string `json:"token"`
	}
	admin.ok("POST", path("/admin/users/%d/reset", me.ID), nil, 201).decode(t, &res)
	e.anon().ok("POST", v1+"/auth/reset", map[string]string{"token": res.Token, "password": "brand-new-pass"}, 200)
	e.anon().wantCode("POST", v1+"/auth/reset", map[string]string{"token": res.Token, "password": "other-new-pass"}, 403, "invalid_reset")
	e.anon().ok("POST", v1+"/auth/login", map[string]string{"handle": "forgetful", "password": "brand-new-pass"}, 200)
	u.wantCode("GET", v1+"/me", nil, 401, "unauthorized")
}

func TestAdminEndpointsRejectNormalUsers(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "alice")
	for _, c := range []struct{ m, p string }{
		{"GET", "/admin/users"}, {"PATCH", "/admin/users/1"}, {"DELETE", "/admin/users/1"}, {"GET", "/admin/invites"}, {"POST", "/admin/invites"},
		{"GET", "/admin/settings"}, {"PUT", "/admin/settings"}, {"GET", "/admin/health"}, {"POST", "/admin/backups"},
	} {
		u.wantCode(c.m, v1+c.p, map[string]any{}, 403, "forbidden")
	}
	e.anon().wantCode("GET", v1+"/admin/users", nil, 401, "unauthorized")
}

func TestUnknownAPIPathIsJSON(t *testing.T) {
	e := newEnv(t)
	r := e.anon().do("GET", v1+"/does-not-exist", nil)
	if r.status != 404 || r.code(t) != "not_found" {
		t.Fatalf("got %d: %s", r.status, r.body)
	}
}

func TestHealthEndpoint(t *testing.T) {
	newEnv(t).anon().ok("GET", v1+"/health", nil, 200)
}
