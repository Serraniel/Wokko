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
	// Substitutes name the recipe's own ingredient they apply to.
	var sub *knowledge.Finding
	for i := range res.Substitutes {
		if res.Substitutes[i].ID == "substitution.cream" {
			sub = &res.Substitutes[i]
		}
	}
	if sub == nil || !slices.Equal(sub.Foods, []string{"Wokkorahm"}) {
		t.Errorf("substitutes: %+v", res.Substitutes)
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

func TestFoodClassesCanBeSetForUnknownFoods(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	var food app.Food
	admin.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "de", Name: "Wokkocreme"}}}, 201).decode(t, &food)
	if food.KnowledgeClasses == nil || food.RecognizedClasses == nil || len(food.KnowledgeClasses) != 0 || len(food.RecognizedClasses) != 0 {
		t.Fatalf("unknown food: %+v", food)
	}
	in := simpleRecipe("Dessert")
	in.Steps = []app.StepInput{{Instruction: "Serve.", Ingredients: []app.IngredientInput{{FoodID: food.ID, Amount: ptr(100.0)}}}}
	r := admin.createRecipe(in)
	var res knowledge.Result
	admin.ok("GET", path("/recipes/%d/insights", r.ID), nil, 200).decode(t, &res)
	if !slices.Equal(res.Unclassified, []string{"Wokkocreme"}) {
		t.Fatalf("before: %+v", res)
	}

	// Someone who may edit the food says what it is.
	classes := []string{"dairy.cream"}
	admin.ok("PUT", path("/foods/%d", food.ID), app.FoodInput{Translations: food.Translations, KnowledgeClasses: &classes}, 200).decode(t, &food)
	if !slices.Equal(food.KnowledgeClasses, classes) {
		t.Fatalf("set: %+v", food)
	}
	admin.ok("GET", path("/recipes/%d/insights", r.ID), nil, 200).decode(t, &res)
	if len(res.Unclassified) != 0 || !slices.Equal(allergenIDs(res), []string{"milk"}) {
		t.Errorf("after: %+v", res)
	}
	// Leaving the classes out keeps them; unknown ones are refused.
	admin.ok("PUT", path("/foods/%d", food.ID), app.FoodInput{Translations: food.Translations}, 200).decode(t, &food)
	if !slices.Equal(food.KnowledgeClasses, classes) {
		t.Errorf("kept: %+v", food)
	}
	bad := []string{"dairy.unicorn"}
	admin.wantCode("PUT", path("/foods/%d", food.ID), app.FoodInput{Translations: food.Translations, KnowledgeClasses: &bad}, 400, "unknown_class")
	other := e.newUser(admin, "other")
	other.wantCode("PUT", path("/foods/%d", food.ID), app.FoodInput{Translations: food.Translations, KnowledgeClasses: &bad}, 403, "forbidden")

	// What Wokko recognises by name is shown too, and the list to choose from is translated.
	broth := foodNamed(t, admin, "chicken broth")
	if !slices.Equal(broth.RecognizedClasses, []string{"liquid.stock.poultry"}) {
		t.Errorf("recognised: %+v", broth.RecognizedClasses)
	}
	var list []knowledge.ClassInfo
	admin.ok("GET", v1+"/knowledge/classes?lang=de", nil, 200).decode(t, &list)
	found := false
	for _, c := range list {
		if c.ID == "dairy.cream" {
			found = c.Name == "Sahne" && slices.Equal(c.Allergens, []string{"milk"}) && c.NotVegan && !c.NotVegetarian
		}
	}
	if !found || len(list) < 100 {
		t.Errorf("classes: %d, cream ok %v", len(list), found)
	}

	// Merging keeps the classes with the surviving food.
	var twin app.Food
	admin.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "de", Name: "Wokkocreme 2"}}}, 201).decode(t, &twin)
	admin.ok("POST", v1+"/foods/merge", app.MergeInput{SourceID: food.ID, TargetID: twin.ID}, 200)
	admin.ok("GET", path("/foods/%d", twin.ID), nil, 200).decode(t, &twin)
	if !slices.Equal(twin.KnowledgeClasses, classes) {
		t.Errorf("merged: %+v", twin.KnowledgeClasses)
	}
}

func TestDraftInsightsWhileEditing(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	var before app.Page[app.Food]
	admin.ok("GET", v1+"/foods", nil, 200).decode(t, &before)
	draft := app.RecipeInput{Name: "Draft", Steps: []app.StepInput{
		{Instruction: "Whisk the yolks into the hot stock and bring to the boil.", Ingredients: []app.IngredientInput{{FoodName: "egg yolk", Amount: ptr(2.0)}, {FoodName: "vegetable broth", Amount: ptr(500.0)}, {FoodName: "Wokkopulver"}}},
	}}
	var res knowledge.Result
	admin.ok("POST", v1+"/recipes/insights", draft, 200).decode(t, &res)
	if !slices.Contains(findingIDs(res.Problems), "problem.liaison_boil") {
		t.Errorf("problems: %v", findingIDs(res.Problems))
	}
	if !slices.Equal(res.Unclassified, []string{"Wokkopulver"}) {
		t.Errorf("unclassified: %v", res.Unclassified)
	}
	// Nothing was saved: no recipe, and no new food.
	var after app.Page[app.Food]
	admin.ok("GET", v1+"/foods", nil, 200).decode(t, &after)
	if after.Total != before.Total {
		t.Errorf("foods: %d -> %d", before.Total, after.Total)
	}
	var recipes app.Page[app.RecipeSummary]
	admin.ok("GET", v1+"/recipes", nil, 200).decode(t, &recipes)
	if recipes.Total != 0 {
		t.Errorf("recipes: %d", recipes.Total)
	}
	e.anon().wantCode("POST", v1+"/recipes/insights", draft, 401, "unauthorized")
}
