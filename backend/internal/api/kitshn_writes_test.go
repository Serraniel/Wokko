package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/importer"
)

// kitshnSend makes a call the way kitshn does and returns the decoded JSON, if any.
func kitshnSend(t *testing.T, e *env, cookie, method, path, body string, want int) map[string]any {
	t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.StatusCode, want, raw)
	}
	if want == 204 && len(raw) != 0 {
		t.Fatalf("%s %s: a 204 must have an empty body, or kitshn's HTTP client fails: %q", method, path, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s %s: not a JSON object: %s", method, path, raw)
		}
	}
	return out
}

func kitshnSetup(t *testing.T) (*env, *client, string) {
	t.Helper()
	e := newEnv(t)
	admin := e.admin()
	return e, admin, kitshnLogin(t, e, "admin", "password123")
}

func idOf(t *testing.T, m map[string]any) int64 {
	t.Helper()
	n, ok := m["id"].(float64)
	if !ok {
		t.Fatalf("no id in %v", m)
	}
	return int64(n)
}

func TestKitshnCreatesAndEditsRecipes(t *testing.T) {
	e, admin, c := kitshnSetup(t)
	// kitshn's "new recipe" flow starts with exactly this body.
	rec := kitshnSend(t, e, c, "POST", "/api/recipe/", `{"name":"New recipe","description":"This recipe is currently being created within the kitshn app.","steps":[],"internal":true}`, 201)
	requireFields(t, "recipe", rec, "id", "name", "keywords", "steps", "working_time", "waiting_time", "created_at", "updated_at", "internal", "properties", "food_properties", "servings", "servings_text")
	id := idOf(t, rec)

	// Then it saves the editor's contents, ingredients and all.
	body := `{"name":"Goulash","description":"Slow","keywords":[{"name":"Beef","description":""}],"working_time":20,"waiting_time":90,"servings":6,"servings_text":"","source_url":"https://example.com/g",
	"steps":[{"name":"Fry","instruction":"Brown it.","time":10,"order":0,"show_as_header":false,"show_ingredients_table":true,
	  "ingredients":[{"food":{"name":"Testbeef"},"unit":{"name":"g"},"amount":500,"note":"cubed","order":0,"is_header":false,"no_amount":false},
	                 {"food":{"name":"salt"},"unit":null,"amount":0,"note":"","order":1,"is_header":false,"no_amount":true},
	                 {"food":null,"unit":null,"amount":0,"note":"For the sauce","order":2,"is_header":true,"no_amount":true}]},
	  {"name":"","instruction":"Simmer.","time":80,"order":1,"ingredients":[]}]}`
	rec = kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/recipe/%d/", id), body, 200)
	if rec["name"] != "Goulash" || rec["waiting_time"].(float64) != 90 || rec["servings"].(float64) != 6 {
		t.Fatalf("patched: %v", rec)
	}
	steps := rec["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("steps: %v", steps)
	}
	first := steps[0].(map[string]any)
	ings := first["ingredients"].([]any)
	if len(ings) != 2 {
		t.Fatalf("ingredients, without the header row: %v", ings)
	}
	if !ings[1].(map[string]any)["no_amount"].(bool) {
		t.Fatalf("no_amount: %v", ings[1])
	}

	// A partial change leaves the rest alone, and the web app sees it all.
	rec = kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/recipe/%d/", id), `{"description":"Very slow"}`, 200)
	if rec["name"] != "Goulash" || len(rec["steps"].([]any)) != 2 {
		t.Fatalf("partial: %v", rec)
	}
	stepID := int64(first["id"].(float64))
	got := kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/step/%d/", stepID), `{"instruction":"Brown it well.","time":12}`, 200)
	if got["instruction"] != "Brown it well." || got["id"].(float64) != float64(stepID) {
		t.Fatalf("step: %v", got)
	}
	var web app.Recipe
	admin.ok("GET", path("/recipes/%d", id), nil, 200).decode(t, &web)
	if web.Steps[0].Instruction != "Brown it well." || web.Steps[0].Minutes != 12 || len(web.Steps[0].Ingredients) != 2 || web.Description != "Very slow" {
		t.Fatalf("web: %+v", web)
	}
	var versions []app.RecipeVersion
	admin.ok("GET", path("/recipes/%d/versions", id), nil, 200).decode(t, &versions)
	if len(versions) < 3 {
		t.Fatalf("kitshn edits keep versions too: %d", len(versions))
	}

	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/step/%d/", int64(steps[1].(map[string]any)["id"].(float64))), "", 204)
	admin.ok("GET", path("/recipes/%d", id), nil, 200).decode(t, &web)
	if len(web.Steps) != 1 {
		t.Fatalf("after deleting a step: %+v", web.Steps)
	}

	link := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/share-link/%d", id), "", 200)
	if !strings.Contains(link["link"].(string), "/s/") {
		t.Fatalf("link: %v", link)
	}

	// Deleting goes to the trash, like everywhere else.
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/recipe/%d/", id), "", 204)
	var trash []app.RecipeSummary
	admin.ok("GET", v1+"/trash", nil, 200).decode(t, &trash)
	if len(trash) != 1 {
		t.Fatalf("trash: %+v", trash)
	}

	// A failed save changes nothing.
	mine := kitshnSend(t, e, c, "POST", "/api/recipe/", `{"name":"Keep"}`, 201)
	kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/recipe/%d/", idOf(t, mine)), `{"name":"  "}`, 400)
}

