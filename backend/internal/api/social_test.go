package api_test

import (
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// Nobody can be found by typing part of a name. Only the complete, exact handle works.
func TestUsersCannotBeDiscovered(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	me := e.newUser(admin, "me-user")
	e.newUser(admin, "john.doe")
	r := me.createRecipe(simpleRecipe("Soup"))

	for _, partial := range []string{"john", "joh", "john.do", "doe", "%", "_ohn.doe", "john.doe%"} {
		me.wantCode("POST", path("/recipes/%d/sharing/users", r.ID), map[string]string{"handle": partial}, 404, "handle_not_found")
		me.wantCode("POST", v1+"/friends", map[string]string{"handle": partial}, 404, "handle_not_found")
	}
	var sharing app.Sharing
	me.ok("GET", path("/recipes/%d/sharing", r.ID), nil, 200).decode(t, &sharing)
	if len(sharing.Users) != 0 {
		t.Fatalf("partial handle shared something: %+v", sharing.Users)
	}

	// The exact handle works, with or without @, in any case.
	me.ok("POST", path("/recipes/%d/sharing/users", r.ID), map[string]string{"handle": "@John.Doe"}, 200)
	me.ok("GET", path("/recipes/%d/sharing", r.ID), nil, 200).decode(t, &sharing)
	if len(sharing.Users) != 1 || sharing.Users[0].Handle != "john.doe" {
		t.Fatalf("sharing: %+v", sharing.Users)
	}
}

func TestHandleLookupsAreRateLimited(t *testing.T) {
	e := newEnv(t)
	me := e.newUser(e.admin(), "me-user")
	for range 30 {
		me.do("POST", v1+"/friends", map[string]string{"handle": "guess"})
	}
	me.wantCode("POST", v1+"/friends", map[string]string{"handle": "guess"}, 429, "too_many_attempts")
}

func TestSharingWithAUserGivesReadOnlyAccess(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	owner := e.newUser(admin, "owner")
	friend := e.newUser(admin, "friend")
	r := owner.createRecipe(simpleRecipe("Soup"))
	friend.ok("GET", path("/recipes/%d", r.ID), nil, 404)

	owner.ok("POST", path("/recipes/%d/sharing/users", r.ID), map[string]string{"handle": "friend"}, 200)
	var got app.Recipe
	friend.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.CanEdit || got.CanManage {
		t.Fatalf("shared user must only read: %+v", got)
	}
	var list app.Page[app.RecipeSummary]
	friend.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 1 {
		t.Fatalf("shared recipe should be listed, got %d", list.Total)
	}
	in := simpleRecipe("hacked")
	in.BaseVersion = got.Version
	friend.wantCode("PUT", path("/recipes/%d", r.ID), in, 403, "forbidden")
	friend.wantCode("GET", path("/recipes/%d/sharing", r.ID), nil, 403, "forbidden")

	var friendUser app.User
	friend.ok("GET", v1+"/me", nil, 200).decode(t, &friendUser)
	owner.ok("DELETE", path("/recipes/%d/sharing/users/%d", r.ID, friendUser.ID), nil, 200)
	friend.ok("GET", path("/recipes/%d", r.ID), nil, 404)
}

func TestFriendRequests(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	a := e.newUser(admin, "alice")
	b := e.newUser(admin, "bob")
	a.wantCode("POST", v1+"/friends", map[string]string{"handle": "alice"}, 404, "handle_not_found")
	a.ok("POST", v1+"/friends", map[string]string{"handle": "bob"}, 200)

	var incoming []app.Friend
	b.ok("GET", v1+"/friends", nil, 200).decode(t, &incoming)
	if len(incoming) != 1 || !incoming[0].Incoming || incoming[0].Accepted || incoming[0].Person.Handle != "alice" {
		t.Fatalf("incoming: %+v", incoming)
	}
	var outgoing []app.Friend
	a.ok("GET", v1+"/friends", nil, 200).decode(t, &outgoing)
	a.wantCode("POST", path("/friends/%d/accept", outgoing[0].ID), nil, 404, "not_found") // can't accept your own request
	b.ok("POST", path("/friends/%d/accept", incoming[0].ID), nil, 200)
	a.ok("GET", v1+"/friends", nil, 200).decode(t, &outgoing)
	if !outgoing[0].Accepted {
		t.Fatal("not accepted")
	}
	c := e.newUser(admin, "carol")
	c.wantCode("DELETE", path("/friends/%d", outgoing[0].ID), nil, 404, "not_found")
	a.ok("DELETE", path("/friends/%d", outgoing[0].ID), nil, 200)
	b.ok("GET", v1+"/friends", nil, 200).decode(t, &incoming)
	if len(incoming) != 0 {
		t.Fatalf("friend not removed: %+v", incoming)
	}

	// Asking someone who already asked you accepts their request.
	a.ok("POST", v1+"/friends", map[string]string{"handle": "bob"}, 200)
	b.ok("POST", v1+"/friends", map[string]string{"handle": "alice"}, 200)
	a.ok("GET", v1+"/friends", nil, 200).decode(t, &outgoing)
	if !outgoing[0].Accepted {
		t.Fatal("mutual request should connect")
	}
}

func TestHousehold(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	a := e.newUser(admin, "alice")
	b := e.newUser(admin, "bob")
	a.ok("PATCH", v1+"/household", map[string]string{"name": "Home"}, 200)
	a.wantCode("PATCH", v1+"/household", map[string]string{"name": " "}, 400, "bad_name")

	var inv app.Invite
	a.ok("POST", v1+"/household/invites", map[string]any{"max_uses": 2}, 201).decode(t, &inv)
	b.wantCode("POST", v1+"/household/join", map[string]string{"token": "wrong"}, 403, "invalid_invite")
	b.ok("POST", v1+"/household/join", map[string]string{"token": inv.Token}, 200)
	var h app.Household
	b.ok("GET", v1+"/household", nil, 200).decode(t, &h)
	if h.Name != "Home" || len(h.Members) != 2 || len(h.Invites) != 1 {
		t.Fatalf("household: %+v", h)
	}

	// A household invite also works for someone without an account.
	c := e.anon()
	c.ok("POST", v1+"/auth/register", map[string]string{"handle": "carol", "password": "password123", "invite": inv.Token}, 201)
	c.ok("GET", v1+"/household", nil, 200).decode(t, &h)
	if h.Name != "Home" || len(h.Members) != 3 {
		t.Fatalf("household after register: %+v", h)
	}

	// Registration invites don't let you join a household.
	var reg app.Invite
	admin.ok("POST", v1+"/admin/invites", map[string]any{"max_uses": 5}, 201).decode(t, &reg)
	b.wantCode("POST", v1+"/household/join", map[string]string{"token": reg.Token}, 403, "invalid_invite")

	stranger := e.newUser(admin, "stranger")
	stranger.wantCode("DELETE", path("/household/invites/%d", inv.ID), nil, 403, "forbidden")
	a.ok("DELETE", path("/household/invites/%d", inv.ID), nil, 200)

	// Leaving keeps your recipes yours and doesn't touch those of others.
	r := b.createRecipe(simpleRecipe("Bob's soup"))
	b.ok("POST", v1+"/household/leave", nil, 200)
	b.ok("GET", path("/recipes/%d", r.ID), nil, 200)
	b.ok("GET", v1+"/household", nil, 200).decode(t, &h)
	if len(h.Members) != 1 {
		t.Fatalf("after leaving: %+v", h)
	}
}

func TestDeletingAUserNeverDeletesRecipesImplicitly(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "leaving")
	var me app.User
	u.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	r := u.createRecipe(simpleRecipe("Keep me"))

	admin.wantCode("DELETE", path("/admin/users/%d", me.ID), nil, 409, "user_has_recipes")
	u.ok("GET", path("/recipes/%d", r.ID), nil, 200) // nothing happened

	var adminUser app.User
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &adminUser)
	admin.ok("DELETE", path("/admin/users/%d?reassign_to=%d", me.ID, adminUser.ID), nil, 200)
	var got app.Recipe
	admin.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.Owner.Handle != "admin" || len(got.Steps) != 2 {
		t.Fatalf("recipe was not transferred intact: %+v", got)
	}
	u.wantCode("GET", v1+"/me", nil, 401, "unauthorized")

	// Deleting the recipes too is possible, but only as an explicit choice.
	v := e.newUser(admin, "second")
	var second app.User
	v.ok("GET", v1+"/me", nil, 200).decode(t, &second)
	r2 := v.createRecipe(simpleRecipe("Delete me"))
	admin.ok("DELETE", path("/admin/users/%d?delete_recipes=true", second.ID), nil, 200)
	admin.ok("GET", path("/recipes/%d", r2.ID), nil, 404)

	var users []app.User
	admin.ok("GET", v1+"/admin/users", nil, 200).decode(t, &users)
	if len(users) != 1 {
		t.Fatalf("users left: %+v", users)
	}
}

