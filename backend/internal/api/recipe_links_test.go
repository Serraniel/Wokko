package api_test

import (
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// linkFood links a food to a recipe through the ingredient editor's API.
func linkFood(c *client, food app.Food, recipe int64, want int) result {
	c.e.t.Helper()
	return c.ok("PUT", path("/foods/%d", food.ID), app.FoodInput{Translations: food.Translations, Aliases: food.Aliases, RecipeID: &recipe}, want)
}

// TestRecipesCanBeIngredients: a fricassee uses chicken stock, which has its own recipe. The
// ingredient links to it for people who may see the stock recipe, and for nobody else.
func TestRecipesCanBeIngredients(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	stockIn := simpleRecipe("Hühnerbrühe")
	stockIn.Steps[1].Ingredients = []app.IngredientInput{{FoodName: "Suppenhuhn"}}
	stock := admin.createRecipe(stockIn)
	dishIn := simpleRecipe("Hühnerfrikassee")
	dishIn.Steps[1].Ingredients = []app.IngredientInput{{FoodName: "Hühnerbrühe", UnitName: "ml", Amount: ptr(450.0)}}
	dish := admin.createRecipe(dishIn)

	var broth app.Food
	admin.ok("GET", path("/foods/%d", dish.Steps[1].Ingredients[0].Food.ID), nil, 200).decode(t, &broth)
	var linked app.Food
	linkFood(admin, broth, stock.ID, 200).decode(t, &linked)
	if linked.Recipe == nil || linked.Recipe.ID != stock.ID || linked.Recipe.Name != "Hühnerbrühe" {
		t.Fatalf("food: %+v", linked.Recipe)
	}
	var got app.Recipe
	admin.ok("GET", path("/recipes/%d", dish.ID), nil, 200).decode(t, &got)
	if l := got.Steps[1].Ingredients[0].Recipe; l == nil || l.ID != stock.ID {
		t.Fatalf("ingredient link: %+v", l)
	}
	// The stock recipe doesn't link to itself through its own food.
	admin.ok("GET", path("/recipes/%d", stock.ID), nil, 200).decode(t, &got)
	for _, st := range got.Steps {
		for _, ing := range st.Ingredients {
			if ing.Recipe != nil {
				t.Fatalf("stock links: %+v", ing)
			}
		}
	}

	// Someone the fricassee is shared with, but not the private stock recipe, sees a plain food.
	friend := e.newUser(admin, "friend")
	admin.ok("POST", path("/recipes/%d/sharing/users", dish.ID), map[string]string{"handle": "friend"}, 200)
	friend.ok("GET", path("/recipes/%d", dish.ID), nil, 200).decode(t, &got)
	if got.Steps[1].Ingredients[0].Recipe != nil {
		t.Fatalf("a recipe the friend can't see is linked: %+v", got.Steps[1].Ingredients[0].Recipe)
	}

	// Cooking chicken from the fricassee would need the fricassee itself.
	var hen app.Food
	admin.ok("GET", path("/foods/%d", stock.Steps[1].Ingredients[0].Food.ID), nil, 200).decode(t, &hen)
	admin.wantCode("PUT", path("/foods/%d", hen.ID), app.FoodInput{Translations: hen.Translations, RecipeID: ptr(dish.ID)}, 400, "recipe_cycle")
	admin.wantCode("PUT", path("/foods/%d", hen.ID), app.FoodInput{Translations: hen.Translations, RecipeID: ptr(int64(999999))}, 400, "unknown_recipe")
	theirs := friend.createRecipe(simpleRecipe("Friend's secret"))
	admin.wantCode("PUT", path("/foods/%d", hen.ID), app.FoodInput{Translations: hen.Translations, RecipeID: ptr(theirs.ID)}, 400, "unknown_recipe")

	// Leaving it out keeps the link, 0 removes it.
	admin.ok("PUT", path("/foods/%d", broth.ID), app.FoodInput{Translations: broth.Translations}, 200).decode(t, &linked)
	if linked.Recipe == nil {
		t.Fatal("an edit without recipe_id removed the link")
	}
	linkFood(admin, broth, 0, 200).decode(t, &linked)
	if linked.Recipe != nil {
		t.Fatalf("not removed: %+v", linked.Recipe)
	}
}

func TestMCPLinksAFoodToARecipe(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	enableMCP(t, admin)
	stock := admin.createRecipe(simpleRecipe("Stock"))
	dishIn := simpleRecipe("Dish")
	dishIn.Steps[1].Ingredients = []app.IngredientInput{{FoodName: "Wokko stock"}}
	dish := admin.createRecipe(dishIn)
	s := connect(t, e, newToken(t, admin, "write"))
	var f app.Food
	call(t, s, "link_food_recipe", map[string]any{"food_id": dish.Steps[1].Ingredients[0].Food.ID, "recipe_id": stock.ID}, &f)
	if f.Recipe == nil || f.Recipe.ID != stock.ID {
		t.Fatalf("linked: %+v", f.Recipe)
	}
	var got app.Recipe
	call(t, s, "get_recipe", map[string]any{"id": dish.ID}, &got)
	if got.Steps[1].Ingredients[0].Recipe == nil {
		t.Fatal("get_recipe doesn't show the link")
	}
}
