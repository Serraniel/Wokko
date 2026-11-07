package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func mustReq(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func mustDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndReadWholeRecipe(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	if len(r.Steps) != 2 || r.Steps[0].Ingredients[0].Food.Name != "onion" || r.Steps[0].Ingredients[0].Unit.Name != "piece" {
		t.Fatalf("recipe content wrong: %+v", r.Steps)
	}
	if *r.Steps[0].Ingredients[0].Amount != 2 || r.Steps[1].Ingredients[0].Amount != nil || r.Steps[1].Ingredients[0].Note != "fine" {
		t.Fatalf("amounts or notes wrong: %+v", r.Steps)
	}
	if len(r.Keywords) != 1 || r.Keywords[0].Name != "Quick" {
		t.Fatalf("seeded keyword should be reused: %+v", r.Keywords)
	}
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.Name != "Soup" || !got.CanEdit || !got.CanManage || got.Owner.Handle != "admin" {
		t.Fatalf("got %+v", got)
	}
	c.wantCode("POST", v1+"/recipes", app.RecipeInput{Name: "  "}, 400, "name_required")
	c.wantCode("POST", v1+"/recipes", app.RecipeInput{Name: "x", Visibility: "everyone"}, 400, "bad_visibility")
	c.wantCode("POST", v1+"/recipes", app.RecipeInput{Name: "x", SourceURL: "javascript:alert(1)"}, 400, "bad_url")
	c.wantCode("POST", v1+"/recipes", app.RecipeInput{Name: "x", Steps: []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: " "}}}}}, 400, "ingredient_required")
	c.wantCode("GET", v1+"/recipes/99999", nil, 404, "not_found")
	c.wantCode("GET", v1+"/recipes/abc", nil, 404, "not_found")
}

// In the old app, editing a recipe that had steps failed. This is the regression test.
func TestEditingARecipeWithStepsWorks(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	in := simpleRecipe("Better soup")
	in.BaseVersion = r.Version
	in.Steps = in.Steps[:1]
	var got app.Recipe
	c.ok("PUT", path("/recipes/%d", r.ID), in, 200).decode(t, &got)
	if got.Name != "Better soup" || len(got.Steps) != 1 || got.Version != r.Version+1 {
		t.Fatalf("got %+v", got)
	}
}

// A failed save must leave no trace: not a partial recipe, and no new catalog entries.
func TestFailedSaveChangesNothing(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Original"))

	var before app.Page[app.Food]
	c.ok("GET", v1+"/foods?q=should-not-exist", nil, 200).decode(t, &before)

	in := simpleRecipe("Changed")
	in.BaseVersion = r.Version
	in.Keywords = []string{"should-not-exist-keyword"}
	in.Steps = []app.StepInput{
		{Instruction: "New", Ingredients: []app.IngredientInput{{FoodName: "should-not-exist"}}},
		{Instruction: "Breaks", Ingredients: []app.IngredientInput{{FoodID: 999999}}},
	}
	c.wantCode("PUT", path("/recipes/%d", r.ID), in, 400, "unknown_food")

	var after app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &after)
	if after.Name != "Original" || len(after.Steps) != 2 || after.Version != r.Version {
		t.Fatalf("failed save changed the recipe: %+v", after)
	}
	var foods app.Page[app.Food]
	c.ok("GET", v1+"/foods?q=should-not-exist", nil, 200).decode(t, &foods)
	if foods.Total != 0 {
		t.Fatalf("failed save left %d foods behind", foods.Total)
	}
	var kw app.Page[app.Keyword]
	c.ok("GET", v1+"/keywords?q=should-not-exist", nil, 200).decode(t, &kw)
	if kw.Total != 0 {
		t.Fatalf("failed save left %d keywords behind", kw.Total)
	}

	bad := simpleRecipe("New")
	bad.Steps[1].Ingredients[0] = app.IngredientInput{FoodID: 999999}
	c.wantCode("POST", v1+"/recipes", bad, 400, "unknown_food")
	var list app.Page[app.RecipeSummary]
	c.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 1 {
		t.Fatalf("failed create left a partial recipe, total %d", list.Total)
	}
}

func TestConflictingEditIsRejected(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	first := simpleRecipe("Tab one")
	first.BaseVersion = r.Version
	c.ok("PUT", path("/recipes/%d", r.ID), first, 200)
	second := simpleRecipe("Tab two")
	second.BaseVersion = r.Version
	c.wantCode("PUT", path("/recipes/%d", r.ID), second, 409, "edit_conflict")
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.Name != "Tab one" {
		t.Fatalf("conflict overwrote data: %q", got.Name)
	}
}

