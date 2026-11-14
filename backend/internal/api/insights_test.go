package api_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/knowledge"
)

func allergenIDs(res knowledge.Result) []string {
	var ids []string
	for _, a := range res.Allergens {
		if !a.Maybe {
			ids = append(ids, a.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

func findingIDs(list []knowledge.Finding) []string {
	var ids []string
	for _, f := range list {
		ids = append(ids, f.ID)
	}
	return ids
}

func TestRecipeInsights(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	// A food known here only by a made-up name, with "cream" as its alias.
	var cream app.Food
	admin.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "de", Name: "Wokkorahm"}}, Aliases: []string{"cream"}}, 201).decode(t, &cream)
	in := app.RecipeInput{Name: "Fricassee", Servings: 4, Visibility: "private", Keywords: []string{}, Steps: []app.StepInput{
		{Instruction: "Melt the butter and stir in the flour.", Minutes: 3, Ingredients: []app.IngredientInput{{FoodName: "butter", UnitName: "g", Amount: ptr(50.0)}, {FoodName: "all-purpose flour", UnitName: "g", Amount: ptr(50.0)}}},
		{Instruction: "Whisk in cold chicken broth and bring to a boil.", Minutes: 10, Ingredients: []app.IngredientInput{{FoodName: "chicken broth", UnitName: "ml", Amount: ptr(750.0)}}},
		{Instruction: "Stir in the yolks beaten with the cream.", Minutes: 2, Ingredients: []app.IngredientInput{{FoodName: "egg yolk", Amount: ptr(2.0)}, {FoodID: cream.ID, Amount: ptr(100.0)}}},
	}}
	r := admin.createRecipe(in)

	var res knowledge.Result
	admin.ok("GET", path("/recipes/%d/insights", r.ID), nil, 200).decode(t, &res)
	if !slices.Contains(findingIDs(res.Sauces), "sauce.allemande") || !slices.Contains(findingIDs(res.Techniques), "technique.roux") {
		t.Fatalf("sauces %v, techniques %v", findingIDs(res.Sauces), findingIDs(res.Techniques))
	}
	if got := allergenIDs(res); !slices.Equal(got, []string{"eggs", "gluten", "milk"}) {
		t.Errorf("allergens: %v", got)
	}
	if res.Diet.Vegetarian == nil || *res.Diet.Vegetarian {
		t.Errorf("chicken broth isn't vegetarian: %+v", res.Diet)
	}
	if len(res.Unclassified) != 0 {
		t.Errorf("unclassified: %v", res.Unclassified)
	}
	// The viewer's language decides the text.
	admin.ok("GET", path("/recipes/%d/insights?lang=de", r.ID), nil, 200).decode(t, &res)
	for _, s := range res.Sauces {
		if s.ID == "sauce.veloute" && !strings.Contains(s.Text, "Grundsoßen") {
			t.Errorf("german text: %s", s.Text)
		}
	}

	// Only people who may see the recipe get its insights.
	other := e.newUser(admin, "other")
	other.wantCode("GET", path("/recipes/%d/insights", r.ID), nil, 404, "not_found")
	anon := e.anon()
	anon.wantCode("GET", path("/public/recipes/%d/insights", r.ID), nil, 404, "not_found")
	var link app.ShareLink
	admin.ok("POST", path("/recipes/%d/sharing/links", r.ID), map[string]any{}, 201).decode(t, &link)
	token := link.URL[strings.LastIndex(link.URL, "/")+1:]
	anon.ok("GET", path("/public/share/%s/insights", token), nil, 200).decode(t, &res)
	if !slices.Contains(findingIDs(res.Sauces), "sauce.allemande") {
		t.Errorf("shared: %v", findingIDs(res.Sauces))
	}
	anon.wantCode("GET", path("/public/share/%s/insights", "nope"), nil, 404, "not_found")

	// Assistants can read them too.
	enableMCP(t, admin)
	s := connect(t, e, newToken(t, admin, "read"))
	var viaMCP knowledge.Result
	call(t, s, "get_recipe_insights", map[string]any{"id": r.ID}, &viaMCP)
	if !slices.Equal(allergenIDs(viaMCP), []string{"eggs", "gluten", "milk"}) {
		t.Errorf("mcp: %+v", viaMCP.Allergens)
	}
}
