package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// unzip returns an archive's files by name.
func unzip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = b
	}
	return out
}

func soupWithPhoto(t *testing.T, c *client, visibility string) app.Recipe {
	t.Helper()
	two, grams := 2.0, 200.0
	r := c.createRecipe(app.RecipeInput{
		Name: "Rote Linsensuppe", Description: "Warm", Servings: 3, ServingsText: "bowls", WorkMinutes: 15, WaitMinutes: 30,
		Visibility: visibility, Keywords: []string{"soup"},
		Steps: []app.StepInput{
			{Name: "Prep", Instruction: "Chop the onions.", Ingredients: []app.IngredientInput{{FoodName: "onion", Amount: &two, Note: "red"}, {FoodName: "lentils", UnitName: "g", Amount: &grams}}},
			{Instruction: "Cook everything.", Ingredients: []app.IngredientInput{{FoodName: "salt"}}},
		},
	})
	if res := c.upload(r.ID, testPNG(t)); res.status != 200 {
		t.Fatal(string(res.body))
	}
	return r
}

func TestExportOneRecipeInBothFormats(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	r := soupWithPhoto(t, admin, "household")

	// Wokko's own format, for someone else in the household too.
	res := member.ok("GET", path("/recipes/%d/export", r.ID), nil, 200)
	if got := res.header.Get("Content-Disposition"); !strings.Contains(got, `filename="rote-linsensuppe-wokko.zip"`) {
		t.Fatalf("file name: %q", got)
	}
	files := unzip(t, res.body)
	var data app.ExportData
	if err := json.Unmarshal(files["recipes.json"], &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Recipes) != 1 || data.User != nil || data.Recipes[0].Photo == "" || files[data.Recipes[0].Photo] == nil || len(data.Recipes[0].Steps) != 2 {
		t.Fatalf("wokko export: %+v %v", data, keys(files))
	}

	// schema.org, in Nextcloud Cookbook's layout.
	files = unzip(t, admin.ok("GET", path("/recipes/%d/export?format=schema", r.ID), nil, 200).body)
	if files["Rote Linsensuppe/full.jpg"] == nil {
		t.Fatalf("schema files: %v", keys(files))
	}
	var doc map[string]any
	if err := json.Unmarshal(files["Rote Linsensuppe/recipe.json"], &doc); err != nil {
		t.Fatal(err)
	}
	ingredients, _ := doc["recipeIngredient"].([]any)
	steps, _ := doc["recipeInstructions"].([]any)
	if doc["@type"] != "Recipe" || doc["recipeYield"] != "3 bowls" || doc["prepTime"] != "PT15M" || doc["cookTime"] != "PT30M" || !strings.EqualFold(doc["keywords"].(string), "soup") || len(ingredients) != 3 || len(steps) != 2 {
		t.Fatalf("schema recipe: %v", doc)
	}
	if ingredients[0] != "2 onions, red" && ingredients[0] != "2 onion, red" {
		t.Fatalf("ingredient line: %v", ingredients)
	}

	// Nobody exports what they can't see, and only the formats that exist.
	stranger.wantCode("GET", path("/recipes/%d/export", r.ID), nil, 404, "not_found")
	e.anon().wantCode("GET", path("/recipes/%d/export", r.ID), nil, 401, "unauthorized")
	admin.wantCode("GET", path("/recipes/%d/export?format=pdf", r.ID), nil, 400, "bad_request")
}

func TestExportThroughAShareLinkAndThePublicGallery(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	private := soupWithPhoto(t, admin, "private")
	var link app.ShareLink
	admin.ok("POST", path("/recipes/%d/sharing/links", private.ID), map[string]any{}, 201).decode(t, &link)
	token := link.URL[strings.LastIndex(link.URL, "/")+1:]

	anon := e.anon()
	files := unzip(t, anon.ok("GET", path("/public/share/%s/export", token), nil, 200).body)
	if files["recipes.json"] == nil || len(files) != 2 {
		t.Fatalf("shared export: %v", keys(files))
	}
	anon.wantCode("GET", path("/public/share/%s/export", "nope"), nil, 404, "not_found")
	anon.wantCode("GET", path("/public/recipes/%d/export", private.ID), nil, 404, "not_found")

	public := admin.createRecipe(app.RecipeInput{Name: "Brot", Servings: 1, Visibility: "public", Keywords: []string{}, Steps: []app.StepInput{{Instruction: "Backen."}}})
	files = unzip(t, anon.ok("GET", path("/public/recipes/%d/export?format=schema", public.ID), nil, 200).body)
	if files["Brot/recipe.json"] == nil {
		t.Fatalf("public export: %v", keys(files))
	}
}

// What one Wokko exports, another imports: losslessly in Wokko's format, and the basics in schema.org.
func TestExportedRecipesImportElsewhere(t *testing.T) {
	from, to := newEnv(t), newEnv(t)
	alice := from.admin()
	r := soupWithPhoto(t, alice, "private")
	bob := to.admin()
	for _, format := range []string{"wokko", "schema"} {
		file := alice.ok("GET", path("/recipes/%d/export?format=%s", r.ID, format), nil, 200).body
		var result app.FileImportResult
		if err := json.Unmarshal(importUpload(t, bob, false, file).body, &result); err != nil || result.Imported != 1 {
			t.Fatalf("%s import: %+v %v", format, result, err)
		}
		var got app.Recipe
		bob.ok("GET", path("/recipes/%d", result.Items[0].RecipeID), nil, 200).decode(t, &got)
		count := 0
		for _, st := range got.Steps {
			count += len(st.Ingredients)
		}
		if got.Servings != 3 || got.WorkMinutes != 15 || got.WaitMinutes != 30 || got.PhotoURL == "" || count != 3 || len(got.Keywords) != 1 {
			t.Fatalf("%s: %+v", format, got)
		}
		if format == "wokko" && (len(got.Steps) != 2 || len(got.Steps[0].Ingredients) != 2) {
			t.Fatalf("wokko keeps each step's ingredients: %+v", got.Steps)
		}
		bob.ok("DELETE", path("/recipes/%d", got.ID), nil, 200)
		bob.ok("DELETE", path("/trash/%d", got.ID), nil, 200)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
