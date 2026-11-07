package api_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// household returns the admin and a second member of the admin's household, plus an outsider.
func household(t *testing.T, e *env) (admin, member, stranger *client) {
	t.Helper()
	admin = e.admin()
	stranger = e.newUser(admin, "stranger")
	var inv app.Invite
	admin.ok("POST", v1+"/household/invites", map[string]any{"max_uses": 1}, 201).decode(t, &inv)
	member = e.anon()
	member.ok("POST", v1+"/auth/register", map[string]string{"handle": "member", "password": "password123", "invite": inv.Token}, 201)
	return admin, member, stranger
}

func TestMealTypesStartWithDefaultsAndStayInTheHousehold(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	var types []app.MealType
	admin.ok("GET", v1+"/meal-types", nil, 200).decode(t, &types)
	if len(types) != 4 || types[0].Name != "Breakfast" {
		t.Fatalf("defaults: %+v", types)
	}
	var again []app.MealType
	member.ok("GET", v1+"/meal-types", nil, 200).decode(t, &again)
	if len(again) != 4 || again[0].ID != types[0].ID {
		t.Fatalf("household shares its types: %+v", again)
	}
	stranger.wantCode("PUT", path("/meal-types/%d", types[0].ID), app.MealTypeInput{Name: "Mine"}, 404, "not_found")
	stranger.wantCode("DELETE", path("/meal-types/%d", types[0].ID), nil, 404, "not_found")

	var made app.MealType
	admin.ok("POST", v1+"/meal-types", app.MealTypeInput{Name: "Brunch", Position: 9, Time: "10:00"}, 201).decode(t, &made)
	admin.ok("PUT", path("/meal-types/%d", made.ID), app.MealTypeInput{Name: "Late brunch", Position: 9}, 200)
	admin.wantCode("POST", v1+"/meal-types", app.MealTypeInput{Name: " "}, 400, "name_required")

	plan := app.MealPlanInput{Title: "Leftovers", FromDate: "2026-11-09", MealTypeID: made.ID}
	admin.ok("POST", v1+"/meal-plans", plan, 201)
	admin.wantCode("DELETE", path("/meal-types/%d", made.ID), nil, 409, "in_use")
}

func TestMealPlans(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	var types []app.MealType
	admin.ok("GET", v1+"/meal-types", nil, 200).decode(t, &types)
	dinner := types[2].ID

	public := simpleRecipe("Curry")
	public.Visibility = "household"
	curry := admin.createRecipe(public)
	hidden := simpleRecipe("Surprise")
	hidden.Visibility = "private"
	surprise := admin.createRecipe(hidden)

	var p app.MealPlan
	admin.ok("POST", v1+"/meal-plans", app.MealPlanInput{RecipeID: curry.ID, Servings: 3, FromDate: "2026-11-09T00:00:00", MealTypeID: dinner, Note: " Spicy "}, 201).decode(t, &p)
	if p.FromDate != "2026-11-09" || p.ToDate != "2026-11-09" || p.Recipe == nil || p.Recipe.Name != "Curry" || p.Note != "Spicy" || p.MealType.ID != dinner {
		t.Fatalf("plan: %+v", p)
	}
	var secret app.MealPlan
	admin.ok("POST", v1+"/meal-plans", app.MealPlanInput{RecipeID: surprise.ID, FromDate: "2026-11-10", ToDate: "2026-11-11", MealTypeID: dinner}, 201).decode(t, &secret)

	var seen []app.MealPlan
	member.ok("GET", v1+"/meal-plans?from=2026-11-01&to=2026-11-30", nil, 200).decode(t, &seen)
	if len(seen) != 2 || seen[1].Recipe != nil || !seen[1].RecipeHidden || seen[1].Title != "" {
		t.Fatalf("a private recipe's name must not reach the household: %+v", seen)
	}
	member.ok("GET", v1+"/meal-plans?from=2026-11-11&to=2026-11-12", nil, 200).decode(t, &seen)
	if len(seen) != 1 || seen[0].ID != secret.ID {
		t.Fatalf("a range overlapping a multi-day plan: %+v", seen)
	}
	member.ok("GET", v1+"/meal-plans?from=2026-12-01", nil, 200).decode(t, &seen)
	if len(seen) != 0 {
		t.Fatalf("later range: %+v", seen)
	}
	// A member can edit the plan without losing a recipe they can't open.
	var edited app.MealPlan
	member.ok("PUT", path("/meal-plans/%d", secret.ID), app.MealPlanInput{RecipeID: surprise.ID, Title: "Birthday", FromDate: "2026-11-12", MealTypeID: dinner}, 200).decode(t, &edited)
	admin.ok("GET", path("/meal-plans/%d", secret.ID), nil, 200).decode(t, &edited)
	if edited.Recipe == nil || edited.Title != "Birthday" {
		t.Fatalf("edit: %+v", edited)
	}
	// Nobody can plan a recipe they can't see.
	member.wantCode("POST", v1+"/meal-plans", app.MealPlanInput{RecipeID: surprise.ID, FromDate: "2026-11-09", MealTypeID: dinner}, 404, "not_found")
	stranger.wantCode("GET", path("/meal-plans/%d", p.ID), nil, 404, "not_found")
	stranger.wantCode("DELETE", path("/meal-plans/%d", p.ID), nil, 404, "not_found")
	stranger.wantCode("POST", v1+"/meal-plans", app.MealPlanInput{Title: "x", FromDate: "2026-11-09", MealTypeID: dinner}, 404, "not_found")
	admin.wantCode("POST", v1+"/meal-plans", app.MealPlanInput{FromDate: "2026-11-09", MealTypeID: dinner}, 400, "plan_empty")
	admin.wantCode("POST", v1+"/meal-plans", app.MealPlanInput{Title: "x", FromDate: "soon", MealTypeID: dinner}, 400, "bad_date")
	admin.wantCode("POST", v1+"/meal-plans", app.MealPlanInput{Title: "x", FromDate: "2026-11-09", ToDate: "2026-11-01", MealTypeID: dinner}, 400, "bad_date")

	// A purged recipe leaves the plan standing, with the name it had.
	admin.ok("DELETE", path("/recipes/%d", curry.ID), nil, 200)
	admin.ok("DELETE", path("/trash/%d", curry.ID), nil, 200)
	var after app.MealPlan
	admin.ok("GET", path("/meal-plans/%d", p.ID), nil, 200).decode(t, &after)
	if after.Recipe != nil || after.Title != "Curry" {
		t.Fatalf("after purge: %+v", after)
	}
	admin.ok("DELETE", path("/meal-plans/%d", p.ID), nil, 200)
	admin.wantCode("GET", path("/meal-plans/%d", p.ID), nil, 404, "not_found")
}

