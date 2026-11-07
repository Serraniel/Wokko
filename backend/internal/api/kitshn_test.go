package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// kitshnClient behaves like kitshn v2.1.5's login in TandoorAllAuthRoute.kt.
func kitshnLogin(t *testing.T, e *env, handle, password string) string {
	t.Helper()
	res, err := http.Get(e.srv.URL + "/_allauth/browser/v1/auth/session")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	var csrf string
	for _, c := range res.Cookies() {
		if c.Name == "csrftoken" {
			csrf = c.Value
		}
	}
	if csrf == "" {
		t.Fatal("kitshn gives up without a csrftoken cookie, even on the 401 answer")
	}
	req := mustReq(t, "POST", e.srv.URL+"/_allauth/browser/v1/auth/login", fmt.Sprintf(`{"username":%q,"password":%q}`, handle, password))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "csrftoken="+csrf)
	req.Header.Set("X-CSRFToken", csrf)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	for _, c := range res.Cookies() {
		if c.Name == "sessionid" {
			return "sessionid=" + c.Value + "; csrftoken=" + csrf
		}
	}
	return ""
}

func kitshnGet(t *testing.T, e *env, cookie, path string, want int) []byte {
	t.Helper()
	req := mustReq(t, "GET", e.srv.URL+path, "")
	req.Header.Set("Cookie", cookie)
	res := e.anon().send(req)
	if res.status != want {
		t.Fatalf("GET %s: status %d, want %d: %s", path, res.status, want, res.body)
	}
	return res.body
}

// requireFields checks a JSON object has every field that kitshn's data class requires,
// meaning those without a default in kitshn v2.1.5. Missing one would crash its parser.
func requireFields(t *testing.T, name string, raw any, fields ...string) {
	t.Helper()
	obj, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("%s: not an object: %v", name, raw)
	}
	for _, f := range fields {
		if _, ok := obj[f]; !ok {
			t.Errorf("%s: missing required field %q in %v", name, f, obj)
		}
	}
}

func TestKitshnConnectionTestAndLogin(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	// testConnection(ignoreAuth = true) accepts only exactly 403 for anonymous requests.
	kitshnGet(t, e, "", "/api/", 403)
	if got := kitshnLogin(t, e, "admin", "wrong-password"); got != "" {
		t.Fatal("wrong password must not create a session")
	}
	cookie := kitshnLogin(t, e, "admin", "password123")
	if cookie == "" {
		t.Fatal("login returned no sessionid cookie")
	}
	var root map[string]any
	if err := json.Unmarshal(kitshnGet(t, e, cookie, "/api/", 200), &root); err != nil {
		t.Fatal(err)
	}

	// Sessions also work as a Bearer token, which kitshn supports too.
	token := strings.TrimPrefix(strings.SplitN(cookie, ";", 2)[0], "sessionid=")
	req := mustReq(t, "GET", e.srv.URL+"/api/recipe/", "")
	req.Header.Set("Authorization", "Bearer "+token)
	if res := e.anon().send(req); res.status != 200 {
		t.Fatalf("bearer: %d", res.status)
	}
	_ = admin
}

func TestKitshnStartupCallsParse(t *testing.T) {
	e := newEnv(t)
	e.admin()
	cookie := kitshnLogin(t, e, "admin", "password123")

	var settings map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/server-settings/current/", 200), &settings)
	requireFields(t, "server settings", settings, "version")

	var prefs []any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/user-preference/", 200), &prefs)
	if len(prefs) != 1 {
		t.Fatalf("kitshn reads element [0]: %v", prefs)
	}
	requireFields(t, "preference", prefs[0], "user")
	requireFields(t, "preference.user", prefs[0].(map[string]any)["user"], "id")

	var sp map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/space/current/", 200), &sp)
	requireFields(t, "space", sp, "id", "name", "created_at", "message")

	var spaces map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/space/", 200), &spaces)
	requireFields(t, "space page", spaces, "count", "next", "previous", "results")
	kitshnGet(t, e, cookie, fmt.Sprintf("/api/switch-active-space/%d/", int(sp["id"].(float64))), 200)

	for _, area := range []string{"meal-plan", "meal-type", "recipe-book", "recipe-book-entry", "shopping-list", "shopping-list-entry", "supermarket", "supermarket-category", "ai-provider"} {
		var p map[string]any
		json.Unmarshal(kitshnGet(t, e, cookie, "/api/"+area+"/", 200), &p)
		requireFields(t, area, p, "count", "next", "previous", "results")
	}
}

