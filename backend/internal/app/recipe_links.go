package app

import (
	"context"
	"database/sql"
	"strings"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// maxRecipeDepth is how deep recipes may use recipes: a stock in a sauce in a dish.
const maxRecipeDepth = 3

// recipeLinks finds which of these foods can be cooked from a recipe this viewer may see. A
// recipe the viewer can't see isn't linked, and nothing hints that it exists.
func (s *Service) recipeLinks(ctx context.Context, q *sqlcgen.Queries, v Viewer, foodIDs []int64) (map[int64]*RecipeLink, error) {
	out := map[int64]*RecipeLink{}
	if len(foodIDs) == 0 {
		return out, nil
	}
	rows, err := q.ListFoodRecipeLinks(ctx, foodIDs)
	if err != nil {
		return nil, err
	}
	seen := map[int64]*RecipeLink{}
	for _, row := range rows {
		id := row.RecipeID.Int64
		link, done := seen[id]
		if !done {
			r, err := q.GetRecipe(ctx, id)
			if isNoRows(err) {
				seen[id] = nil
				continue
			}
			if err != nil {
				return nil, err
			}
			ok, err := s.canView(ctx, q, v, r)
			if err != nil {
				return nil, err
			}
			if ok && !r.DeletedAt.Valid {
				link = &RecipeLink{ID: r.ID, Name: r.Name, Servings: r.Servings, HasYield: r.YieldAmount.Valid && r.YieldAmount.Float64 > 0}
			}
			seen[id] = link
		}
		if link != nil {
			out[row.ID] = link
		}
	}
	return out, nil
}

// setFoodRecipe links a food to a recipe the person may see, or removes the link with 0. A
// recipe that already uses the food, directly or through other linked recipes, is refused,
// since cooking it would need itself.
func (s *Service) setFoodRecipe(ctx context.Context, q *sqlcgen.Queries, v Viewer, food, recipe int64) error {
	if recipe == 0 {
		return q.SetFoodRecipe(ctx, sqlcgen.SetFoodRecipeParams{RecipeID: nullID(0), ID: food})
	}
	r, err := q.GetRecipe(ctx, recipe)
	if isNoRows(err) {
		return invalid("unknown_recipe", "This recipe doesn't exist")
	}
	if err != nil {
		return err
	}
	ok, err := s.canView(ctx, q, v, r)
	if err != nil {
		return err
	}
	if !ok || r.DeletedAt.Valid {
		return invalid("unknown_recipe", "This recipe doesn't exist")
	}
	uses, err := s.recipeUses(ctx, q, recipe, food, maxRecipeDepth)
	if err != nil {
		return err
	}
	if uses {
		return invalid("recipe_cycle", "This recipe already uses this ingredient, so it can't be cooked from it")
	}
	return q.SetFoodRecipe(ctx, sqlcgen.SetFoodRecipeParams{RecipeID: nullID(recipe), ID: food})
}

// recipeUses tells whether a recipe needs a food, itself or through the recipes its foods link to.
func (s *Service) recipeUses(ctx context.Context, q *sqlcgen.Queries, recipe, food int64, depth int) (bool, error) {
	if depth == 0 {
		return false, nil
	}
	ings, err := q.ListIngredientsForRecipe(ctx, recipe)
	if err != nil {
		return false, err
	}
	var foods []int64
	for _, i := range ings {
		if i.FoodID == food {
			return true, nil
		}
		foods = append(foods, i.FoodID)
	}
	links, err := q.ListFoodRecipeLinks(ctx, foods)
	if err != nil {
		return false, err
	}
	for _, l := range links {
		if uses, err := s.recipeUses(ctx, q, l.RecipeID.Int64, food, depth-1); err != nil || uses {
			return uses, err
		}
	}
	return false, nil
}

// LinkFoodRecipe sets or removes (with 0) the recipe a food can be cooked from, under the same
// rules as editing the food.
func (s *Service) LinkFoodRecipe(ctx context.Context, v Viewer, food, recipe int64) (Food, error) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.GetFood(ctx, food)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !s.canEditCatalog(v, row.CreatedBy.Int64) {
			return ErrForbidden
		}
		return s.setFoodRecipe(ctx, q, v, food, recipe)
	})
	if err != nil {
		return Food{}, err
	}
	return s.GetFood(ctx, v, food)
}