func TestShoppingList(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	in := simpleRecipe("Soup")
	in.Servings = 2
	in.Visibility = "household"
	two := 200.0
	in.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "onion", UnitName: "g", Amount: &two, Note: "red"}, {FoodName: "salt"}}}}
	soup := admin.createRecipe(in)

	var group app.ShoppingRecipe
	member.ok("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: soup.ID, Servings: 5}, 201).decode(t, &group)
	var items []app.ShoppingItem
	admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if len(items) != 2 || *items[0].Amount != 500 || items[0].Unit == nil || items[0].Note != "red" || items[1].Amount != nil || items[0].Recipe == nil || items[0].Recipe.Name != "Soup" {
		t.Fatalf("scaled items: %+v", items)
	}
	var loose app.ShoppingItem
	three := 3.0
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{FoodName: "Kitchen towels", Amount: &three}, 201).decode(t, &loose)
	if loose.Food.Name != "Kitchen towels" || loose.Checked {
		t.Fatalf("item: %+v", loose)
	}
	admin.wantCode("POST", v1+"/shopping", app.ShoppingItemInput{}, 400, "ingredient_required")
	var typed app.ShoppingItem
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "2 kg Zwiebeln, rot"}, 201).decode(t, &typed)
	if *typed.Amount != 2 || typed.Unit == nil || typed.Unit.Abbreviation != "kg" || typed.Food.Name != "onion" || typed.Note != "rot" {
		t.Fatalf("typed line: %+v", typed)
	}
	admin.ok("DELETE", path("/shopping/%d", typed.ID), nil, 200)
	var empty []app.ShoppingItem
	stranger.ok("GET", v1+"/shopping", nil, 200).decode(t, &empty)
	if len(empty) != 0 {
		t.Fatalf("stranger: %+v", empty)
	}
	stranger.wantCode("DELETE", path("/shopping/%d", loose.ID), nil, 404, "not_found")
	stranger.wantCode("PUT", path("/shopping/%d", loose.ID), app.ShoppingItemInput{}, 404, "not_found")
	stranger.wantCode("POST", v1+"/shopping/recipes", app.ShoppingAddRecipe{RecipeID: soup.ID}, 404, "not_found")
	stranger.ok("POST", v1+"/shopping/check", app.ShoppingCheckInput{IDs: []int64{loose.ID}, Checked: true}, 200)
	admin.ok("GET", path("/shopping"), nil, 200).decode(t, &items)
	if items[2].Checked {
		t.Fatal("someone else's check reached this list")
	}

	member.ok("POST", v1+"/shopping/check", app.ShoppingCheckInput{IDs: []int64{items[0].ID, items[2].ID}, Checked: true}, 200)
	admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if !items[0].Checked || items[1].Checked || !items[2].Checked {
		t.Fatalf("checked: %+v", items)
	}
	member.ok("POST", v1+"/shopping/check", app.ShoppingCheckInput{IDs: []int64{items[2].ID}, Checked: false}, 200)

	var changed app.ShoppingItem
	four := 4.0
	admin.ok("PUT", path("/shopping/%d", loose.ID), app.ShoppingItemInput{Amount: &four, UnitName: "pcs", Note: "big"}, 200).decode(t, &changed)
	if *changed.Amount != 4 || changed.Food.Name != "Kitchen towels" || changed.Note != "big" || changed.Unit == nil {
		t.Fatalf("update: %+v", changed)
	}
	admin.ok("DELETE", v1+"/shopping/checked", nil, 200)
	admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if len(items) != 2 {
		t.Fatalf("after clearing: %+v", items)
	}
	admin.ok("DELETE", path("/shopping/%d", items[0].ID), nil, 200)

	// A food on the list can't vanish, and merging moves it along.
	admin.wantCode("DELETE", path("/foods/%d", changed.Food.ID), nil, 409, "in_use")
	var onion app.Food
	var foods app.Page[app.Food]
	admin.ok("GET", v1+"/foods?q=onion", nil, 200).decode(t, &foods)
	onion = foods.Items[0]
	admin.ok("POST", v1+"/foods/merge", app.MergeInput{SourceID: changed.Food.ID, TargetID: onion.ID}, 200)
	admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if items[len(items)-1].Food.ID != onion.ID {
		t.Fatalf("after merge: %+v", items)
	}
}