func TestCatalogTranslationsFallBackAndWorkForAnyLanguage(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	name := func(q, lang string) string {
		t.Helper()
		var p app.Page[app.Food]
		c.ok("GET", v1+"/foods?q="+q+"&lang="+lang, nil, 200).decode(t, &p)
		if len(p.Items) == 0 {
			t.Fatalf("no food for %q", q)
		}
		return p.Items[0].Name
	}
	if got := name("zwiebel", "de"); got != "Zwiebel" {
		t.Errorf("German: %q", got)
	}
	if got := name("zwiebel", "en"); got != "onion" {
		t.Errorf("English: %q", got)
	}
	if got := name("zwiebel", "es"); got != "onion" {
		t.Errorf("a language without a translation must fall back to English: %q", got)
	}

	// A new language needs no schema or code change, only a row.
	var created app.Food
	c.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "es", Name: "Pimentón ahumado"}, {Locale: "fr", Name: "Paprika fumé"}}}, 201).decode(t, &created)
	if got := name("pimenton", "es"); got != "Pimentón ahumado" {
		t.Errorf("Spanish: %q", got)
	}
	if got := name("pimenton", "en"); got != "Paprika fumé" && got != "Pimentón ahumado" {
		t.Errorf("no English: should show any available name, got %q", got)
	}
	c.wantCode("POST", v1+"/foods", app.FoodInput{}, 400, "name_required")
	c.wantCode("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "not valid", Name: "x"}}}, 400, "bad_locale")
	c.wantCode("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "x"}, {Locale: "en", Name: "y"}}}, 400, "duplicate_locale")

	// A recipe written in one language is found and shown in another.
	german := e.newUser(c, "german")
	german.ok("PATCH", v1+"/me", map[string]string{"locale": "de"}, 200)
	r := german.createRecipe(app.RecipeInput{Name: "Suppe", Steps: []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "Zwiebel"}, {FoodName: "Geheimzutat"}}}}})
	if r.Steps[0].Ingredients[0].Food.Name != "Zwiebel" || r.Steps[0].Ingredients[1].Food.Name != "Geheimzutat" {
		t.Fatalf("new food keeps the author's language: %+v", r.Steps[0].Ingredients)
	}
	var en app.Recipe
	german.ok("GET", path("/recipes/%d?lang=en", r.ID), nil, 200).decode(t, &en)
	if en.Steps[0].Ingredients[0].Food.Name != "onion" || en.Steps[0].Ingredients[1].Food.Name != "Geheimzutat" {
		t.Fatalf("English view: %+v", en.Steps[0].Ingredients)
	}
	var found app.Page[app.RecipeSummary]
	german.ok("GET", v1+"/recipes?q=onion", nil, 200).decode(t, &found)
	if found.Total != 1 {
		t.Fatal("recipe must be found through any language's ingredient name")
	}
}

