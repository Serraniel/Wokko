package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/serraniel/wokko/backend/internal/app"
)

// passkeySend talks like a browser at the site's public address.
func passkeySend(t *testing.T, c *client, path, body string) result {
	t.Helper()
	req := mustReq(t, "POST", c.e.srv.URL+path, body)
	req.Host = "wokko.test"
	req.Header.Set("Origin", "http://wokko.test")
	req.Header.Set("Content-Type", "application/json")
	// With the Host changed, the cookie jar would look under that name, so this does it by hand.
	address := req.URL
	for _, cookie := range c.http.Jar.Cookies(address) {
		req.AddCookie(cookie)
	}
	res := c.send(req)
	c.http.Jar.SetCookies(address, (&http.Response{Header: res.header}).Cookies())
	return res
}

func TestPasskeysRegisterAndLogIn(t *testing.T) {
	e := newEnv(t) // PublicURL is http://wokko.test
	admin := e.admin()
	rp := virtualwebauthn.RelyingParty{Name: "Wokko", ID: "wokko.test", Origin: "http://wokko.test"}
	authenticator := virtualwebauthn.NewAuthenticator()
	cred := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	// Register.
	res := passkeySend(t, admin, v1+"/me/passkeys/begin", "")
	if res.status != 200 {
		t.Fatalf("begin: %d %s", res.status, res.body)
	}
	var start struct {
		Flow    string          `json:"flow"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(res.body, &start); err != nil {
		t.Fatal(err)
	}
	attestation, err := virtualwebauthn.ParseAttestationOptions(string(start.Options))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(start.Options), `"residentKey":"required"`) {
		t.Fatalf("discoverable keys are what make login without a handle work: %s", start.Options)
	}
	authenticator.Options.UserHandle = []byte(attestation.UserID)
	answer := virtualwebauthn.CreateAttestationResponse(rp, authenticator, cred, *attestation)
	res = passkeySend(t, admin, v1+"/me/passkeys/finish?name=Phone&flow="+start.Flow, answer)
	if res.status != 201 {
		t.Fatalf("finish: %d %s", res.status, res.body)
	}
	authenticator.AddCredential(cred)
	var keys []app.Passkey
	admin.ok("GET", v1+"/me/passkeys", nil, 200).decode(t, &keys)
	if len(keys) != 1 || keys[0].Name != "Phone" {
		t.Fatalf("keys: %+v", keys)
	}
	// A flow only works once.
	if res := passkeySend(t, admin, v1+"/me/passkeys/finish?flow="+start.Flow, answer); res.status != 400 {
		t.Fatalf("replay: %d", res.status)
	}

	login := func(c *client) result {
		t.Helper()
		res := passkeySend(t, c, v1+"/auth/passkey/begin", "")
		if res.status != 200 {
			t.Fatalf("login begin: %d %s", res.status, res.body)
		}
		var start struct {
			Flow    string          `json:"flow"`
			Options json.RawMessage `json:"options"`
		}
		if err := json.Unmarshal(res.body, &start); err != nil {
			t.Fatal(err)
		}
		assertion, err := virtualwebauthn.ParseAssertionOptions(string(start.Options))
		if err != nil {
			t.Fatal(err)
		}
		answer := virtualwebauthn.CreateAssertionResponse(rp, authenticator, cred, *assertion)
		return passkeySend(t, c, v1+"/auth/passkey/finish?flow="+start.Flow, answer)
	}
	fresh := e.anon()
	if res := login(fresh); res.status != 200 {
		t.Fatalf("login: %d %s", res.status, res.body)
	}
	var me app.User
	fresh.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	if me.Handle != "admin" {
		t.Fatalf("me: %+v", me)
	}
	admin.ok("GET", v1+"/me/passkeys", nil, 200).decode(t, &keys)
	if keys[0].LastUsedAt == "" {
		t.Fatalf("last used: %+v", keys[0])
	}

	// A wrong flow gets nowhere, and neither does a deleted key.
	if res := passkeySend(t, e.anon(), v1+"/auth/passkey/finish?flow=nonsense", "{}"); res.status != 400 {
		t.Fatalf("bad flow: %d", res.status)
	}
	other := e.newUser(admin, "other")
	other.wantCode("DELETE", path("/me/passkeys/%d", keys[0].ID), nil, 404, "not_found")
	admin.ok("DELETE", path("/me/passkeys/%d", keys[0].ID), nil, 200)
	if res := login(e.anon()); res.status != 400 {
		t.Fatalf("deleted key: %d", res.status)
	}
}
