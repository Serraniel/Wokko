package api_test

import (
	"net/url"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

func searchFoods(t *testing.T, c *client, q string) []app.Food {
	t.Helper()
	var page app.Page[app.Food]
	c.ok("GET", v1+"/foods?q="+url.QueryEscape(q), nil, 200).decode(t, &page)
	return page.Items
}

// TestFoodSearchFindsKnownFoodsInsideLongerNames: a search for "geriebener Cheddar" offers
// Cheddar instead of nothing, so nobody adds a second Cheddar.
func TestFoodSearchFindsKnownFoodsInsideLongerNames(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()

	got := searchFoods(t, admin, "geriebener Cheddar")
	if len(got) == 0 || got[0].Name != "cheddar" || got[0].Match != "within" || got[0].MatchedName == "" {
		t.Fatalf("geriebener Cheddar: %+v", got)
	}
	if got := searchFoods(t, admin, "Cheddar"); len(got) == 0 || got[0].Match != "exact" {
		t.Fatalf("a direct hit stays a direct hit: %+v", got)
	}
	if got := searchFoods(t, admin, "chedd"); len(got) == 0 || got[0].Match != "contains" {
		t.Fatalf("part of a name: %+v", got)
	}

	// The longest known name wins.
	r := simpleRecipe("Toast")
	r.Steps[1].Ingredients = []app.IngredientInput{{FoodName: "Schinken"}, {FoodName: "Kochschinken"}}
	admin.createRecipe(r)
	if got := searchFoods(t, admin, "Kochschinken in Scheiben"); len(got) == 0 || got[0].Name != "Kochschinken" {
		t.Fatalf("longest match first: %+v", got)
	}

	// German joins words, so a compound ending in a known German name finds it.
	if got := searchFoods(t, admin, "Gemüsezwiebel"); len(got) == 0 || got[0].Name != "onion" || got[0].MatchedName != "Zwiebel" {
		t.Fatalf("compound: %+v", got)
	}
	// English doesn't: a unicorn isn't corn.
	for _, f := range searchFoods(t, admin, "unicorn dust") {
		if f.MatchedName == "corn" {
			t.Fatalf("unicorn matched corn: %+v", f)
		}
	}
	if got := searchFoods(t, admin, "Dingsbums"); len(got) != 0 {
		t.Fatalf("unknown: %+v", got)
	}
}