func TestKitshnRecipeShapes(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(app.RecipeInput{Name: "Soup", Description: "Warm", Servings: 4, WorkMinutes: 10, WaitMinutes: 20, Keywords: []string{"Quick"}, Steps: []app.StepInput{
		{Instruction: "Chop", Minutes: 5, Ingredients: []app.IngredientInput{{FoodName: "onion", UnitName: "piece", Amount: ptr(2.0)}, {FoodName: "salt"}}},
	}})
	var withPhoto app.Recipe
	c.upload(r.ID, testPNG(t)).decode(t, &withPhoto)
	cookie := kitshnLogin(t, e, "admin", "password123")

	var page map[string]any
	if err := json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/?page=1&page_size=10", 200), &page); err != nil {
		t.Fatal(err)
	}
	requireFields(t, "recipe page", page, "count", "next", "previous", "results")
	results := page["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results: %v", results)
	}
	requireFields(t, "recipe overview", results[0], "id", "name", "keywords", "working_time", "waiting_time", "created_at", "updated_at", "internal", "servings", "servings_text")
	ov := results[0].(map[string]any)
	if ov["id"].(float64) != float64(r.ID) {
		t.Fatal("recipe IDs must be the real, stable database IDs")
	}
	requireFields(t, "keyword overview", ov["keywords"].([]any)[0], "id", "label")

	// In the old app, IDs from the list could never be fetched.
	var rec map[string]any
	if err := json.Unmarshal(kitshnGet(t, e, cookie, fmt.Sprintf("/api/recipe/%d/", r.ID), 200), &rec); err != nil {
		t.Fatal(err)
	}
	requireFields(t, "recipe", rec, "id", "name", "keywords", "steps", "working_time", "waiting_time", "created_at", "updated_at", "internal", "properties", "food_properties", "servings", "servings_text")
	requireFields(t, "keyword", rec["keywords"].([]any)[0], "id", "name", "created_at", "updated_at")
	step := rec["steps"].([]any)[0]
	requireFields(t, "step", step, "id", "name", "ingredients", "time", "order", "show_as_header")
	ings := step.(map[string]any)["ingredients"].([]any)
	requireFields(t, "ingredient", ings[0], "id", "order", "is_header", "no_amount")
	first := ings[0].(map[string]any)
	if first["amount"].(float64) != 2 || first["no_amount"].(bool) {
		t.Fatalf("amount: %v", first)
	}
	if !ings[1].(map[string]any)["no_amount"].(bool) {
		t.Fatal("ingredient without amount must set no_amount")
	}
	requireFields(t, "food", first["food"], "id", "name")
	requireFields(t, "unit", first["unit"], "id", "name")

	// Images are relative so kitshn adds the instance URL and its session, as its code does.
	img, _ := rec["image"].(string)
	if !strings.HasPrefix(img, "/media/") {
		t.Fatalf("image: %q", img)
	}
	kitshnGet(t, e, cookie, img, 200)

	stepID := int(step.(map[string]any)["id"].(float64))
	kitshnGet(t, e, cookie, fmt.Sprintf("/api/step/%d/", stepID), 200)
	kitshnGet(t, e, cookie, "/api/recipe/99999/", 404)
	kitshnGet(t, e, cookie, "/api/step/99999/", 404)

	var search map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/?query=nothing", 200), &search)
	if search["count"].(float64) != 0 {
		t.Fatalf("search: %v", search)
	}
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/?query=sou&random=true", 200), &search)
	if search["count"].(float64) != 1 {
		t.Fatalf("search: %v", search)
	}

	for _, p := range []string{"/api/food/?query=onion", "/api/unit/?query=piece", "/api/keyword/?query=quick"} {
		var list map[string]any
		json.Unmarshal(kitshnGet(t, e, cookie, p, 200), &list)
		items := list["results"].([]any)
		if len(items) == 0 {
			t.Fatalf("%s: no results", p)
		}
		requireFields(t, p, items[0], "id", "name")
		id := int(items[0].(map[string]any)["id"].(float64))
		base := p[:strings.Index(p, "?")]
		kitshnGet(t, e, cookie, fmt.Sprintf("%s%d/", base, id), 200)
	}
	var kw map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/keyword/?query=quick", 200), &kw)
	requireFields(t, "keyword", kw["results"].([]any)[0], "id", "name", "created_at", "updated_at")
}

