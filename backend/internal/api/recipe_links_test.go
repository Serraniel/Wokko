package api_test

import (
	"math"
	"strings"
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

// TestCookingALinkedRecipeForShopping: the fricassee needs 450 ml of a stock that makes 1.5 l,
// so cooking it puts 0.3 of the stock's ingredients on the list.
func TestCookingALinkedRecipeForShopping(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	stockIn := simpleRecipe("Stock")
	stockIn.Servings = 4
	stockIn.Steps = []app.StepInput{{Instruction: "Simmer.", Ingredients: []app.IngredientInput{{FoodName: "Boiling hen", Amount: ptr(1.0)}, {FoodName: "water", UnitName: "l", Amount: ptr(2.0)}}}}
	stockIn.Yield = &app.RecipeYieldInput{Amount: 1.5, UnitName: "l"}
	stock := admin.createRecipe(stockIn)
	if stock.Yield == nil || stock.Yield.Amount != 1.5 || stock.Yield.Unit == nil {
		t.Fatalf("yield: %+v", stock.Yield)
	}
	dishIn := simpleRecipe("Fricassee")
	dishIn.Servings = 2
	dishIn.Steps = []app.StepInput{{Instruction: "Stir.", Ingredients: []app.IngredientInput{{FoodName: "Wokko stock", UnitName: "ml", Amount: ptr(450.0)}, {FoodName: "Rice", UnitName: "g", Amount: ptr(200.0)}}}}
	dish := admin.createRecipe(dishIn)
	var broth app.Food
	admin.ok("GET", path("/foods/%d", dish.Steps[0].Ingredients[0].Food.ID), nil, 200).decode(t, &broth)
	linkFood(admin, broth, stock.ID, 200)
	stockIng := dish.Steps[0].Ingredients[0].ID

	shopping := func() map[string]app.ShoppingItem {
		var list []app.ShoppingItem
		admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &list)
		out := map[string]app.ShoppingItem{}
		for _, it := range list {
			out[it.Food.Name] = it
		}
		return out
	}
	clear := func() {
		for _, it := range shopping() {
			admin.ok("DELETE", path("/shopping/%d", it.ID), nil, 200)
		}
	}

	// Bought, as before.
	admin.ok("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: dish.ID}, 201)
	if items := shopping(); items["Wokko stock"].Amount == nil || *items["Wokko stock"].Amount != 450 || len(items) != 2 {
		t.Fatalf("bought: %+v", items)
	}
	clear()

	// Cooked by its yield: 450 ml of 1.5 l.
	admin.ok("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: dish.ID, Cook: []app.CookChoice{{IngredientID: stockIng}}}, 201)
	items := shopping()
	if _, bought := items["Wokko stock"]; bought {
		t.Fatalf("the stock itself is still on the list: %+v", items)
	}
	hen, water := items["Boiling hen"], items["water"]
	if hen.Amount == nil || math.Abs(*hen.Amount-0.3) > 1e-9 || water.Amount == nil || math.Abs(*water.Amount-0.6) > 1e-9 || !strings.Contains(hen.Note, "Stock") || items["rice"].Amount == nil {
		t.Fatalf("cooked: %+v", items)
	}
	clear()

	// Twice the servings needs twice as much.
	admin.ok("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: dish.ID, Servings: 4, Cook: []app.CookChoice{{IngredientID: stockIng}}}, 201)
	if a := shopping()["Boiling hen"].Amount; a == nil || math.Abs(*a-0.6) > 1e-9 {
		t.Fatalf("scaled dish: %v", a)
	}
	clear()

	// Without a yield, the servings asked for decide: 2 of 4.
	stockIn.Yield = &app.RecipeYieldInput{}
	stockIn.BaseVersion = stock.Version
	admin.ok("PUT", path("/recipes/%d", stock.ID), stockIn, 200)
	admin.ok("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: dish.ID, Cook: []app.CookChoice{{IngredientID: stockIng, Servings: 2}}}, 201)
	if a := shopping()["Boiling hen"].Amount; a == nil || math.Abs(*a-0.5) > 1e-9 {
		t.Fatalf("by servings: %v", a)
	}
}

// TestYieldSurvivesAppsThatDontSendIt: kitshn and assistants save whole recipes without a yield.
func TestYieldSurvivesAppsThatDontSendIt(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	in := simpleRecipe("Stock")
	in.Yield = &app.RecipeYieldInput{Amount: 2, UnitName: "l"}
	r := admin.createRecipe(in)
	in.Yield = nil
	in.BaseVersion = r.Version
	var got app.Recipe
	admin.ok("PUT", path("/recipes/%d", r.ID), in, 200).decode(t, &got)
	if got.Yield == nil || got.Yield.Amount != 2 {
		t.Fatalf("yield lost: %+v", got.Yield)
	}
}