func TestCatalogPermissionsAndMerge(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "alice")
	r := u.createRecipe(app.RecipeInput{Name: "Soup", Steps: []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "unyun", UnitName: "pcs"}}}}})
	dup := r.Steps[0].Ingredients[0].Food.ID
	dupUnit := r.Steps[0].Ingredients[0].Unit.ID

	var onion app.Page[app.Food]
	u.ok("GET", v1+"/foods?q=zwiebel", nil, 200).decode(t, &onion)
	target := onion.Items[0]
	if target.DefaultUnit == nil {
		t.Fatal("seeded food should suggest a unit")
	}
	if target.CanEdit {
		t.Fatal("seed data is admin-only")
	}
	u.wantCode("PUT", path("/foods/%d", target.ID), app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "hacked"}}}, 403, "forbidden")
	u.wantCode("DELETE", path("/foods/%d", dup), nil, 403, "forbidden")
	u.wantCode("POST", v1+"/foods/merge", app.MergeInput{SourceID: dup, TargetID: target.ID}, 403, "forbidden")

	// Creators can fix their own entries.
	var own app.Food
	u.ok("GET", path("/foods/%d", dup), nil, 200).decode(t, &own)
	if !own.CanEdit {
		t.Fatal("creator should be able to edit")
	}
	u.ok("PUT", path("/foods/%d", dup), app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "unyun"}, {Locale: "de", Name: "Zwibbel"}}, Aliases: []string{"onyon"}}, 200)

	admin.wantCode("DELETE", path("/foods/%d", dup), nil, 409, "in_use")
	admin.wantCode("POST", v1+"/foods/merge", app.MergeInput{SourceID: dup, TargetID: dup}, 400, "merge_same")
	admin.wantCode("POST", v1+"/foods/merge", app.MergeInput{SourceID: dup, TargetID: 999999}, 404, "not_found")
	admin.ok("POST", v1+"/foods/merge", app.MergeInput{SourceID: dup, TargetID: target.ID}, 200)

	var after app.Recipe
	u.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &after)
	if after.Steps[0].Ingredients[0].Food.ID != target.ID {
		t.Fatal("merge did not move the recipe's ingredient")
	}
	var bySource app.Page[app.Food]
	u.ok("GET", v1+"/foods?q=zwibbel", nil, 200).decode(t, &bySource)
	if bySource.Total != 1 || bySource.Items[0].ID != target.ID {
		t.Fatalf("merged name must stay searchable: %+v", bySource)
	}

	// Units behave the same.
	var pc app.Page[app.Unit]
	u.ok("GET", v1+"/units?q=gramm", nil, 200).decode(t, &pc)
	if len(pc.Items) == 0 {
		t.Fatal("seed units missing")
	}
	gram := pc.Items[0].ID
	u.ok("GET", path("/units/%d", gram), nil, 200)
	u.wantCode("PUT", path("/units/%d", gram), app.UnitInput{Translations: []app.Translation{{Locale: "en", Name: "x"}}}, 403, "forbidden")
	var created app.Unit
	u.ok("POST", v1+"/units", app.UnitInput{Translations: []app.Translation{{Locale: "en", Name: "smidgen", Abbreviation: "smg"}}}, 201).decode(t, &created)
	u.ok("PUT", path("/units/%d", created.ID), app.UnitInput{Translations: []app.Translation{{Locale: "en", Name: "smidgens"}}, Aliases: []string{"tad"}}, 200)
	admin.ok("DELETE", path("/units/%d", created.ID), nil, 200)
	admin.wantCode("DELETE", path("/units/%d", dupUnit), nil, 409, "in_use")
	admin.ok("POST", v1+"/units/merge", app.MergeInput{SourceID: dupUnit, TargetID: gram}, 200)
	u.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &after)
	if after.Steps[0].Ingredients[0].Unit.ID != gram {
		t.Fatal("unit merge did not move the ingredient")
	}
	var unused app.Food
	admin.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "unused"}}}, 201).decode(t, &unused)
	admin.ok("DELETE", path("/foods/%d", unused.ID), nil, 200)
	if strings.Contains(string(u.ok("GET", v1+"/foods?q=unused", nil, 200).body), "unused") {
		t.Fatal("food not deleted")
	}
}

func TestSeedDataIsThere(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	var h app.Health
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if h.Foods != 163 || h.Units != 31 {
		t.Fatalf("seed counts: foods %d units %d", h.Foods, h.Units)
	}
}
