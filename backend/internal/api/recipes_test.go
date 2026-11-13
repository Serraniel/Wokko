package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/importer"
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

func TestImportFromAPageGivesADraftAndSavesNothing(t *testing.T) {
	e := newEnv(t)
	photo := testPNG(t)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/photo.png" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><script type="application/ld+json">{"@type":"Recipe","name":"Tomatensoße","image":"/photo.png","recipeYield":"2 Portionen","prepTime":"PT10M","recipeIngredient":["800 g Tomaten","2 Zwiebeln","1 EL Olivenöl, kalt"],"recipeInstructions":[{"@type":"HowToStep","text":"Köcheln."}]}</script></html>`))
	}))
	defer site.Close()
	e.svc.Fetcher = importer.NewFetcher("test", true)
	me := e.admin()
	var out app.ImportedRecipe
	me.ok("POST", v1+"/import", app.ImportRequest{URL: site.URL + "/r"}, 200).decode(t, &out)
	if out.Recipe.Name != "Tomatensoße" || out.Recipe.Servings != 2 || out.Recipe.WorkMinutes != 10 || out.Recipe.Visibility != "private" || out.Recipe.SourceURL != site.URL+"/r" {
		t.Fatalf("draft: %+v", out.Recipe)
	}
	if out.ImageData == "" || out.ImageType != "image/png" {
		t.Fatalf("photo: %q", out.ImageType)
	}
	ing := out.Recipe.Steps[0].Ingredients
	if len(ing) != 3 || ing[0].UnitName != "gram" && ing[0].UnitName != "Gramm" || *ing[0].Amount != 800 || ing[2].Note != "kalt" {
		t.Fatalf("ingredients: %+v", ing)
	}
	var list app.Page[app.RecipeSummary]
	me.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 0 {
		t.Fatalf("an import must save nothing, found %d recipes", list.Total)
	}
	// The draft saves cleanly, and plural spellings find the catalog's foods.
	saved := me.createRecipe(out.Recipe)
	if got := saved.Steps[0].Ingredients[1].Food.Name; got != "onion" && got != "Zwiebel" {
		t.Fatalf("plural created a new food instead of matching: %q", got)
	}
}

func TestImportFromPastedText(t *testing.T) {
	e := newEnv(t)
	var out app.ImportedRecipe
	e.admin().ok("POST", v1+"/import", app.ImportRequest{Text: "Suppe\nZutaten\n1 l Wasser\n2 Karotten\nZubereitung\nKochen."}, 200).decode(t, &out)
	if out.Recipe.Name != "Suppe" || !out.Guessed || len(out.Recipe.Steps) != 1 || len(out.Recipe.Steps[0].Ingredients) != 2 {
		t.Fatalf("got %+v", out)
	}
}

func TestImportRefusals(t *testing.T) {
	e := newEnv(t)
	e.anon().wantCode("POST", v1+"/import", app.ImportRequest{URL: "https://example.com/"}, 401, "unauthorized")
	me := e.admin()
	me.wantCode("POST", v1+"/import", app.ImportRequest{}, 400, "import_empty")
	me.wantCode("POST", v1+"/import", app.ImportRequest{URL: "file:///etc/passwd"}, 400, "import_bad_url")
	me.wantCode("POST", v1+"/import", app.ImportRequest{URL: "http://127.0.0.1/"}, 400, "import_private")
	me.wantCode("POST", v1+"/import", app.ImportRequest{URL: e.srv.URL + "/"}, 400, "import_bad_url")
	me.wantCode("POST", v1+"/import", app.ImportRequest{Text: "just one line"}, 422, "import_no_recipe")
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer blocked.Close()
	e.svc.Fetcher = importer.NewFetcher("test", true)
	me.wantCode("POST", v1+"/import", app.ImportRequest{URL: blocked.URL}, 502, "import_blocked")
}

func importUpload(t *testing.T, c *client, dryRun bool, content []byte) result {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "export.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content)
	_ = mw.Close()
	url := c.e.srv.URL + v1 + "/import/file"
	if dryRun {
		url += "?dry_run=1"
	}
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.send(req)
}

func zipOf(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	names := slices.Sorted(maps.Keys(files)) // a stable order, because order decides duplicates
	for _, name := range names {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write(files[name])
	}
	_ = w.Close()
	return buf.Bytes()
}

