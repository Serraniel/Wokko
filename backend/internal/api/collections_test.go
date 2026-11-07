package api_test

import (
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

func TestStepIDsSurviveEdits(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	in := simpleRecipe("Soup")
	in.Steps = []app.StepInput{{Instruction: "One"}, {Instruction: "Two"}, {Instruction: "Three"}}
	r := c.createRecipe(in)
	first, second := r.Steps[0].ID, r.Steps[1].ID

	in.BaseVersion = r.Version
	in.Steps = []app.StepInput{{Instruction: "One changed", Ingredients: []app.IngredientInput{{FoodName: "onion"}}}, {Instruction: "Two changed"}}
	var saved app.Recipe
	c.ok("PUT", path("/recipes/%d", r.ID), in, 200).decode(t, &saved)
	if len(saved.Steps) != 2 || saved.Steps[0].ID != first || saved.Steps[1].ID != second || saved.Steps[0].Instruction != "One changed" || len(saved.Steps[0].Ingredients) != 1 {
		t.Fatalf("steps: %+v", saved.Steps)
	}
	in.BaseVersion = saved.Version
	in.Steps = []app.StepInput{{Instruction: "Only"}, {Instruction: "B"}, {Instruction: "C"}}
	c.ok("PUT", path("/recipes/%d", r.ID), in, 200).decode(t, &saved)
	if len(saved.Steps) != 3 || saved.Steps[0].ID != first || len(saved.Steps[0].Ingredients) != 0 {
		t.Fatalf("after adding: %+v", saved.Steps)
	}
}

func TestRenamingAFoodKeepsRecipesFindable(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	in := simpleRecipe("Stew")
	in.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "Blaukraut"}}}}
	r := c.createRecipe(in)
	food := r.Steps[0].Ingredients[0].Food.ID
	c.ok("PUT", path("/foods/%d", food), app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "Red cabbage"}}}, 200)
	var p app.Page[app.RecipeSummary]
	c.ok("GET", v1+"/recipes?q=cabbage", nil, 200).decode(t, &p)
	if p.Total != 1 {
		t.Fatalf("not found under the new name: %+v", p)
	}
}

func TestBooksAreSharedInAHouseholdAndKeepPrivateRecipesPrivate(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	stranger := e.newUser(admin, "stranger")
	var inv app.Invite
	admin.ok("POST", v1+"/household/invites", map[string]any{"max_uses": 1}, 201).decode(t, &inv)
	member := e.anon()
	member.ok("POST", v1+"/auth/register", map[string]string{"handle": "member", "password": "password123", "invite": inv.Token}, 201)

	secret := simpleRecipe("Secret")
	secret.Visibility = "private"
	s := admin.createRecipe(secret)
	shared := simpleRecipe("Shared")
	shared.Visibility = "household"
	o := admin.createRecipe(shared)

	var b app.RecipeBook
	admin.ok("POST", v1+"/books", app.RecipeBookInput{Name: " Weeknight "}, 201).decode(t, &b)
	if b.Name != "Weeknight" {
		t.Fatalf("name: %q", b.Name)
	}
	admin.wantCode("POST", v1+"/books", app.RecipeBookInput{Name: " "}, 400, "name_required")
	admin.ok("PUT", path("/books/%d/recipes/%d", b.ID, s.ID), nil, 200)
	admin.ok("PUT", path("/books/%d/recipes/%d", b.ID, o.ID), nil, 200)
	admin.ok("PUT", path("/books/%d/recipes/%d", b.ID, o.ID), nil, 200) // adding twice is harmless

	var d app.RecipeBookDetail
	admin.ok("GET", path("/books/%d", b.ID), nil, 200).decode(t, &d)
	if len(d.Recipes) != 2 || d.RecipeCount != 2 {
		t.Fatalf("owner sees: %+v", d)
	}
	member.ok("GET", path("/books/%d", b.ID), nil, 200).decode(t, &d)
	if len(d.Recipes) != 1 || d.Recipes[0].Name != "Shared" || d.RecipeCount != 1 {
		t.Fatalf("household member sees: %+v", d)
	}
	member.wantCode("PUT", path("/books/%d/recipes/%d", b.ID, s.ID), nil, 404, "not_found")
	stranger.wantCode("GET", path("/books/%d", b.ID), nil, 404, "not_found")
	stranger.wantCode("DELETE", path("/books/%d", b.ID), nil, 404, "not_found")
	var books []app.RecipeBook
	stranger.ok("GET", v1+"/books", nil, 200).decode(t, &books)
	if len(books) != 0 {
		t.Fatalf("stranger lists: %+v", books)
	}
	var in []int64
	admin.ok("GET", path("/recipes/%d/books", o.ID), nil, 200).decode(t, &in)
	if len(in) != 1 || in[0] != b.ID {
		t.Fatalf("books for recipe: %v", in)
	}

	// A trashed recipe leaves the list, and deleting a book leaves recipes alone.
	admin.ok("DELETE", path("/recipes/%d", s.ID), nil, 200)
	admin.ok("GET", path("/books/%d", b.ID), nil, 200).decode(t, &d)
	if len(d.Recipes) != 1 {
		t.Fatalf("after trash: %+v", d)
	}
	admin.ok("POST", path("/recipes/%d/restore", s.ID), nil, 200)
	admin.ok("DELETE", path("/books/%d", b.ID), nil, 200)
	admin.ok("GET", path("/recipes/%d", o.ID), nil, 200)
}