func TestKitshnPermissionsApplyToWrites(t *testing.T) {
	e, admin, _ := kitshnSetup(t)
	e.newUser(admin, "stranger")
	other := kitshnLogin(t, e, "stranger", "password123")
	in := simpleRecipe("Private")
	in.Visibility = "private"
	r := admin.createRecipe(in)
	kitshnSend(t, e, other, "PATCH", fmt.Sprintf("/api/recipe/%d/", r.ID), `{"name":"Stolen"}`, 404)
	kitshnSend(t, e, other, "DELETE", fmt.Sprintf("/api/recipe/%d/", r.ID), "", 404)
	kitshnSend(t, e, other, "PUT", fmt.Sprintf("/api/recipe/%d/shopping/", r.ID), `{"servings":2}`, 404)
	kitshnSend(t, e, other, "GET", fmt.Sprintf("/api/share-link/%d", r.ID), "", 404)
	if res, err := http.Post(e.srv.URL+"/api/recipe/", "application/json", strings.NewReader(`{"name":"x"}`)); err != nil {
		t.Fatal(err)
	} else if res.Body.Close(); res.StatusCode != 403 {
		t.Fatalf("logged out: %d", res.StatusCode)
	}
}

func TestKitshnImagesAndUnknownPaths(t *testing.T) {
	e, _, c := kitshnSetup(t)
	id := idOf(t, kitshnSend(t, e, c, "POST", "/api/recipe/", `{"name":"Photo"}`, 201))
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("image", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(testPNG(t))
	_ = mw.Close()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/recipe/%d/image/", e.srv.URL, id), &buf)
	req.Header.Set("Cookie", c)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("upload: %d", res.StatusCode)
	}
	got := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe/%d/", id), "", 200)
	if got["image"] == nil {
		t.Fatalf("no image: %v", got)
	}
	// Features Wokko doesn't have fail as JSON, not as the web app's HTML (K-7).
	got = kitshnSend(t, e, c, "POST", "/api/ai-import/", `{}`, 404)
	if got["detail"] == nil {
		t.Fatalf("detail: %v", got)
	}
}

func TestKitshnImportsFromALink(t *testing.T) {
	e, _, c := kitshnSetup(t)
	e.svc.Fetcher = importer.NewFetcher("test", true)
	photo := testPNG(t)
	var site *httptest.Server
	site = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/p.png" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
			return
		}
		fmt.Fprintf(w, `<script type="application/ld+json">{"@type":"Recipe","name":"Dal","image":"%s/p.png","recipeYield":"4","recipeIngredient":["200 g Linsen"],"recipeInstructions":"Kochen.\n\nWürzen."}</script>`, site.URL)
	}))
	defer site.Close()
	got := kitshnSend(t, e, c, "POST", "/api/recipe-from-source/", fmt.Sprintf(`{"url":%q,"data":""}`, site.URL+"/r"), 200)
	requireFields(t, "from-source", got, "recipe", "images", "error", "duplicates")
	draft := got["recipe"].(map[string]any)
	requireFields(t, "from-source recipe", draft, "steps", "internal", "source_url", "name", "description", "servings", "servings_text", "working_time", "waiting_time", "image_url", "keywords")
	first := draft["steps"].([]any)[0].(map[string]any)
	requireFields(t, "from-source step", first, "instruction", "ingredients", "show_ingredients_table")
	requireFields(t, "from-source ingredient", first["ingredients"].([]any)[0], "amount", "food", "unit", "note", "original_text")

	// kitshn saves what it received, with the chosen image.
	raw, _ := json.Marshal(map[string]any{"name": draft["name"], "steps": draft["steps"], "servings": draft["servings"], "image": draft["image_url"], "keywords": draft["keywords"]})
	saved := kitshnSend(t, e, c, "POST", "/api/recipe/", string(raw), 201)
	if saved["image"] == nil || saved["name"] != "Dal" || len(saved["steps"].([]any)) != 2 {
		t.Fatalf("saved: %v", saved)
	}
	// A failure is reported the way kitshn expects, not as an HTTP error.
	e.svc.Fetcher = importer.NewFetcher("test", false)
	failed := kitshnSend(t, e, c, "POST", "/api/recipe-from-source/", `{"url":"http://127.0.0.1/","data":""}`, 200)
	if failed["error"] != true || failed["msg"] == "" {
		t.Fatalf("failed: %v", failed)
	}
}