func TestVersionHistoryCanRestore(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("First"))
	in := simpleRecipe("Second")
	in.BaseVersion = r.Version
	in.Steps = nil
	c.ok("PUT", path("/recipes/%d", r.ID), in, 200)
	var versions []app.RecipeVersion
	c.ok("GET", path("/recipes/%d/versions", r.ID), nil, 200).decode(t, &versions)
	if len(versions) != 1 {
		t.Fatalf("want one saved version, got %+v", versions)
	}
	var got app.Recipe
	c.ok("POST", path("/recipes/%d/versions/%d/restore", r.ID, versions[0].ID), nil, 200).decode(t, &got)
	if got.Name != "First" || len(got.Steps) != 2 {
		t.Fatalf("restore failed: %+v", got)
	}
	c.ok("GET", path("/recipes/%d/versions", r.ID), nil, 200).decode(t, &versions)
	if len(versions) != 2 {
		t.Fatalf("restoring must keep the replaced state in history, got %d", len(versions))
	}
}

func TestTrashRestoreAndPurge(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	c.wantCode("DELETE", path("/trash/%d", r.ID), nil, 409, "not_in_trash")
	c.ok("DELETE", path("/recipes/%d", r.ID), nil, 200)
	var list app.Page[app.RecipeSummary]
	c.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 0 {
		t.Fatal("deleted recipe must leave the list")
	}
	var trash []app.RecipeSummary
	c.ok("GET", v1+"/trash", nil, 200).decode(t, &trash)
	if len(trash) != 1 {
		t.Fatalf("trash: %+v", trash)
	}
	c.ok("POST", path("/recipes/%d/restore", r.ID), nil, 200)
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if len(got.Steps) != 2 {
		t.Fatal("restored recipe lost its steps")
	}
	c.ok("DELETE", path("/recipes/%d", r.ID), nil, 200)
	c.ok("DELETE", path("/trash/%d", r.ID), nil, 200)
	c.wantCode("GET", path("/recipes/%d", r.ID), nil, 404, "not_found")
}

func TestSearchFiltersAndSort(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	a := simpleRecipe("Käsespätzle")
	a.Keywords = []string{"Dinner"}
	a.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "onion"}}}}
	c.createRecipe(a)
	b := simpleRecipe("Apple pie")
	b.Keywords = []string{"Dessert"}
	b.Steps = nil
	c.createRecipe(b)

	find := func(q string) int64 {
		t.Helper()
		var p app.Page[app.RecipeSummary]
		c.ok("GET", v1+"/recipes?"+q, nil, 200).decode(t, &p)
		return p.Total
	}
	for q, want := range map[string]int64{"q=kase": 1, "q=K%C3%84SE": 1, "q=pie": 1, "q=nothing": 0, "q=zwiebel": 1, "q=dessert": 1, "": 2} {
		if got := find(q); got != want {
			t.Errorf("query %q: got %d, want %d", q, got, want)
		}
	}
	var kw app.Page[app.Keyword]
	c.ok("GET", v1+"/keywords?q=dessert", nil, 200).decode(t, &kw)
	if len(kw.Items) != 1 {
		t.Fatalf("keywords: %+v", kw)
	}
	var p app.Page[app.RecipeSummary]
	c.ok("GET", path("/recipes?keyword=%d", kw.Items[0].ID), nil, 200).decode(t, &p)
	if p.Total != 1 || p.Items[0].Name != "Apple pie" {
		t.Fatalf("keyword filter: %+v", p)
	}
	c.ok("GET", v1+"/recipes?sort=name&limit=1", nil, 200).decode(t, &p)
	if p.Total != 2 || len(p.Items) != 1 || p.Items[0].Name != "Apple pie" {
		t.Fatalf("sort and paging: %+v", p)
	}
	c.ok("GET", v1+"/recipes?sort=name&limit=1&offset=1", nil, 200).decode(t, &p)
	if p.Items[0].Name != "Käsespätzle" {
		t.Fatalf("second page: %+v", p)
	}
}