func TestCollectionLinkShowsOnlyWhatIsNotPrivate(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	other := e.newUser(admin, "other")
	shared := simpleRecipe("Shared one")
	shared.Visibility = "household"
	s := admin.createRecipe(shared)
	secret := simpleRecipe("Hidden one")
	secret.Visibility = "private"
	h := admin.createRecipe(secret)
	var photo app.Recipe
	admin.upload(s.ID, testPNG(t)).decode(t, &photo)
	admin.upload(h.ID, testPNG(t))
	var hiddenFull app.Recipe
	admin.ok("GET", path("/recipes/%d", h.ID), nil, 200).decode(t, &hiddenFull)
	foreign := simpleRecipe("Not mine")
	foreign.Visibility = "household"
	f := other.createRecipe(foreign)

	var l app.CollectionLink
	admin.ok("POST", v1+"/me/collection-links", map[string]any{}, 201).decode(t, &l)
	token := l.URL[strings.LastIndex(l.URL, "/")+1:]
	anon := e.anon()
	var p app.Page[app.RecipeSummary]
	anon.ok("GET", path("/public/collections/%s", token), nil, 200).decode(t, &p)
	if p.Total != 1 || p.Items[0].Name != "Shared one" || p.Items[0].Owner != nil {
		t.Fatalf("list: %+v", p)
	}
	anon.ok("GET", p.Items[0].ThumbURL, nil, 200)
	var got app.Recipe
	anon.ok("GET", path("/public/collections/%s/recipes/%d", token, s.ID), nil, 200).decode(t, &got)
	if got.Owner != nil || got.CanEdit {
		t.Fatalf("author details leaked: %+v", got)
	}
	anon.wantCode("GET", path("/public/collections/%s/recipes/%d", token, h.ID), nil, 404, "not_found")
	anon.wantCode("GET", path("/public/collections/%s/recipes/%d", token, f.ID), nil, 404, "not_found")
	if res := anon.do("GET", hiddenFull.PhotoURL+"?share="+token, nil); res.status != 404 {
		t.Fatalf("private photo through the link: %d", res.status)
	}
	other.wantCode("DELETE", path("/me/collection-links/%d", l.ID), nil, 404, "not_found")
	admin.ok("DELETE", path("/me/collection-links/%d", l.ID), nil, 200)
	anon.wantCode("GET", path("/public/collections/%s", token), nil, 404, "not_found")
	var links []app.CollectionLink
	admin.ok("GET", v1+"/me/collection-links", nil, 200).decode(t, &links)
	if len(links) != 0 {
		t.Fatalf("links: %+v", links)
	}
}