func TestKitshnMealPlansKeepTheDayChosen(t *testing.T) {
	e, _, c := kitshnSetup(t)
	types := kitshnSend(t, e, c, "GET", "/api/meal-type/", "", 200)
	results := types["results"].([]any)
	if len(results) != 4 {
		t.Fatalf("types: %v", types)
	}
	requireFields(t, "meal type", results[0], "id", "name", "order", "time", "color", "created_by")
	dinner, _ := json.Marshal(results[2])
	r := kitshnSend(t, e, c, "POST", "/api/recipe/", `{"name":"Soup"}`, 201)
	overview, _ := json.Marshal(r)

	// Germany in November: local midnight of the 9th is 23:00 UTC on the 8th.
	body := fmt.Sprintf(`{"title":"","recipe":%s,"servings":3.0,"note":"Hot","from_date":"2026-11-08T23:00:00Z","meal_type":%s,"addshopping":false}`, overview, dinner)
	plan := kitshnSend(t, e, c, "POST", "/api/meal-plan/", body, 201)
	requireFields(t, "meal plan", plan, "id", "title", "recipe", "servings", "note", "note_markdown", "from_date", "to_date", "meal_type", "created_by", "recipe_name", "meal_type_name", "shopping")
	if plan["from_date"] != "2026-11-09" || plan["to_date"] != "2026-11-09" {
		t.Fatalf("the meal moved to another day: %v %v", plan["from_date"], plan["to_date"])
	}
	// New York in November: local midnight of the 9th is 05:00 UTC on the 9th.
	id := idOf(t, plan)
	plan = kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/meal-plan/%d/", id), `{"from_date":"2026-11-10T05:00:00Z","to_date":"2026-11-11T05:00:00Z","note":"Hotter"}`, 200)
	if plan["from_date"] != "2026-11-10" || plan["to_date"] != "2026-11-11" || plan["note"] != "Hotter" || plan["recipe"] == nil {
		t.Fatalf("patched: %v", plan)
	}
	list := kitshnSend(t, e, c, "GET", "/api/meal-plan/?from_date=2026-11-11&to_date=2026-11-12", "", 200)
	if list["count"].(float64) != 1 {
		t.Fatalf("list: %v", list)
	}
	empty := kitshnSend(t, e, c, "GET", "/api/meal-plan/?from_date=2027-01-01&to_date=2027-01-02", "", 200)
	if empty["count"].(float64) != 0 {
		t.Fatalf("empty: %v", empty)
	}
	kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/meal-plan/%d/", id), "", 200)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/meal-plan/%d/", id), "", 204)

	made := kitshnSend(t, e, c, "POST", "/api/meal-type/", `{"name":"Brunch","order":9,"time":"10:30","color":"#ff0000"}`, 201)
	changed := kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/meal-type/%d/", idOf(t, made)), `{"name":"Late brunch"}`, 200)
	if changed["name"] != "Late brunch" || changed["time"] != "10:30" {
		t.Fatalf("meal type: %v", changed)
	}
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/meal-type/%d/", idOf(t, made)), "", 204)
}

