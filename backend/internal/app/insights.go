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

// DraftInsights reads a recipe that isn't saved, such as one being edited, so the editor can
// point out problems before saving. It saves nothing and creates no foods.
func (s *Service) DraftInsights(ctx context.Context, v Viewer, in RecipeInput) (knowledge.Result, error) {
	if len(in.Steps) > 200 {
		return knowledge.Result{}, invalid("too_many_steps", "A recipe can have at most 200 steps")
	}
	q := s.Store.Queries
	r := Recipe{RecipeSummary: RecipeSummary{Name: in.Name, WorkMinutes: in.WorkMinutes, WaitMinutes: in.WaitMinutes}}
	for _, st := range in.Steps {
		if len(st.Ingredients) > 200 {
			return knowledge.Result{}, invalid("too_many_ingredients", "A step can have at most 200 ingredients")
		}
		out := Step{Name: st.Name, Instruction: st.Instruction, Minutes: st.Minutes}
		for _, ing := range st.Ingredients {
			// A food that already exists counts with all its names and set kinds; a new name
			// counts by itself.
			ref := FoodRef{Name: strings.TrimSpace(ing.FoodName)}
			id := ing.FoodID
			if id <= 0 && ref.Name != "" {
				found, ok, err := lookupFood(ctx, q, ref.Name)
				if err != nil {
					return knowledge.Result{}, err
				}
				if ok {
					id = found
				}
			}
			if id > 0 {
				refs, err := foodRefs(ctx, q, []int64{id}, v.locale())
				if err != nil {
					return knowledge.Result{}, err
				}
				if known, ok := refs[id]; ok {
					ref = known
				}
			}
			if ref.Name == "" {
				continue
			}
			out.Ingredients = append(out.Ingredients, Ingredient{Food: ref, Amount: ing.Amount, Note: ing.Note})
		}
		r.Steps = append(r.Steps, out)
	}
	return s.insights(ctx, r, v.locale())
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
			if ing.Food.ID > 0 && !seen[ing.Food.ID] {
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
