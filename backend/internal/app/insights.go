package app

import (
	"context"
	"strings"

	"github.com/serraniel/wokko/backend/internal/knowledge"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// RecipeInsights reads a recipe with Wokko's built-in kitchen knowledge: its sauces, techniques,
// allergens, diets and likely problems. It changes nothing.
func (s *Service) RecipeInsights(ctx context.Context, v Viewer, id int64) (knowledge.Result, error) {
	r, err := s.GetRecipe(ctx, v, id)
	if err != nil {
		return knowledge.Result{}, err
	}
	return s.insights(ctx, r, v.locale())
}

// SharedRecipeInsights is RecipeInsights for someone with a share link.
func (s *Service) SharedRecipeInsights(ctx context.Context, locale, token string) (knowledge.Result, error) {
	r, err := s.GetSharedRecipe(ctx, locale, token)
	if err != nil {
		return knowledge.Result{}, err
	}
	return s.insights(ctx, r, locale)
}

// PublicRecipeInsights is RecipeInsights for a public recipe.
func (s *Service) PublicRecipeInsights(ctx context.Context, locale string, id int64) (knowledge.Result, error) {
	r, err := s.GetPublicRecipe(ctx, locale, id)
	if err != nil {
		return knowledge.Result{}, err
	}
	return s.insights(ctx, r, locale)
}

func (s *Service) insights(ctx context.Context, r Recipe, locale string) (knowledge.Result, error) {
	names, set, err := s.allFoodNames(ctx, s.Store.Queries, r)
	if err != nil {
		return knowledge.Result{}, err
	}
	in := knowledge.Recipe{Name: r.Name, WorkMinutes: r.WorkMinutes, WaitMinutes: r.WaitMinutes}
	for _, st := range r.Steps {
		// Notes such as "stems only" or "finely chopped" belong to what the step says.
		parts := []string{st.Name, st.Instruction}
		k := knowledge.Step{Minutes: st.Minutes}
		for _, ing := range st.Ingredients {
			parts = append(parts, ing.Note)
			// Classes someone set for the food win over what its names suggest.
			out := knowledge.Ingredient{Names: append([]string{ing.Food.Name}, names[ing.Food.ID]...), Classes: set[ing.Food.ID], Amount: ing.Amount}
			if ing.Unit != nil {
				out.Unit = ing.Unit.Name
			}
			k.Ingredients = append(k.Ingredients, out)
		}
		k.Text = strings.Join(parts, " ")
		in.Steps = append(in.Steps, k)
	}
	return s.Knowledge.Analyze(in, locale), nil
}

// allFoodNames collects every name a recipe's foods have, in all languages and as aliases, so a
// food is recognised whatever language it was entered in.
func (s *Service) allFoodNames(ctx context.Context, q *sqlcgen.Queries, r Recipe) (map[int64][]string, map[int64][]string, error) {
	var ids []int64
	seen := map[int64]bool{}
	for _, st := range r.Steps {
		for _, ing := range st.Ingredients {
			if !seen[ing.Food.ID] {
				seen[ing.Food.ID] = true
				ids = append(ids, ing.Food.ID)
			}
		}
	}
	out, set := map[int64][]string{}, map[int64][]string{}
	if len(ids) == 0 {
		return out, set, nil
	}
	names, err := q.ListFoodNames(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range names {
		out[n.FoodID] = append(out[n.FoodID], n.Name)
		if n.Plural.Valid {
			out[n.FoodID] = append(out[n.FoodID], n.Plural.String)
		}
	}
	aliases, err := q.ListFoodAliases(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, a := range aliases {
		out[a.FoodID] = append(out[a.FoodID], a.Alias)
	}
	classes, err := q.ListFoodKnowledgeClasses(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range classes {
		if s.Knowledge.HasClass(c.Class) {
			set[c.FoodID] = append(set[c.FoodID], c.Class)
		}
	}
	return out, set, nil
}