func TestKitshnBooks(t *testing.T) {
	e, _, c := kitshnSetup(t)
	r := kitshnSend(t, e, c, "POST", "/api/recipe/", `{"name":"Soup"}`, 201)
	book := kitshnSend(t, e, c, "POST", "/api/recipe-book/", `{"name":"Winter","description":"","order":0,"filter":null,"shared":[]}`, 201)
	requireFields(t, "book", book, "id", "name", "description", "order", "filter", "shared")
	bid := idOf(t, book)
	kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/recipe-book/%d/", bid), `{"description":"Warm"}`, 200)
	entry := kitshnSend(t, e, c, "POST", "/api/recipe-book-entry/", fmt.Sprintf(`{"book":%d,"recipe":%d,"order":0}`, bid, idOf(t, r)), 201)
	requireFields(t, "entry", entry, "id", "book", "book_content", "recipe", "recipe_content")
	requireFields(t, "entry recipe", entry["recipe_content"], "id", "name", "keywords", "working_time", "waiting_time", "created_at", "updated_at", "internal", "servings", "servings_text")
	list := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe-book-entry/?book=%d", bid), "", 200)
	if list["count"].(float64) != 1 {
		t.Fatalf("entries: %v", list)
	}
	byRecipe := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe-book-entry/?recipe=%d", idOf(t, r)), "", 200)
	if byRecipe["count"].(float64) != 1 {
		t.Fatalf("by recipe: %v", byRecipe)
	}
	// The book filter on recipes works too.
	inBook := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe/?books=%d", bid), "", 200)
	if inBook["count"].(float64) != 1 {
		t.Fatalf("recipes in book: %v", inBook)
	}
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/recipe-book-entry/%d/", idOf(t, entry)), "", 204)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/recipe-book/%d/", bid), "", 204)
	kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe/%d/", idOf(t, r)), "", 200)
}

func TestKitshnShopping(t *testing.T) {
	e, admin, c := kitshnSetup(t)
	amount := 100.0
	in := simpleRecipe("Rice")
	in.Servings = 2
	in.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "rice", UnitName: "g", Amount: &amount}}}}
	r := admin.createRecipe(in)

	item := kitshnSend(t, e, c, "POST", "/api/shopping-list-entry/", `{"amount":2.0,"checked":false,"food":{"name":"Testmilk"},"unit":{"name":"l"},"shopping_lists":[]}`, 201)
	requireFields(t, "entry", item, "id", "food", "unit", "amount", "order", "checked", "created_by", "shopping_lists")
	requireFields(t, "entry person", item["created_by"], "id", "username", "display_name")
	requireFields(t, "entry food", item["food"], "id", "name")
	id := idOf(t, item)

	// One way kitshn adds a recipe: a group first, then its entries. Expect exactly 204.
	group := kitshnSend(t, e, c, "POST", "/api/shopping-list-recipe/", fmt.Sprintf(`{"recipe":%d,"servings":4.0}`, r.ID), 201)
	gid := idOf(t, group)
	entries := fmt.Sprintf(`{"entries":[{"amount":200.0,"unit_id":%d,"food_id":%d,"ingredient_id":%d}],"shopping_lists_ids":[]}`, r.Steps[0].Ingredients[0].Unit.ID, r.Steps[0].Ingredients[0].Food.ID, r.Steps[0].Ingredients[0].ID)
	kitshnSend(t, e, c, "POST", fmt.Sprintf("/api/shopping-list-recipe/%d/bulk_create_entries/", gid), entries, 204)
	// The other way: one call.
	kitshnSend(t, e, c, "PUT", fmt.Sprintf("/api/recipe/%d/shopping/", r.ID), `{"servings":1.0}`, 204)

	list := kitshnSend(t, e, c, "GET", "/api/shopping-list-entry/?page_size=50", "", 200)
	if list["count"].(float64) != 3 {
		t.Fatalf("list: %v", list)
	}
	grouped := list["results"].([]any)[1].(map[string]any)
	if grouped["amount"].(float64) != 200 {
		t.Fatalf("grouped: %v", grouped)
	}
	data := grouped["list_recipe_data"].(map[string]any)
	requireFields(t, "recipe data", data, "id", "name", "recipe", "recipe_data", "mealplan", "meal_plan_data", "servings")
	scaled := list["results"].([]any)[2].(map[string]any)
	if scaled["amount"].(float64) != 50 {
		t.Fatalf("scaled to 1 of 2 servings: %v", scaled)
	}

	kitshnSend(t, e, c, "POST", "/api/shopping-list-entry/bulk/", fmt.Sprintf(`{"ids":[%d],"checked":true}`, id), 200)
	changed := kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/shopping-list-entry/%d/", id), `{"amount":3.0}`, 200)
	if changed["amount"].(float64) != 3 || changed["checked"] != true || changed["unit"] == nil {
		t.Fatalf("patch: %v", changed)
	}
	cleared := kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/shopping-list-entry/%d/", id), `{"unit":null}`, 200)
	if cleared["unit"] != nil {
		t.Fatalf("unit not cleared: %v", cleared)
	}
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/shopping-list-entry/%d/", id), "", 204)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/shopping-list-recipe/%d/", gid), "", 204)
	if left := kitshnSend(t, e, c, "GET", "/api/shopping-list-entry/", "", 200); left["count"].(float64) != 1 {
		t.Fatalf("left: %v", left)
	}

	cat := kitshnSend(t, e, c, "POST", "/api/supermarket-category/", `{"name":"Dairy"}`, 201)
	requireFields(t, "category", cat, "id", "name", "description")
	market := kitshnSend(t, e, c, "POST", "/api/supermarket/", `{"name":"Corner"}`, 201)
	requireFields(t, "supermarket", market, "id", "name", "description", "category_to_supermarket")
	food := kitshnSend(t, e, c, "POST", "/api/food/", `{"name":"Testcream"}`, 201)
	body := fmt.Sprintf(`{"id":%d,"name":"Testcream","supermarket_category":{"id":%d,"name":"Dairy","description":""}}`, idOf(t, food), idOf(t, cat))
	patched := kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/food/%d/", idOf(t, food)), body, 200)
	if patched["supermarket_category"] == nil {
		t.Fatalf("category: %v", patched)
	}
	got := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/food/%d/", idOf(t, food)), "", 200)
	if got["supermarket_category"] == nil {
		t.Fatalf("read back: %v", got)
	}
	kitshnSend(t, e, c, "PATCH", fmt.Sprintf("/api/food/%d/", idOf(t, food)), fmt.Sprintf(`{"id":%d,"name":"Testcream","supermarket_category":null}`, idOf(t, food)), 200)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/food/%d/", idOf(t, food)), "", 204)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/supermarket/%d/", idOf(t, market)), "", 204)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/supermarket-category/%d/", idOf(t, cat)), "", 204)
	unit := kitshnSend(t, e, c, "POST", "/api/unit/", `{"name":"Testscoop"}`, 201)
	kitshnSend(t, e, c, "DELETE", fmt.Sprintf("/api/unit/%d/", idOf(t, unit)), "", 204)
	if k := kitshnSend(t, e, c, "POST", "/api/keyword/", `{"name":"Testtag","description":""}`, 201); k["name"] != "Testtag" {
		t.Fatalf("keyword: %v", k)
	}
}