// writeYield stores how much a recipe makes. nil keeps what's there.
func (s *Service) writeYield(ctx context.Context, q *sqlcgen.Queries, v Viewer, recipe int64, in *RecipeYieldInput) error {
	if in == nil {
		return nil
	}
	if in.Amount <= 0 {
		return q.SetRecipeYield(ctx, sqlcgen.SetRecipeYieldParams{ID: recipe})
	}
	unit, err := s.resolveUnit(ctx, q, v, IngredientInput{UnitID: in.UnitID, UnitName: strings.TrimSpace(in.UnitName)})
	if err != nil {
		return err
	}
	return q.SetRecipeYield(ctx, sqlcgen.SetRecipeYieldParams{YieldAmount: sql.NullFloat64{Float64: in.Amount, Valid: true}, YieldUnitID: unit, ID: recipe})
}

// recipeYield reads how much a recipe makes.
func recipeYield(ctx context.Context, q *sqlcgen.Queries, r sqlcgen.Recipe, locale string) (*RecipeYield, error) {
	if !r.YieldAmount.Valid || r.YieldAmount.Float64 <= 0 {
		return nil, nil
	}
	y := &RecipeYield{Amount: r.YieldAmount.Float64}
	if r.YieldUnitID.Valid {
		units, err := unitRefs(ctx, q, []int64{r.YieldUnitID.Int64}, locale)
		if err != nil {
			return nil, err
		}
		if u, ok := units[r.YieldUnitID.Int64]; ok {
			y.Unit = &u
		}
	}
	return y, nil
}

// cookScale is how much of a linked recipe a dish needs. Its yield decides when the amount
// can be compared with it, directly or by weight (450 ml of a stock that makes 1.5 l is 0.3);
// otherwise the servings asked for; otherwise the whole recipe.
func (s *Service) cookScale(ctx context.Context, q *sqlcgen.Queries, sub sqlcgen.Recipe, unit sql.NullInt64, amount sql.NullFloat64, servings float64) (float64, error) {
	if scale, ok, err := yieldScale(ctx, q, sub, unit, amount); err != nil || ok {
		return scale, err
	}
	if servings > 0 && sub.Servings > 0 {
		return servings / float64(sub.Servings), nil
	}
	return 1, nil
}

// yieldScale compares an amount with what a recipe makes, directly or by weight. ok is false
// when they can't be compared.
func yieldScale(ctx context.Context, q *sqlcgen.Queries, sub sqlcgen.Recipe, unit sql.NullInt64, amount sql.NullFloat64) (float64, bool, error) {
	if !sub.YieldAmount.Valid || sub.YieldAmount.Float64 <= 0 || !amount.Valid || amount.Float64 <= 0 || !unit.Valid || !sub.YieldUnitID.Valid {
		return 0, false, nil
	}
	if unit.Int64 == sub.YieldUnitID.Int64 {
		return amount.Float64 / sub.YieldAmount.Float64, true, nil
	}
	rows, err := q.ListUnitGrams(ctx, []int64{unit.Int64, sub.YieldUnitID.Int64})
	if err != nil {
		return 0, false, err
	}
	grams := map[int64]float64{}
	for _, r := range rows {
		if r.Grams.Valid {
			grams[r.ID] = r.Grams.Float64
		}
	}
	if g, y := grams[unit.Int64], grams[sub.YieldUnitID.Int64]; g > 0 && y > 0 {
		return amount.Float64 * g / (sub.YieldAmount.Float64 * y), true, nil
	}
	return 0, false, nil
}
