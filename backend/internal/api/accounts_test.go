package api_test

import (
	"testing"
	"time"

	"github.com/serraniel/wokko/backend/internal/app"
)

// TestAdminsCreateAccountsWithASetupLink: the person sets their own password through the link,
// so the admin never picks or sees it.
func TestAdminsCreateAccountsWithASetupLink(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	var me app.User
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &me)

	var made app.NewAccount
	admin.ok("POST", v1+"/admin/users", app.NewAccountInput{Handle: "oma", DisplayName: "Oma", Locale: "de", JoinMyHousehold: true}, 201).decode(t, &made)
	if made.Token == "" || !made.User.NeedsSetup || made.User.HouseholdID != me.HouseholdID || made.User.Role != "user" || made.User.Locale != "de" {
		t.Fatalf("made: %+v", made)
	}
	var users []app.User
	admin.ok("GET", v1+"/admin/users", nil, 200).decode(t, &users)
	waiting := false
	for _, u := range users {
		waiting = waiting || (u.Handle == "oma" && u.NeedsSetup)
	}
	if !waiting {
		t.Fatalf("not waiting for setup: %+v", users)
	}
	e.anon().wantCode("POST", v1+"/auth/login", map[string]string{"handle": "oma", "password": "password123"}, 401, "bad_login")

	e.anon().ok("POST", v1+"/auth/reset", map[string]string{"token": made.Token, "password": "omas-secret"}, 200)
	e.anon().ok("POST", v1+"/auth/login", map[string]string{"handle": "oma", "password": "omas-secret"}, 200)
	e.anon().wantCode("POST", v1+"/auth/reset", map[string]string{"token": made.Token, "password": "again-secret"}, 403, "invalid_reset")
	admin.ok("GET", v1+"/admin/users", nil, 200).decode(t, &users)
	for _, u := range users {
		if u.Handle == "oma" && u.NeedsSetup {
			t.Fatal("still waiting for setup")
		}
	}

	// A setup link lasts a week, then a new one is needed.
	admin.ok("POST", v1+"/admin/users", app.NewAccountInput{Handle: "opa"}, 201).decode(t, &made)
	if made.User.HouseholdID == me.HouseholdID {
		t.Fatal("should get its own household")
	}
	real := e.svc.Now
	e.svc.Now = func() time.Time { return real().Add(8 * 24 * time.Hour) }
	e.anon().wantCode("POST", v1+"/auth/reset", map[string]string{"token": made.Token, "password": "opas-secret"}, 403, "invalid_reset")
	e.svc.Now = real

	admin.wantCode("POST", v1+"/admin/users", app.NewAccountInput{Handle: "oma"}, 409, "handle_taken")
	admin.wantCode("POST", v1+"/admin/users", app.NewAccountInput{Handle: "x"}, 400, "bad_handle")
	other := e.newUser(admin, "other")
	other.wantCode("POST", v1+"/admin/users", app.NewAccountInput{Handle: "sneaky"}, 403, "forbidden")

	// Invite links can carry a note saying who they're for.
	var inv app.Invite
	admin.ok("POST", v1+"/admin/invites", app.InviteInput{MaxUses: 1, Note: "For Oma's neighbour"}, 201).decode(t, &inv)
	var invites []app.Invite
	admin.ok("GET", v1+"/admin/invites", nil, 200).decode(t, &invites)
	if len(invites) == 0 || invites[0].Note != "For Oma's neighbour" {
		t.Fatalf("invites: %+v", invites)
	}
}
