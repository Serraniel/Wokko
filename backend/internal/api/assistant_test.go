package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/importer"
)

func enableMCP(t *testing.T, admin *client) {
	t.Helper()
	admin.ok("PUT", v1+"/admin/settings", app.Settings{AllowPublic: true, MCPEnabled: true}, 200)
}

func newToken(t *testing.T, c *client, scope string) string {
	t.Helper()
	var tok app.NewAPIToken
	c.ok("POST", v1+"/me/tokens", app.APITokenInput{Name: "Claude " + scope, Scope: scope}, 201).decode(t, &tok)
	if !strings.HasPrefix(tok.Secret, "wokko_") {
		t.Fatalf("secret: %q", tok.Secret)
	}
	return tok.Secret
}

type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// connect uses the official MCP client, so this tests the real protocol.
func connect(t *testing.T, e *env, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: e.srv.URL + "/mcp", HTTPClient: &http.Client{Transport: bearer{token, http.DefaultTransport}}}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func toolNames(t *testing.T, s *mcp.ClientSession) []string {
	t.Helper()
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	return names
}

// call returns the tool's result, failing the test if the tool reported an error.
func call(t *testing.T, s *mcp.ClientSession, name string, args any, out any) {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s failed: %+v", name, res.Content[0])
	}
	if out != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		var wrapper struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(raw, &wrapper); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(wrapper.Result, out); err != nil {
			t.Fatalf("%s: %v in %s", name, err, raw)
		}
	}
}

func callFails(t *testing.T, s *mcp.ClientSession, name string, args any) string {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	if !res.IsError {
		t.Fatalf("%s should have failed: %+v", name, res.StructuredContent)
	}
	return fmt.Sprint(res.Content[0].(*mcp.TextContent).Text)
}

func TestMCPIsOffUntilAnAdminEnablesIt(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	token := newToken(t, admin, "read")
	req := mustReq(t, "POST", e.srv.URL+"/mcp", `{}`)
	req.Header.Set("Authorization", "Bearer "+token)
	if res := e.anon().send(req); res.status != 404 {
		t.Fatalf("while off: %d", res.status)
	}
	for _, p := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-authorization-server"} {
		if res := e.anon().do("GET", p, nil); res.status != 404 {
			t.Fatalf("%s while off: %d", p, res.status)
		}
	}
	var status app.Status
	e.anon().ok("GET", v1+"/status", nil, 200).decode(t, &status)
	if status.MCPEnabled {
		t.Fatal("off by default")
	}
	enableMCP(t, admin)
	e.anon().ok("GET", v1+"/status", nil, 200).decode(t, &status)
	if !status.MCPEnabled {
		t.Fatal("not on")
	}
	bad := mustReq(t, "POST", e.srv.URL+"/mcp", `{}`)
	bad.Header.Set("Authorization", "Bearer wokko_nonsense")
	res := e.anon().send(bad)
	if res.status != 401 || !strings.Contains(res.header.Get("WWW-Authenticate"), "/.well-known/oauth-protected-resource") {
		t.Fatalf("bad token: %d %v", res.status, res.header)
	}
	// A session cookie isn't an MCP credential, and a token isn't a web session.
	webWithToken := mustReq(t, "GET", e.srv.URL+v1+"/me", "")
	webWithToken.Header.Set("Authorization", "Bearer "+token)
	if res := e.anon().send(webWithToken); res.status != 401 {
		t.Fatalf("token on the web API: %d", res.status)
	}
	if res := admin.do("POST", "/mcp", map[string]any{}); res.status != 401 {
		t.Fatalf("cookie on MCP: %d", res.status)
	}
}

func TestMCPScopesDecideWhichToolsExist(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	read := toolNames(t, connect(t, e, newToken(t, admin, "read")))
	write := toolNames(t, connect(t, e, newToken(t, admin, "write")))
	del := toolNames(t, connect(t, e, newToken(t, admin, "delete")))
	if !slices.Contains(read, "get_recipe") || slices.Contains(read, "create_recipe") || slices.Contains(read, "delete_recipe") {
		t.Fatalf("read: %v", read)
	}
	if !slices.Contains(write, "create_recipe") || slices.Contains(write, "delete_recipe") {
		t.Fatalf("write: %v", write)
	}
	if !slices.Contains(del, "delete_recipe") {
		t.Fatalf("delete: %v", del)
	}
	// These can't be reached with any token.
	for _, name := range del {
		for _, banned := range []string{"user", "people", "friend", "purge", "admin", "visibility", "public", "share"} {
			if strings.Contains(name, banned) {
				t.Errorf("tool %q should not exist", name)
			}
		}
	}
}