func TestKitshnRecipeFiltersAndNutrition(t *testing.T) {
	e, admin, c := kitshnSetup(t)
	amount := 100.0
	a := simpleRecipe("Rated")
	a.Keywords = []string{"Soup"}
	a.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "rice", UnitName: "g", Amount: &amount}}}}
	ra := admin.createRecipe(a)
	rb := admin.createRecipe(simpleRecipe("Plain"))
	kitshnSend(t, e, c, "POST", "/api/cook-log/", fmt.Sprintf(`{"recipe":%d,"rating":5,"servings":2,"comment":""}`, ra.ID), 201)

	count := func(q string) float64 {
		t.Helper()
		return kitshnSend(t, e, c, "GET", "/api/recipe/?"+q, "", 200)["count"].(float64)
	}
	kw := ra.Keywords[0].ID
	food := ra.Steps[0].Ingredients[0].Food.ID
	for q, want := range map[string]float64{"": 2, "rating_gte=4": 1, "timescooked_gte=1": 1, fmt.Sprintf("foods=%d", food): 1, fmt.Sprintf("keywords_or=%d&keywords_or=%d", kw, 9999): 1, "keywords=9999": 0} {
		if got := count(q); got != want {
			t.Errorf("%q: got %v, want %v", q, got, want)
		}
	}
	order := func(sort string) string {
		t.Helper()
		r := kitshnSend(t, e, c, "GET", "/api/recipe/?sort_order="+sort, "", 200)
		return r["results"].([]any)[0].(map[string]any)["name"].(string)
	}
	if order("-rating") != "Rated" || order("-favorite") != "Rated" || order("-lastcooked") != "Rated" || order("name") != "Plain" || order("-name") != "Rated" {
		t.Errorf("sorting: %s %s %s %s %s", order("-rating"), order("-favorite"), order("-lastcooked"), order("name"), order("-name"))
	}
	_ = rb
	// Nutrition from the seed data reaches kitshn in its own shape.
	rec := kitshnSend(t, e, c, "GET", fmt.Sprintf("/api/recipe/%d/", ra.ID), "", 200)
	props := rec["food_properties"].(map[string]any)
	energy, ok := props["1"].(map[string]any)
	if !ok {
		t.Fatalf("no energy: %v", props)
	}
	requireFields(t, "food property", energy, "id", "name", "unit", "order", "food_values", "total_value", "missing_value")
	if energy["total_value"].(float64) < 300 {
		t.Fatalf("100 g of rice: %v", energy)
	}
}