func TestKitshnPagingLinks(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	for i := range 3 {
		c.createRecipe(app.RecipeInput{Name: fmt.Sprintf("Recipe %d", i)})
	}
	cookie := kitshnLogin(t, e, "admin", "password123")
	var p map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/?page=1&page_size=2", 200), &p)
	if p["count"].(float64) != 3 || p["next"] == nil || p["previous"] != nil {
		t.Fatalf("first page: %v", p)
	}
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/?page=2&page_size=2", 200), &p)
	if p["next"] != nil || p["previous"] == nil || len(p["results"].([]any)) != 1 {
		t.Fatalf("second page: %v", p)
	}
}

// kitshn must not be a way around the rules of the web app.
func TestKitshnHasTheSamePermissions(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	owner := e.newUser(admin, "owner")
	e.newUser(admin, "john.doe")
	friend := e.newUser(admin, "friend")
	private := owner.createRecipe(simpleRecipe("Secret"))

	owner.ok("POST", v1+"/friends", map[string]string{"handle": "friend"}, 200)
	var incoming []app.Friend
	friend.ok("GET", v1+"/friends", nil, 200).decode(t, &incoming)
	friend.ok("POST", path("/friends/%d/accept", incoming[0].ID), nil, 200)

	cookie := kitshnLogin(t, e, "friend", "password123")
	kitshnGet(t, e, cookie, fmt.Sprintf("/api/recipe/%d/", private.ID), 404)
	var page map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, "/api/recipe/", 200), &page)
	if page["count"].(float64) != 0 {
		t.Fatalf("private recipe leaked: %v", page)
	}

	// kitshn asks for a user list. It gets you and your friends, never everyone.
	body := string(kitshnGet(t, e, cookie, "/api/user/", 200))
	if !strings.Contains(body, "owner") || !strings.Contains(body, "friend") {
		t.Fatalf("user list should have friends: %s", body)
	}
	if strings.Contains(body, "john.doe") || strings.Contains(body, "admin") {
		t.Fatalf("user list reveals strangers: %s", body)
	}
	for _, p := range []string{"/api/recipe/", "/api/food/", "/api/unit/", "/api/keyword/", "/api/user/", "/api/user-preference/", "/api/space/", "/api/space/current/", "/api/cook-log/", "/api/meal-plan/", "/api/step/1/", "/api/food/1/", "/api/unit/1/", "/api/keyword/1/", "/api/recipe/1/"} {
		kitshnGet(t, e, "", p, 403)
	}
}

func TestKitshnDisabledUserLosesAccess(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	u := e.newUser(admin, "alice")
	cookie := kitshnLogin(t, e, "alice", "password123")
	kitshnGet(t, e, cookie, "/api/recipe/", 200)
	var me app.User
	u.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	admin.ok("PATCH", path("/admin/users/%d", me.ID), map[string]bool{"disabled": true}, 200)
	kitshnGet(t, e, cookie, "/api/recipe/", 403)
}

func TestKitshnCookLog(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	cookie := kitshnLogin(t, e, "admin", "password123")
	req := mustReq(t, "POST", e.srv.URL+"/api/cook-log/", fmt.Sprintf(`{"recipe":%d,"servings":2,"rating":5,"comment":"tasty","created_at":"2026-11-07T10:00:00Z"}`, r.ID))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")
	if res := e.anon().send(req); res.status != 201 {
		t.Fatalf("cook log: %d %s", res.status, res.body)
	}
	var p map[string]any
	json.Unmarshal(kitshnGet(t, e, cookie, fmt.Sprintf("/api/cook-log/?recipe=%d", r.ID), 200), &p)
	if p["count"].(float64) != 1 {
		t.Fatalf("log: %v", p)
	}
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.Rating == nil || *got.Rating != 5 {
		t.Fatalf("rating from kitshn must show on the web: %+v", got.Rating)
	}
}

// TestKitshnSetupBeforeLogin follows OnboardingSignIn.kt. It checks the address as it's typed,
// with nobody logged in, and shows "this version doesn't support Tandoor v1" if the server
// settings call fails in any way.
func TestKitshnSetupBeforeLogin(t *testing.T) {
	e := newEnv(t)
	e.admin()
	res, err := http.Get(e.srv.URL + "/api/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("connection test: %d, kitshn needs exactly 403", res.StatusCode)
	}
	res, err = http.Get(e.srv.URL + "/api/server-settings/current/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("server settings before login: %d", res.StatusCode)
	}
	var settings struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil || settings.Version == "" {
		t.Fatalf("version: %q %v", settings.Version, err)
	}
	// Nothing else is public.
	for _, p := range []string{"/api/recipe/", "/api/user/", "/api/meal-plan/", "/api/shopping-list-entry/", "/api/space/current/"} {
		res, err := http.Get(e.srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 403 {
			t.Errorf("%s while logged out: %d", p, res.StatusCode)
		}
	}
}