func TestMCPToolsWorkAndStayWithinTheOwnersRights(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	other := e.newUser(admin, "other")
	secret := simpleRecipe("Others secret")
	secret.Visibility = "private"
	hidden := other.createRecipe(secret)

	s := connect(t, e, newToken(t, admin, "delete"))
	amount := 200.0
	var made app.Recipe
	call(t, s, "create_recipe", map[string]any{"name": "Dal", "servings": 2, "keywords": []string{"Soup"}, "steps": []map[string]any{{"instruction": "Cook.", "ingredients": []map[string]any{{"food": "Linsen", "unit": "g", "amount": amount}}}}}, &made)
	if made.Visibility != "private" || made.Steps[0].Ingredients[0].Unit == nil {
		t.Fatalf("created: %+v", made)
	}
	var page app.Page[app.RecipeSummary]
	call(t, s, "search_recipes", map[string]any{"query": "dal"}, &page)
	if page.Total != 1 {
		t.Fatalf("search: %+v", page)
	}
	call(t, s, "search_recipes", map[string]any{"query": "secret"}, &page)
	if page.Total != 0 {
		t.Fatalf("another person's private recipe was found: %+v", page)
	}
	if msg := callFails(t, s, "get_recipe", map[string]any{"id": hidden.ID}); !strings.Contains(msg, "not_found") {
		t.Fatalf("private recipe: %s", msg)
	}
	callFails(t, s, "update_recipe", map[string]any{"id": hidden.ID, "name": "Stolen", "base_version": hidden.Version})
	callFails(t, s, "delete_recipe", map[string]any{"id": hidden.ID})
	callFails(t, s, "add_recipe_to_shopping", map[string]any{"recipe_id": hidden.ID})

	var read app.Recipe
	call(t, s, "get_recipe", map[string]any{"id": made.ID}, &read)
	var updated app.Recipe
	call(t, s, "update_recipe", map[string]any{"id": made.ID, "name": "Better dal", "base_version": read.Version, "servings": 4}, &updated)
	if updated.Name != "Better dal" || updated.Visibility != "private" {
		t.Fatalf("updated: %+v", updated)
	}
	if msg := callFails(t, s, "update_recipe", map[string]any{"id": made.ID, "name": "Stale", "base_version": read.Version}); !strings.Contains(msg, "edit_conflict") {
		t.Fatalf("stale edit: %s", msg)
	}

	var types []app.MealType
	call(t, s, "list_meal_types", map[string]any{}, &types)
	var plan app.MealPlan
	call(t, s, "create_meal_plan", map[string]any{"date": "2026-11-09", "meal_type_id": types[2].ID, "recipe_id": made.ID, "servings": 3}, &plan)
	var plans []app.MealPlan
	call(t, s, "list_meal_plans", map[string]any{"from": "2026-11-01", "to": "2026-11-30"}, &plans)
	if len(plans) != 1 || plans[0].ID != plan.ID {
		t.Fatalf("plans: %+v", plans)
	}
	call(t, s, "delete_meal_plan", map[string]any{"id": plan.ID}, nil)

	call(t, s, "add_to_shopping", map[string]any{"items": []string{"2 kg potatoes"}}, nil)
	call(t, s, "add_recipe_to_shopping", map[string]any{"recipe_id": updated.ID}, nil)
	var list []app.ShoppingItem
	call(t, s, "list_shopping", map[string]any{}, &list)
	if len(list) != 1 || *list[0].Amount != 2 {
		t.Fatalf("shopping: %+v", list)
	}
	call(t, s, "check_shopping", map[string]any{"ids": []int64{list[0].ID}, "checked": true}, nil)
	var book app.RecipeBook
	call(t, s, "create_book", map[string]any{"name": "Soups"}, &book)
	call(t, s, "add_recipe_to_book", map[string]any{"book_id": book.ID, "recipe_id": made.ID}, nil)
	var books []app.RecipeBook
	call(t, s, "list_books", map[string]any{}, &books)
	if len(books) != 1 || books[0].RecipeCount != 1 {
		t.Fatalf("books: %+v", books)
	}
	var nutrition app.RecipeNutrition
	call(t, s, "get_recipe_nutrition", map[string]any{"id": made.ID}, &nutrition)
	var foods app.Page[app.Food]
	call(t, s, "search_foods", map[string]any{"query": "onion"}, &foods)
	if foods.Total == 0 {
		t.Fatal("foods")
	}
	call(t, s, "search_units", map[string]any{"query": "gram"}, nil)
	call(t, s, "search_tags", map[string]any{}, nil)

	// Deleting goes to the trash, and the person can see what happened.
	call(t, s, "delete_recipe", map[string]any{"id": made.ID}, nil)
	var trash []app.RecipeSummary
	admin.ok("GET", v1+"/trash", nil, 200).decode(t, &trash)
	if len(trash) != 1 {
		t.Fatalf("trash: %+v", trash)
	}
	var activity []app.AssistantActivity
	admin.ok("GET", v1+"/me/assistant-activity", nil, 200).decode(t, &activity)
	if len(activity) < 5 || activity[0].Tool != "delete_recipe" || activity[0].TokenName != "Claude delete" {
		t.Fatalf("activity: %+v", activity)
	}
	var theirs []app.AssistantActivity
	other.ok("GET", v1+"/me/assistant-activity", nil, 200).decode(t, &theirs)
	if len(theirs) != 0 {
		t.Fatalf("someone else's activity: %+v", theirs)
	}
}