func TestImportFromAnotherAppsFile(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	in := simpleRecipe("Already here")
	c.createRecipe(in)
	tandoor := func(name string, withPhoto []byte) []byte {
		body := `{"name":"` + name + `","description":"D","keywords":[{"name":"Altbau"}],"working_time":5,"waiting_time":0,"servings":2,"servings_text":"","source_url":null,"steps":[{"name":"","instruction":"Do.","time":0,"ingredients":[{"food":{"name":"Zwiebel"},"unit":{"name":"EL"},"amount":"1.000","note":"","is_header":false,"no_amount":false}]}]}`
		files := map[string][]byte{"recipe.json": []byte(body)}
		if withPhoto != nil {
			files["image.png"] = withPhoto
		}
		return zipOf(t, files)
	}
	file := zipOf(t, map[string][]byte{
		"1.zip": tandoor("Neu eins", testPNG(t)),
		"2.zip": tandoor("Already Here", nil),
		"3.zip": tandoor("Kaputtes Bild", []byte("not an image")),
		"4.zip": tandoor("neu eins", nil), // the same recipe twice in one file
	})

	var preview app.FileImportResult
	importUpload(t, c, true, file).decode(t, &preview)
	if len(preview.Items) != 4 || preview.Imported != 0 || preview.Skipped != 2 {
		t.Fatalf("preview: %+v", preview)
	}
	var list app.Page[app.RecipeSummary]
	c.ok("GET", v1+"/recipes", nil, 200).decode(t, &list)
	if list.Total != 1 {
		t.Fatalf("a dry run saved %d recipes", list.Total-1)
	}

	var done app.FileImportResult
	importUpload(t, c, false, file).decode(t, &done)
	if done.Imported != 2 || done.Skipped != 2 || done.Failed != 0 {
		t.Fatalf("result: %+v", done)
	}
	var saved, broken app.FileImportItem
	for _, it := range done.Items {
		switch it.Name {
		case "Neu eins":
			saved = it
		case "Kaputtes Bild":
			broken = it
		}
	}
	if !broken.PhotoFailed || broken.RecipeID == 0 {
		t.Fatalf("a bad photo must not lose the recipe: %+v", broken)
	}
	var r app.Recipe
	c.ok("GET", path("/recipes/%d", saved.RecipeID), nil, 200).decode(t, &r)
	if r.Visibility != "private" || r.PhotoURL == "" || r.Steps[0].Ingredients[0].Food.Name != "onion" || *r.Steps[0].Ingredients[0].Amount != 1 || r.Keywords[0].Name != "Altbau" {
		t.Fatalf("recipe: %+v", r)
	}
	// Importing the same file again changes nothing.
	importUpload(t, c, false, file).decode(t, &done)
	if done.Imported != 0 || done.Skipped != 4 {
		t.Fatalf("second time: %+v", done)
	}
	if res := importUpload(t, c, false, []byte("hello")); res.status != 422 || !contains(string(res.body), "import_file_unknown") {
		t.Fatalf("not recipes: %d %s", res.status, res.body)
	}
	if res := importUpload(t, e.anon(), false, file); res.status != 401 {
		t.Fatalf("logged out: %d", res.status)
	}
	// Nothing is left behind beside the data.
	entries, err := os.ReadDir(e.cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range entries {
		if strings.HasSuffix(f.Name(), ".tmp") {
			t.Fatalf("left behind: %s", f.Name())
		}
	}
}

// Two Wokko servers exchange recipes through the export file, and nothing gets lost.
func TestWokkoExportImportsIntoAnotherWokko(t *testing.T) {
	from, to := newEnv(t), newEnv(t)
	alice := from.admin()
	amount, grams := 2.0, 200.0
	sent := alice.createRecipe(app.RecipeInput{
		Name: "Linsensuppe", Description: "Warm", Servings: 3, ServingsText: "bowls", WorkMinutes: 15, WaitMinutes: 30,
		SourceURL: "https://example.com/soup", Visibility: "household", Keywords: []string{"soup", "vegan"},
		Steps: []app.StepInput{
			{Name: "Prep", Instruction: "Chop.", Minutes: 5, Ingredients: []app.IngredientInput{{FoodName: "onion", Amount: &amount, Note: "red"}, {FoodName: "lentils", UnitName: "g", Amount: &grams}}},
			{Instruction: "Cook.", Minutes: 25, Ingredients: []app.IngredientInput{{FoodName: "salt"}}},
		},
	})
	if res := alice.upload(sent.ID, testPNG(t)); res.status != 200 {
		t.Fatal(string(res.body))
	}
	file := alice.ok("GET", v1+"/me/export", nil, 200).body

	bob := to.admin()
	var preview app.FileImportResult
	if err := json.Unmarshal(importUpload(t, bob, true, file).body, &preview); err != nil || len(preview.Items) != 1 || preview.Items[0].Format != "Wokko" || !preview.Items[0].HasPhoto {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	var result app.FileImportResult
	if err := json.Unmarshal(importUpload(t, bob, false, file).body, &result); err != nil || result.Imported != 1 {
		t.Fatalf("import: %+v %v", result, err)
	}
	var got app.Recipe
	bob.ok("GET", path("/recipes/%d", result.Items[0].RecipeID), nil, 200).decode(t, &got)
	if got.Name != "Linsensuppe" || got.Description != "Warm" || got.Servings != 3 || got.ServingsText != "bowls" || got.WorkMinutes != 15 || got.WaitMinutes != 30 || got.SourceURL != "https://example.com/soup" {
		t.Fatalf("recipe: %+v", got)
	}
	if got.Visibility != "private" || got.PhotoURL == "" || len(got.Keywords) != 2 || len(got.Steps) != 2 {
		t.Fatalf("visibility, photo, tags or steps: %+v", got)
	}
	prep, cook := got.Steps[0], got.Steps[1]
	if prep.Name != "Prep" || prep.Minutes != 5 || len(prep.Ingredients) != 2 || cook.Minutes != 25 || len(cook.Ingredients) != 1 || cook.Ingredients[0].Food.Name != "salt" {
		t.Fatalf("steps: %+v", got.Steps)
	}
	onion, lentils := prep.Ingredients[0], prep.Ingredients[1]
	if onion.Food.Name != "onion" || *onion.Amount != 2 || onion.Note != "red" || onion.Unit != nil || lentils.Unit == nil || *lentils.Amount != 200 {
		t.Fatalf("ingredients: %+v %+v", onion, lentils)
	}
	// Importing the same file again skips what's already there.
	if err := json.Unmarshal(importUpload(t, bob, false, file).body, &result); err != nil || result.Imported != 0 || result.Skipped != 1 {
		t.Fatalf("second import: %+v %v", result, err)
	}
}