func TestVisibilityRules(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	owner := e.newUser(admin, "owner")
	stranger := e.newUser(admin, "stranger")

	var inv app.Invite
	owner.ok("POST", v1+"/household/invites", map[string]any{"max_uses": 1}, 201).decode(t, &inv)
	partner := e.newUser(admin, "partner")
	partner.ok("POST", v1+"/household/join", map[string]string{"token": inv.Token}, 200)

	make := func(name, vis string) app.Recipe {
		in := simpleRecipe(name)
		in.Visibility = vis
		return owner.createRecipe(in)
	}
	private := make("private", "private")
	household := make("household", "household")
	public := make("public", "public")

	for _, c := range []struct {
		who  *client
		r    app.Recipe
		want int
	}{
		{owner, private, 200}, {partner, private, 404}, {stranger, private, 404},
		{partner, household, 200}, {stranger, household, 404},
		{stranger, public, 200}, {partner, public, 200},
	} {
		c.who.ok("GET", path("/recipes/%d", c.r.ID), nil, c.want)
	}

	// Household members can edit shared recipes, but only the owner manages them.
	var seen app.Recipe
	partner.ok("GET", path("/recipes/%d", household.ID), nil, 200).decode(t, &seen)
	if !seen.CanEdit || seen.CanManage {
		t.Fatalf("partner permissions wrong: edit=%v manage=%v", seen.CanEdit, seen.CanManage)
	}
	in := simpleRecipe("edited by partner")
	in.Visibility = "household"
	in.BaseVersion = seen.Version
	partner.ok("PUT", path("/recipes/%d", household.ID), in, 200)
	in.BaseVersion = seen.Version + 1
	in.Visibility = "public"
	partner.wantCode("PUT", path("/recipes/%d", household.ID), in, 403, "forbidden")
	partner.wantCode("DELETE", path("/recipes/%d", household.ID), nil, 403, "forbidden")
	stranger.wantCode("PUT", path("/recipes/%d", private.ID), in, 404, "not_found")
	stranger.wantCode("DELETE", path("/recipes/%d", private.ID), nil, 404, "not_found")
	stranger.wantCode("POST", path("/recipes/%d/photo", public.ID), nil, 400, "")
	stranger.wantCode("GET", path("/recipes/%d/versions", public.ID), nil, 403, "forbidden")

	var list app.Page[app.RecipeSummary]
	partner.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 2 {
		t.Fatalf("partner should reach household and public recipes of the household, got %d", list.Total)
	}
	stranger.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 0 {
		t.Fatalf("other people's public recipes must not appear in your own list, got %d", list.Total)
	}

	// After leaving the household, access ends.
	partner.ok("POST", v1+"/household/leave", nil, 200)
	partner.ok("GET", path("/recipes/%d", household.ID), nil, 404)

	// Public visibility can be turned off for the whole server.
	e.anon().ok("GET", path("/public/recipes/%d", public.ID), nil, 200)
	admin.ok("PUT", v1+"/admin/settings", app.Settings{AllowPublic: false}, 200)
	e.anon().ok("GET", path("/public/recipes/%d", public.ID), nil, 404)
	stranger.ok("GET", path("/recipes/%d", public.ID), nil, 404)
	bad := simpleRecipe("new public")
	bad.Visibility = "public"
	owner.wantCode("POST", v1+"/recipes", bad, 403, "public_disabled")
}