func TestMCPTokensCanBeRevokedAndExpire(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	other := e.newUser(admin, "other")
	var tok app.NewAPIToken
	other.ok("POST", v1+"/me/tokens", app.APITokenInput{Name: "x", Scope: "read"}, 201).decode(t, &tok)
	admin.wantCode("POST", v1+"/me/tokens", app.APITokenInput{Name: "x", Scope: "root"}, 400, "bad_scope")
	admin.wantCode("POST", v1+"/me/tokens", app.APITokenInput{Name: " ", Scope: "read"}, 400, "name_required")
	admin.wantCode("DELETE", path("/me/tokens/%d", tok.ID), nil, 404, "not_found")
	connect(t, e, tok.Secret)

	var listed []app.APIToken
	other.ok("GET", v1+"/me/tokens", nil, 200).decode(t, &listed)
	if len(listed) != 1 || listed[0].LastUsedAt == "" {
		t.Fatalf("tokens: %+v", listed)
	}
	if raw, _ := json.Marshal(listed); strings.Contains(string(raw), tok.Secret) {
		t.Fatal("the secret must only be shown once")
	}

	// A disabled person's token stops at once.
	var users []app.User
	admin.ok("GET", v1+"/admin/users", nil, 200).decode(t, &users)
	var otherID int64
	for _, u := range users {
		if u.Handle == "other" {
			otherID = u.ID
		}
	}
	mcpStatus := func() int {
		req := mustReq(t, "POST", e.srv.URL+"/mcp", `{}`)
		req.Header.Set("Authorization", "Bearer "+tok.Secret)
		return e.anon().send(req).status
	}
	admin.ok("PATCH", path("/admin/users/%d", otherID), map[string]any{"disabled": true}, 200)
	if got := mcpStatus(); got != 401 {
		t.Fatalf("disabled owner: %d", got)
	}
	admin.ok("PATCH", path("/admin/users/%d", otherID), map[string]any{"disabled": false}, 200)
	other2 := e.anon()
	other2.ok("POST", v1+"/auth/login", map[string]string{"handle": "other", "password": "password123"}, 200)
	other2.ok("DELETE", path("/me/tokens/%d", tok.ID), nil, 200)
	if got := mcpStatus(); got != 401 {
		t.Fatalf("revoked: %d", got)
	}
}

func oauthForm(t *testing.T, e *env, values url.Values) (int, map[string]any) {
	t.Helper()
	res, err := http.PostForm(e.srv.URL+"/oauth/token", values)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("token answer: %s", raw)
	}
	return res.StatusCode, out
}

