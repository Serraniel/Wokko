package api_test

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
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
	// kitshn shows it as Tandoor's food.recipe.
	var kr struct {
		Steps []struct {
			Ingredients []struct {
				Food struct {
					Recipe *struct {
						ID   int64  `json:"id"`
						Name string `json:"name"`
					} `json:"recipe"`
				} `json:"food"`
			} `json:"ingredients"`
		} `json:"steps"`
	}
	cookie := kitshnLogin(t, e, "admin", "password123")
	if err := json.Unmarshal(kitshnGet(t, e, cookie, fmt.Sprintf("/api/recipe/%d/", dish.ID), 200), &kr); err != nil {
		t.Fatal(err)
	}
	if fr := kr.Steps[1].Ingredients[0].Food.Recipe; fr == nil || fr.ID != stock.ID || fr.Name != "Hühnerbrühe" {
		t.Fatalf("kitshn food.recipe: %+v", fr)
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

// TestLinkedRecipesCountForNutritionAndKnowledge: what's in the stock counts in the dish, for
// allergens and diet always, and for nutrition when the stock's yield says how much is used.
func TestLinkedRecipesCountForNutritionAndKnowledge(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	stockIn := simpleRecipe("Veg stock")
	stockIn.Steps = []app.StepInput{{Instruction: "Simmer.", Ingredients: []app.IngredientInput{{FoodName: "celery", UnitName: "g", Amount: ptr(100.0)}, {FoodName: "rice", UnitName: "g", Amount: ptr(300.0)}}}}
	stockIn.Yield = &app.RecipeYieldInput{Amount: 1.5, UnitName: "l"}
	stock := admin.createRecipe(stockIn)
	dishIn := simpleRecipe("Risotto")
	dishIn.Steps = []app.StepInput{{Instruction: "Stir.", Ingredients: []app.IngredientInput{{FoodName: "Wokko veg stock", UnitName: "ml", Amount: ptr(450.0)}}}}
	dish := admin.createRecipe(dishIn)
	var broth app.Food
	admin.ok("GET", path("/foods/%d", dish.Steps[0].Ingredients[0].Food.ID), nil, 200).decode(t, &broth)

	var before app.RecipeNutrition
	admin.ok("GET", path("/recipes/%d/nutrition", dish.ID), nil, 200).decode(t, &before)
	if before.Counted != 0 || len(before.Missing) != 1 {
		t.Fatalf("before linking: %+v", before)
	}
	linkFood(admin, broth, stock.ID, 200)

	var stockN, dishN app.RecipeNutrition
	admin.ok("GET", path("/recipes/%d/nutrition", stock.ID), nil, 200).decode(t, &stockN)
	admin.ok("GET", path("/recipes/%d/nutrition", dish.ID), nil, 200).decode(t, &dishN)
	if dishN.Counted != 1 || len(dishN.Missing) != 0 {
		t.Fatalf("after linking: %+v", dishN)
	}
	if want := stockN.Nutrients[0].Total * 0.3; math.Abs(dishN.Nutrients[0].Total-want) > 1e-6 || want <= 0 {
		t.Fatalf("energy %v, want %v (0.3 of the stock)", dishN.Nutrients[0].Total, want)
	}

	// Without a yield, the amount can't be compared, so it stays missing rather than guessed.
	stockIn.Yield = &app.RecipeYieldInput{}
	stockIn.BaseVersion = stock.Version
	admin.ok("PUT", path("/recipes/%d", stock.ID), stockIn, 200)
	admin.ok("GET", path("/recipes/%d/nutrition", dish.ID), nil, 200).decode(t, &dishN)
	if dishN.Counted != 0 {
		t.Fatalf("guessed without a yield: %+v", dishN)
	}

	// The stock's celery is an allergen of the risotto.
	var res struct {
		Allergens []struct {
			ID    string   `json:"id"`
			Foods []string `json:"foods"`
		} `json:"allergens"`
	}
	admin.ok("GET", path("/recipes/%d/insights", dish.ID), nil, 200).decode(t, &res)
	found := false
	for _, a := range res.Allergens {
		if a.ID == "celery" && slices.Contains(a.Foods, "Wokko veg stock") {
			found = true
		}
	}
	if !found {
		t.Fatalf("celery from the stock: %+v", res.Allergens)
	}
}

// TestImportKeepsRecipeLinksAndYield: a Wokko export and a Tandoor file bring their links
// between foods and recipes, and Wokko's how much a recipe makes.
func TestImportKeepsRecipeLinksAndYield(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	wokko := `{"format":"wokko-export-1","recipes":[
	  {"name":"Imported stock","servings":4,"yield":{"amount":1.5,"unit":{"name":"l"}},"steps":[{"instruction":"Simmer.","ingredients":[{"food":{"name":"Imported hen"},"amount":1}]}]},
	  {"name":"Imported dish","servings":2,"steps":[{"instruction":"Stir.","ingredients":[{"food":{"name":"Imported stock food"},"unit":{"name":"ml"},"amount":450,"recipe":{"id":99,"name":"Imported stock"}}]}]}]}`
	var res app.FileImportResult
	if r := importUpload(t, admin, false, []byte(wokko)); r.status != 200 {
		t.Fatalf("import: %d %s", r.status, r.body)
	} else {
		r.decode(t, &res)
	}
	var stock, dish app.Recipe
	admin.ok("GET", path("/recipes/%d", res.Items[0].RecipeID), nil, 200).decode(t, &stock)
	admin.ok("GET", path("/recipes/%d", res.Items[1].RecipeID), nil, 200).decode(t, &dish)
	if stock.Yield == nil || stock.Yield.Amount != 1.5 {
		t.Fatalf("yield: %+v", stock.Yield)
	}
	if l := dish.Steps[0].Ingredients[0].Recipe; l == nil || l.ID != stock.ID {
		t.Fatalf("wokko link: %+v", l)
	}

	tandoor := `{"name":"Tandoor dish","working_time":10,"waiting_time":0,"servings":2,"steps":[{"instruction":"Mix.","ingredients":[{"food":{"name":"Tandoor sauce","recipe":{"id":5,"name":"Imported stock"}},"amount":1}]}]}`
	if r := importUpload(t, admin, false, []byte(tandoor)); r.status != 200 {
		t.Fatalf("tandoor import: %d %s", r.status, r.body)
	} else {
		r.decode(t, &res)
	}
	var td app.Recipe
	admin.ok("GET", path("/recipes/%d", res.Items[0].RecipeID), nil, 200).decode(t, &td)
	if l := td.Steps[0].Ingredients[0].Recipe; l == nil || l.ID != stock.ID {
		t.Fatalf("tandoor link to an existing recipe: %+v", l)
	}
}