func TestPlanningAddsToShopping(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	in := simpleRecipe("Rice")
	in.Servings = 1
	one := 100.0
	in.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{{FoodName: "rice", UnitName: "g", Amount: &one}}}}
	r := c.createRecipe(in)
	var types []app.MealType
	c.ok("GET", v1+"/meal-types", nil, 200).decode(t, &types)
	var p app.MealPlan
	c.ok("POST", v1+"/meal-plans", app.MealPlanInput{RecipeID: r.ID, Servings: 3, FromDate: "2026-11-09", MealTypeID: types[0].ID, AddToShopping: true}, 201).decode(t, &p)
	var items []app.ShoppingItem
	c.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if len(items) != 1 || *items[0].Amount != 300 || items[0].Recipe.MealPlanID != p.ID {
		t.Fatalf("items: %+v", items)
	}
}

func TestSupermarketsSortCategories(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	var dairy, veg app.SupermarketCategory
	admin.ok("POST", v1+"/categories", app.CategoryInput{Name: "Dairy"}, 201).decode(t, &dairy)
	member.ok("POST", v1+"/categories", app.CategoryInput{Name: "Vegetables"}, 201).decode(t, &veg)
	var market app.Supermarket
	admin.ok("POST", v1+"/supermarkets", app.CategoryInput{Name: "Corner shop"}, 201).decode(t, &market)
	admin.ok("PUT", path("/supermarkets/%d/order", market.ID), map[string]any{"category_ids": []int64{veg.ID, dairy.ID}}, 200).decode(t, &market)
	if len(market.Categories) != 2 || market.Categories[0].Category.ID != veg.ID {
		t.Fatalf("order: %+v", market)
	}
	var list []app.Supermarket
	member.ok("GET", v1+"/supermarkets", nil, 200).decode(t, &list)
	if len(list) != 1 || list[0].Categories[1].Category.Name != "Dairy" {
		t.Fatalf("member: %+v", list)
	}
	stranger.wantCode("PUT", path("/supermarkets/%d/order", market.ID), map[string]any{"category_ids": []int64{}}, 404, "not_found")
	stranger.wantCode("DELETE", path("/categories/%d", dairy.ID), nil, 404, "not_found")

	var foods app.Page[app.Food]
	admin.ok("GET", v1+"/foods?q=onion", nil, 200).decode(t, &foods)
	stranger.wantCode("PUT", path("/foods/%d/category", foods.Items[0].ID), map[string]int64{"category_id": veg.ID}, 404, "not_found")
	admin.ok("PUT", path("/foods/%d/category", foods.Items[0].ID), map[string]int64{"category_id": veg.ID}, 200)
	var item app.ShoppingItem
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{FoodID: foods.Items[0].ID}, 201).decode(t, &item)
	if item.Category == nil || item.Category.Name != "Vegetables" {
		t.Fatalf("category: %+v", item)
	}
	admin.ok("PUT", path("/foods/%d/category", foods.Items[0].ID), map[string]int64{"category_id": 0}, 200)
	admin.ok("DELETE", path("/supermarkets/%d", market.ID), nil, 200)
	admin.ok("DELETE", path("/categories/%d", dairy.ID), nil, 200)
}