func TestOAuthFlowForAssistants(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	anon := e.anon()

	var meta map[string]any
	anon.ok("GET", "/.well-known/oauth-protected-resource", nil, 200).decode(t, &meta)
	if !strings.HasSuffix(meta["resource"].(string), "/mcp") {
		t.Fatalf("resource: %v", meta)
	}
	anon.ok("GET", "/.well-known/oauth-authorization-server", nil, 200).decode(t, &meta)
	for _, f := range []string{"authorization_endpoint", "token_endpoint", "registration_endpoint"} {
		if meta[f] == nil {
			t.Fatalf("missing %s: %v", f, meta)
		}
	}
	if !slices.Contains(meta["code_challenge_methods_supported"].([]any), any("S256")) {
		t.Fatalf("PKCE: %v", meta)
	}

	// Only https, or plain http back to the same computer.
	for _, bad := range []string{"http://evil.example/cb", "javascript:alert(1)", "ftp://x/y", "https://user:pw@x.example/"} {
		res := anon.do("POST", "/oauth/register", map[string]any{"client_name": "Evil", "redirect_uris": []string{bad}})
		if res.status != 400 {
			t.Errorf("%s: %d", bad, res.status)
		}
	}
	redirect := "http://127.0.0.1:33333/callback"
	var reg map[string]any
	anon.ok("POST", "/oauth/register", map[string]any{"client_name": "Claude", "redirect_uris": []string{redirect, "https://claude.ai/api/mcp/auth_callback"}}, 201).decode(t, &reg)
	id := reg["client_id"].(string)

	verifier := strings.Repeat("v", 50)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	var info app.OAuthRequest
	admin.ok("GET", v1+"/oauth/request?"+url.Values{"client_id": {id}, "redirect_uri": {redirect}, "scope": {"write"}}.Encode(), nil, 200).decode(t, &info)
	if info.ClientName != "Claude" || info.RedirectHost != "127.0.0.1" || info.Scope != "write" {
		t.Fatalf("request: %+v", info)
	}
	anon.wantCode("GET", v1+"/oauth/request?client_id="+id, nil, 401, "unauthorized")
	admin.wantCode("GET", v1+"/oauth/request?"+url.Values{"client_id": {id}, "redirect_uri": {"https://evil.example/"}}.Encode(), nil, 400, "invalid_redirect_uri")

	approve := func() string {
		t.Helper()
		var out app.OAuthRedirect
		admin.ok("POST", v1+"/oauth/approve", app.OAuthApproval{ClientID: id, RedirectURI: redirect, CodeChallenge: challenge, Scope: "write", State: "xyz"}, 200).decode(t, &out)
		u, err := url.Parse(out.RedirectURL)
		if err != nil || u.Query().Get("state") != "xyz" {
			t.Fatalf("redirect: %q", out.RedirectURL)
		}
		return u.Query().Get("code")
	}
	admin.wantCode("POST", v1+"/oauth/approve", app.OAuthApproval{ClientID: id, RedirectURI: redirect, Scope: "write"}, 400, "invalid_request")
	anon.wantCode("POST", v1+"/oauth/approve", app.OAuthApproval{ClientID: id, RedirectURI: redirect, CodeChallenge: challenge}, 401, "unauthorized")

	exchange := func(code, verify string) (int, map[string]any) {
		return oauthForm(t, e, url.Values{"grant_type": {"authorization_code"}, "client_id": {id}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verify}})
	}
	// A code without the right verifier is useless, and is spent by the attempt.
	code := approve()
	if status, _ := exchange(code, "wrong"); status != 400 {
		t.Fatalf("wrong verifier: %d", status)
	}
	if status, _ := exchange(code, verifier); status != 400 {
		t.Fatalf("a spent code worked: %d", status)
	}
	code = approve()
	status, tokens := exchange(code, verifier)
	if status != 200 || tokens["access_token"] == nil || tokens["token_type"] != "Bearer" {
		t.Fatalf("exchange: %d %v", status, tokens)
	}
	if status, _ := exchange(code, verifier); status != 400 {
		t.Fatalf("code reuse: %d", status)
	}
	s := connect(t, e, tokens["access_token"].(string))
	names := toolNames(t, s)

	// The app asked for write access, and the person chose to give it read access only.
	var low app.OAuthRedirect
	admin.ok("POST", v1+"/oauth/approve", app.OAuthApproval{ClientID: id, RedirectURI: redirect, CodeChallenge: challenge, Scope: "read", State: "xyz"}, 200).decode(t, &low)
	lowURL, err := url.Parse(low.RedirectURL)
	if err != nil {
		t.Fatal(err)
	}
	status, lowTokens := exchange(lowURL.Query().Get("code"), verifier)
	if status != 200 {
		t.Fatalf("read-only exchange: %d", status)
	}
	if readOnly := toolNames(t, connect(t, e, lowTokens["access_token"].(string))); len(readOnly) == 0 || len(readOnly) >= len(names) {
		t.Fatalf("read-only app got %v, write app %v", readOnly, names)
	}
	if !slices.Contains(names, "create_recipe") || slices.Contains(names, "delete_recipe") {
		t.Fatalf("scope: %v", names)
	}

	// Refresh replaces the old token, and the app shows once in settings.
	refresh := tokens["refresh_token"].(string)
	status, again := oauthForm(t, e, url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {refresh}})
	if status != 200 || again["access_token"] == tokens["access_token"] {
		t.Fatalf("refresh: %d %v", status, again)
	}
	if status, _ := oauthForm(t, e, url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {refresh}}); status != 400 {
		t.Fatalf("old refresh token: %d", status)
	}
	var listed []app.APIToken
	admin.ok("GET", v1+"/me/tokens", nil, 200).decode(t, &listed)
	if len(listed) != 1 || !listed[0].IsApp || listed[0].Name != "Claude" {
		t.Fatalf("listed: %+v", listed)
	}
	// Revoking ends the app's access for good.
	admin.ok("DELETE", path("/me/tokens/%d", listed[0].ID), nil, 200)
	if status, _ := oauthForm(t, e, url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {again["refresh_token"].(string)}}); status != 400 {
		t.Fatalf("refresh after revoke: %d", status)
	}
	req := mustReq(t, "POST", e.srv.URL+"/mcp", `{}`)
	req.Header.Set("Authorization", "Bearer "+again["access_token"].(string))
	if res := e.anon().send(req); res.status != 401 {
		t.Fatalf("access after revoke: %d", res.status)
	}
	// Authorize sends the browser to the consent page.
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := noRedirect.Get(e.srv.URL + "/oauth/authorize?client_id=" + id)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 302 || !strings.HasPrefix(res.Header.Get("Location"), "/oauth/consent?client_id=") {
		t.Fatalf("authorize: %d %s", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestMCPAsksBeforeAddingFoodsAndUnits(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	var herb app.Food
	admin.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "de", Name: "Wokkokraut"}}}, 201).decode(t, &herb)
	s := connect(t, e, newToken(t, admin, "write"))
	recipe := func(food string, extra map[string]any) map[string]any {
		args := map[string]any{"name": "Test " + food, "steps": []map[string]any{{"instruction": "Mix.", "ingredients": []map[string]any{{"food": food, "unit": "g", "amount": 5}}}}}
		for k, v := range extra {
			args[k] = v
		}
		return args
	}

	// By default the person confirms new entries, and the refusal points at close matches.
	msg := callFails(t, s, "create_recipe", recipe("Wokkokrauts", nil))
	if !strings.Contains(msg, "new_entries_need_confirmation") || !strings.Contains(msg, fmt.Sprintf("food_id %d", herb.ID)) {
		t.Fatalf("ask: %s", msg)
	}
	var made app.Recipe
	call(t, s, "create_recipe", recipe("Wokkobeere", map[string]any{"confirmed_new": []string{"wokkobeere"}}), &made)
	if made.Steps[0].Ingredients[0].Food.Name != "Wokkobeere" {
		t.Fatalf("confirmed: %+v", made.Steps[0].Ingredients[0])
	}

	// "never" refuses even a confirmed name, but known entries still work.
	admin.ok("PATCH", v1+"/me", map[string]any{"assistant_new_entries": "never"}, 200)
	if msg := callFails(t, s, "create_recipe", recipe("Wokkonuss", map[string]any{"confirmed_new": []string{"Wokkonuss"}})); !strings.Contains(msg, "new_entries_not_allowed") {
		t.Fatalf("never: %s", msg)
	}
	call(t, s, "create_recipe", map[string]any{"name": "Known", "steps": []map[string]any{{"instruction": "Mix.", "ingredients": []map[string]any{{"food_id": herb.ID}, {"food": "wokkobeere"}}}}}, nil)

	admin.ok("PATCH", v1+"/me", map[string]any{"assistant_new_entries": "allow"}, 200)
	call(t, s, "create_recipe", recipe("Wokkonuss", nil), nil)
	var me app.User
	admin.ok("GET", v1+"/me", nil, 200).decode(t, &me)
	if me.AssistantNewEntries != "allow" {
		t.Fatalf("me: %+v", me)
	}
	admin.wantCode("PATCH", v1+"/me", map[string]any{"assistant_new_entries": "sometimes"}, 400, "bad_new_entries")
}