func TestPublicPagesShowFullRecipeButNotTheAuthor(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	in := simpleRecipe("Shared soup")
	in.Visibility = "public"
	r := admin.createRecipe(in)

	anon := e.anon()
	var got app.Recipe
	anon.ok("GET", path("/public/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if len(got.Steps) != 2 || got.Steps[0].Ingredients[0].Food.Name != "onion" {
		t.Fatalf("public recipe must include steps and ingredients: %+v", got)
	}
	var page app.Page[app.RecipeSummary]
	anon.ok("GET", v1+"/public/recipes", nil, 200).decode(t, &page)
	if page.Total != 1 {
		t.Fatalf("gallery: %+v", page)
	}
	raw := string(anon.ok("GET", path("/public/recipes/%d", r.ID), nil, 200).body)
	raw += string(anon.ok("GET", v1+"/public/recipes", nil, 200).body)
	if strings.Contains(raw, "admin") {
		t.Fatalf("public responses reveal the author's handle: %s", raw)
	}
	if got.Owner != nil || got.CanEdit || got.CanManage {
		t.Fatalf("anonymous viewer got account data: %+v", got)
	}
}

func TestShareLinks(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	r := admin.createRecipe(simpleRecipe("Private soup"))
	if res := admin.upload(r.ID, testPNG(t)); res.status != 200 {
		t.Fatalf("upload: %s", res.body)
	}
	var link app.ShareLink
	admin.ok("POST", path("/recipes/%d/sharing/links", r.ID), map[string]any{}, 201).decode(t, &link)
	if !strings.HasPrefix(link.URL, "http://wokko.test/s/") {
		t.Fatalf("link must use the public URL and a page that exists: %s", link.URL)
	}
	token := link.URL[strings.LastIndex(link.URL, "/")+1:]

	anon := e.anon()
	var got app.Recipe
	anon.ok("GET", path("/public/share/%s", token), nil, 200).decode(t, &got)
	if got.Name != "Private soup" || len(got.Steps) != 2 || got.Owner != nil {
		t.Fatalf("shared recipe: %+v", got)
	}
	if res := anon.do("GET", got.ThumbURL, nil); res.status != 200 {
		t.Fatalf("share link must also unlock the photo: %d", res.status)
	}

	var sharing app.Sharing
	admin.ok("GET", path("/recipes/%d/sharing", r.ID), nil, 200).decode(t, &sharing)
	if len(sharing.Links) != 1 {
		t.Fatalf("links: %+v", sharing)
	}
	admin.ok("DELETE", path("/recipes/%d/sharing/links/%d", r.ID, link.ID), nil, 200)
	anon.ok("GET", path("/public/share/%s", token), nil, 404)
	if res := anon.do("GET", got.PhotoURL, nil); res.status != 404 {
		t.Fatalf("revoked link must not unlock the photo: %d", res.status)
	}

	var expiring app.ShareLink
	admin.ok("POST", path("/recipes/%d/sharing/links", r.ID), map[string]any{"expires_in_days": 7}, 201).decode(t, &expiring)
	if expiring.ExpiresAt == "" {
		t.Fatal("expiry was not stored")
	}
	anon.ok("GET", v1+"/public/share/does-not-exist", nil, 404)
}

func TestPhotosArePrivateAndReplacedSafely(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	stranger := e.newUser(admin, "stranger")
	r := admin.createRecipe(simpleRecipe("Soup"))

	var withPhoto app.Recipe
	res := admin.upload(r.ID, testPNG(t))
	if res.status != 200 {
		t.Fatalf("upload: %s", res.body)
	}
	res.decode(t, &withPhoto)
	if withPhoto.PhotoURL == "" || withPhoto.ThumbURL == "" {
		t.Fatalf("no photo URLs: %+v", withPhoto)
	}
	img := admin.do("GET", withPhoto.ThumbURL, nil)
	if img.status != 200 || img.header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("owner can't load the photo: %d", img.status)
	}
	if !strings.Contains(img.header.Get("Cache-Control"), "private") {
		t.Fatal("private photos must not be stored by shared caches")
	}
	stranger.ok("GET", withPhoto.PhotoURL, nil, 404)
	e.anon().ok("GET", withPhoto.PhotoURL, nil, 404)
	e.anon().ok("GET", "/media/..%2fwokko.db", nil, 404)

	// A rejected upload must keep the old photo. The old app deleted it first.
	bad := admin.upload(r.ID, []byte("this is not an image"))
	if bad.status != 400 || bad.code(t) != "invalid_image" {
		t.Fatalf("got %d: %s", bad.status, bad.body)
	}
	var same app.Recipe
	admin.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &same)
	if same.PhotoURL != withPhoto.PhotoURL {
		t.Fatal("rejected upload changed the photo")
	}
	admin.ok("GET", withPhoto.PhotoURL, nil, 200)

	// A successful replace removes the old files.
	var replaced app.Recipe
	admin.upload(r.ID, testPNG(t)).decode(t, &replaced)
	if replaced.PhotoURL == withPhoto.PhotoURL {
		t.Fatal("photo was not replaced")
	}
	old := filepath.Join(e.cfg.UploadsDir(), filepath.Base(withPhoto.PhotoURL))
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old photo file remains: %v", err)
	}
	admin.ok("DELETE", path("/recipes/%d/photo", r.ID), nil, 200)
	admin.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &same)
	if same.PhotoURL != "" {
		t.Fatal("photo was not removed")
	}
}

func TestCookLog(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	c.wantCode("POST", path("/recipes/%d/cooklog", r.ID), map[string]any{"rating": 9}, 400, "bad_rating")
	c.ok("POST", path("/recipes/%d/cooklog", r.ID), map[string]any{"rating": 4, "servings": 2, "comment": "good"}, 201)
	var log []app.CookLogEntry
	c.ok("GET", path("/recipes/%d/cooklog", r.ID), nil, 200).decode(t, &log)
	if len(log) != 1 || *log[0].Rating != 4 {
		t.Fatalf("log: %+v", log)
	}
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", r.ID), nil, 200).decode(t, &got)
	if got.Rating == nil || *got.Rating != 4 || got.TimesCooked != 1 || got.LastCooked == "" {
		t.Fatalf("stats: %+v", got.RecipeSummary)
	}
}

func TestExportIsUsableWithoutWokko(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	if res := c.upload(r.ID, testPNG(t)); res.status != 200 {
		t.Fatal(string(res.body))
	}
	trashed := c.createRecipe(simpleRecipe("Binned"))
	c.ok("DELETE", path("/recipes/%d", trashed.ID), nil, 200)

	res := c.ok("GET", v1+"/me/export", nil, 200)
	zr, err := zip.NewReader(bytes.NewReader(res.body), int64(len(res.body)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var data app.ExportData
	for _, f := range zr.File {
		names = append(names, f.Name)
		if f.Name == "recipes.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(data.Recipes) != 1 || data.Recipes[0].Steps[0].Ingredients[0].Food.Name != "onion" {
		t.Fatalf("export: %+v", data)
	}
	if len(names) != 2 || !strings.HasPrefix(names[1], "photos/") {
		t.Fatalf("export files: %v", names)
	}
}