func TestNutritionAddsUpAndAdmitsWhatItDoesNotKnow(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	var nutrients []app.Nutrient
	c.ok("GET", v1+"/nutrients", nil, 200).decode(t, &nutrients)
	if len(nutrients) != 8 || nutrients[0].Name != "Energy" || nutrients[0].Unit != "kcal" {
		t.Fatalf("nutrients: %+v", nutrients)
	}
	amt := func(v float64) *float64 { return &v }
	in := simpleRecipe("Toast")
	in.Servings = 2
	in.Steps = []app.StepInput{{Ingredients: []app.IngredientInput{
		{FoodName: "Testbread", UnitName: "g", Amount: amt(200)},
		{FoodName: "Testegg", Amount: amt(2)},
		{FoodName: "Mystery", UnitName: "g", Amount: amt(50)},
		{FoodName: "Testbread", UnitName: "handful", Amount: amt(1)},
	}}}
	r := c.createRecipe(in)
	bread, egg := r.Steps[0].Ingredients[0].Food.ID, r.Steps[0].Ingredients[1].Food.ID
	c.ok("PUT", path("/foods/%d/nutrition", bread), app.FoodNutrition{Values: map[int64]float64{1: 250, 7: 10}}, 200)
	piece := 60.0
	var saved app.FoodNutrition
	c.ok("PUT", path("/foods/%d/nutrition", egg), app.FoodNutrition{PieceGrams: &piece, Values: map[int64]float64{1: 150}}, 200).decode(t, &saved)
	if *saved.PieceGrams != 60 || saved.Values[1] != 150 {
		t.Fatalf("saved: %+v", saved)
	}
	c.wantCode("PUT", path("/foods/%d/nutrition", bread), app.FoodNutrition{Values: map[int64]float64{1: -1}}, 400, "bad_amount")

	var got app.RecipeNutrition
	c.ok("GET", path("/recipes/%d/nutrition", r.ID), nil, 200).decode(t, &got)
	if got.Total != 4 || got.Counted != 2 || len(got.Missing) != 2 {
		t.Fatalf("counts: %+v", got)
	}
	// 200 g at 250 kcal/100 g, plus 2 pieces of 60 g at 150 kcal/100 g.
	if e := got.Nutrients[0]; math.Abs(e.Total-680) > 0.001 || math.Abs(e.PerServing-340) > 0.001 {
		t.Fatalf("energy: %+v", e)
	}
	// A user can't change seeded catalog nutrition.
	other := e.newUser(c, "other")
	var foods app.Page[app.Food]
	other.ok("GET", v1+"/foods?q=onion", nil, 200).decode(t, &foods)
	other.wantCode("PUT", path("/foods/%d/nutrition", foods.Items[0].ID), app.FoodNutrition{}, 403, "forbidden")
	res := other.do("GET", path("/recipes/%d/nutrition", r.ID), nil)
	if res.status != 200 && res.status != 404 {
		t.Fatalf("nutrition follows recipe access: %d", res.status)
	}
	var raw map[string]any
	_ = json.Unmarshal(res.body, &raw)
}

// On bad signal a reply can get lost after the server saved the item, so the app sends it again.
func TestAddingTheSameShoppingItemAgainChangesNothing(t *testing.T) {
	e := newEnv(t)
	admin, member, stranger := household(t, e)
	var first, again, theirs app.ShoppingItem
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "3 Tomaten", ClientRef: "ref-1", Checked: true}, 201).decode(t, &first)
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "3 Tomaten", ClientRef: "ref-1", Checked: true}, 201).decode(t, &again)
	if first.ID != again.ID || !again.Checked {
		t.Fatalf("repeat made another item: %+v %+v", first, again)
	}
	// The same household shares references, while another one has its own.
	member.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "3 Tomaten", ClientRef: "ref-1"}, 201).decode(t, &again)
	if again.ID != first.ID {
		t.Fatalf("household member: %+v", again)
	}
	stranger.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "Salz", ClientRef: "ref-1"}, 201).decode(t, &theirs)
	if theirs.ID == first.ID {
		t.Fatal("a reference must not reach another household's item")
	}
	// Items without a reference stay separate, as when someone really wants two lines.
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "Salz"}, 201)
	admin.ok("POST", v1+"/shopping", app.ShoppingItemInput{Line: "Salz"}, 201)
	var items []app.ShoppingItem
	admin.ok("GET", v1+"/shopping", nil, 200).decode(t, &items)
	if len(items) != 3 {
		t.Fatalf("items: %+v", items)
	}
	admin.wantCode("POST", v1+"/shopping", app.ShoppingItemInput{Line: "x", ClientRef: strings.Repeat("r", 101)}, 400, "bad_request")
}