func TestMCPUsesTheWholeCatalog(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	e.svc.Fetcher = importer.NewFetcher("test", true)
	photo := testPNG(t)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(photo)
	}))
	defer site.Close()
	s := connect(t, e, newToken(t, admin, "write"))

	// One call looks up a whole ingredient list.
	var found []struct {
		Query string     `json:"query"`
		Total int64      `json:"total"`
		Items []app.Food `json:"items"`
	}
	call(t, s, "search_foods", map[string]any{"queries": []string{"onion", "nothing like this"}}, &found)
	if len(found) != 2 || found[0].Total == 0 || found[1].Total != 0 {
		t.Fatalf("search: %+v", found)
	}
	onion := found[0].Items[0]
	var units []struct {
		Items []app.Unit `json:"items"`
	}
	call(t, s, "search_units", map[string]any{"queries": []string{"g"}}, &units)
	gram := units[0].Items[0]

	// Ids, the serving label and a photo are all available.
	var made app.Recipe
	call(t, s, "create_recipe", map[string]any{"name": "Bread", "servings": 1, "servings_text": "loaf", "steps": []map[string]any{{"instruction": "Bake.", "ingredients": []map[string]any{{"food_id": onion.ID, "unit_id": gram.ID, "amount": 50}}}}}, &made)
	got := made.Steps[0].Ingredients[0]
	if made.ServingsText != "loaf" || got.Food.ID != onion.ID || got.Unit == nil || got.Unit.ID != gram.ID {
		t.Fatalf("made: %+v %+v", made, got)
	}
	var withPhoto app.Recipe
	call(t, s, "set_recipe_photo", map[string]any{"id": made.ID, "image_url": site.URL + "/p.png"}, &withPhoto)
	if withPhoto.PhotoURL == "" {
		t.Fatalf("photo: %+v", withPhoto)
	}

	// A known food learns another name, without losing the ones it has.
	var named app.Food
	call(t, s, "add_food_names", map[string]any{"id": onion.ID, "aliases": []string{"Wokkozwiebel"}, "translations": []map[string]any{{"locale": "fr", "name": "oignon"}}}, &named)
	if !slices.Contains(named.Aliases, "Wokkozwiebel") || len(named.Translations) <= len(onion.Translations) {
		t.Fatalf("named: %+v", named)
	}
	call(t, s, "search_foods", map[string]any{"queries": []string{"wokkozwiebel"}}, &found)
	if found[0].Total != 1 || found[0].Items[0].ID != onion.ID {
		t.Fatalf("alias search: %+v", found)
	}
	var unit app.Unit
	call(t, s, "add_unit_names", map[string]any{"id": gram.ID, "aliases": []string{"Wokkogramm"}}, &unit)
	if !slices.Contains(unit.Aliases, "Wokkogramm") {
		t.Fatalf("unit: %+v", unit)
	}

	// Someone who may not edit an entry can't name it either.
	other := e.newUser(admin, "other")
	theirs := connect(t, e, newToken(t, other, "write"))
	if msg := callFails(t, theirs, "add_food_names", map[string]any{"id": onion.ID, "aliases": []string{"x"}}); !strings.Contains(msg, "forbidden") {
		t.Fatalf("other: %s", msg)
	}
	// Photos come only from public addresses.
	e.svc.Fetcher = importer.NewFetcher("test", false)
	callFails(t, s, "set_recipe_photo", map[string]any{"id": made.ID, "image_url": site.URL + "/p.png"})
}
