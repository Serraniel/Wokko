package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/media"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

func sql64(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: true} }

// sameHousehold reports whether the viewer shares a household with the recipe's owner.
func sameHousehold(ctx context.Context, q *sqlcgen.Queries, v Viewer, r sqlcgen.Recipe) (bool, error) {
	if v.User == nil {
		return false, nil
	}
	owner, err := q.GetUser(ctx, r.OwnerID)
	if err != nil {
		return false, err
	}
	return owner.HouseholdID == v.User.HouseholdID, nil
}

// canView is the one permission rule for reading a recipe, used by every client.
func (s *Service) canView(ctx context.Context, q *sqlcgen.Queries, v Viewer, r sqlcgen.Recipe) (bool, error) {
	if r.DeletedAt.Valid {
		return v.id() == r.OwnerID, nil
	}
	if v.id() == r.OwnerID && v.User != nil {
		return true, nil
	}
	if r.Visibility == "public" {
		allowed, err := s.settingBool(ctx, "allow_public_recipes")
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	if v.User == nil {
		return false, nil
	}
	if r.Visibility != "private" {
		same, err := sameHousehold(ctx, q, v, r)
		if err != nil || same {
			return same, err
		}
	}
	n, err := q.IsSharedWith(ctx, sqlcgen.IsSharedWithParams{RecipeID: r.ID, UserID: v.id()})
	return n > 0, err
}

// canEdit: the owner, plus household members for recipes shared with the household.
func canEdit(ctx context.Context, q *sqlcgen.Queries, v Viewer, r sqlcgen.Recipe) (bool, error) {
	if v.User == nil {
		return false, nil
	}
	if v.id() == r.OwnerID {
		return true, nil
	}
	if r.Visibility == "private" {
		return false, nil
	}
	return sameHousehold(ctx, q, v, r)
}

func photoURLs(photo, share string) (string, string) {
	if photo == "" {
		return "", ""
	}
	suffix := ""
	if share != "" {
		suffix = "?share=" + share
	}
	return "/media/" + photo + suffix, "/media/" + media.ThumbName(photo) + suffix
}

func (s *Service) summaries(ctx context.Context, q *sqlcgen.Queries, v Viewer, rows []sqlcgen.Recipe) ([]RecipeSummary, error) {
	out := make([]RecipeSummary, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	links, err := q.ListRecipeKeywordsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	var kwIDs []int64
	seen := map[int64]bool{}
	byRecipe := map[int64][]int64{}
	for _, l := range links {
		byRecipe[l.RecipeID] = append(byRecipe[l.RecipeID], l.KeywordID)
		if !seen[l.KeywordID] {
			seen[l.KeywordID] = true
			kwIDs = append(kwIDs, l.KeywordID)
		}
	}
	kws, err := keywordRefs(ctx, q, kwIDs, v.locale())
	if err != nil {
		return nil, err
	}
	stats := map[int64]sqlcgen.RecipeStatsRow{}
	if v.User != nil {
		st, err := q.RecipeStats(ctx, sqlcgen.RecipeStatsParams{UserID: v.id(), Ids: ids})
		if err != nil {
			return nil, err
		}
		for _, x := range st {
			stats[x.RecipeID] = x
		}
	}
	owners := map[int64]*Person{}
	for i, r := range rows {
		item := RecipeSummary{
			ID: r.ID, Name: r.Name, Description: r.Description, WorkMinutes: r.WorkMinutes, WaitMinutes: r.WaitMinutes,
			Servings: r.Servings, ServingsText: r.ServingsText, Visibility: r.Visibility, Keywords: []Keyword{},
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt.String,
		}
		item.PhotoURL, item.ThumbURL = photoURLs(r.Photo.String, "")
		for _, id := range byRecipe[r.ID] {
			item.Keywords = append(item.Keywords, kws[id])
		}
		sort.Slice(item.Keywords, func(a, b int) bool { return item.Keywords[a].Name < item.Keywords[b].Name })
		if st, ok := stats[r.ID]; ok {
			item.TimesCooked = st.TimesCooked
			item.LastCooked = st.LastCooked
			if st.Rating > 0 {
				rating := st.Rating
				item.Rating = &rating
			}
		}
		if v.User != nil { // anonymous viewers never learn who wrote a recipe
			if _, ok := owners[r.OwnerID]; !ok {
				u, err := q.GetUser(ctx, r.OwnerID)
				if err != nil {
					return nil, err
				}
				p := toPerson(u)
				owners[r.OwnerID] = &p
			}
			item.Owner = owners[r.OwnerID]
		}
		out[i] = item
	}
	return out, nil
}

func (s *Service) buildRecipe(ctx context.Context, q *sqlcgen.Queries, v Viewer, r sqlcgen.Recipe) (Recipe, error) {
	sum, err := s.summaries(ctx, q, v, []sqlcgen.Recipe{r})
	if err != nil {
		return Recipe{}, err
	}
	steps, err := q.ListSteps(ctx, r.ID)
	if err != nil {
		return Recipe{}, err
	}
	ings, err := q.ListIngredientsForRecipe(ctx, r.ID)
	if err != nil {
		return Recipe{}, err
	}
	var foodIDs, unitIDs []int64
	for _, i := range ings {
		foodIDs = append(foodIDs, i.FoodID)
		if i.UnitID.Valid {
			unitIDs = append(unitIDs, i.UnitID.Int64)
		}
	}
	foods, err := foodRefs(ctx, q, foodIDs, v.locale())
	if err != nil {
		return Recipe{}, err
	}
	units, err := unitRefs(ctx, q, unitIDs, v.locale())
	if err != nil {
		return Recipe{}, err
	}
	byStep := map[int64][]Ingredient{}
	for _, i := range ings {
		ing := Ingredient{ID: i.ID, Food: foods[i.FoodID], Note: i.Note}
		if i.UnitID.Valid {
			u := units[i.UnitID.Int64]
			ing.Unit = &u
		}
		if i.Amount.Valid {
			a := i.Amount.Float64
			ing.Amount = &a
		}
		byStep[i.StepID] = append(byStep[i.StepID], ing)
	}
	out := Recipe{RecipeSummary: sum[0], SourceURL: r.SourceUrl, Version: r.Version, Steps: make([]Step, len(steps))}
	for i, st := range steps {
		list := byStep[st.ID]
		if list == nil {
			list = []Ingredient{}
		}
		out.Steps[i] = Step{ID: st.ID, Name: st.Name, Instruction: st.Instruction, Minutes: st.Minutes, Ingredients: list}
	}
	out.CanEdit, err = canEdit(ctx, q, v, r)
	if err != nil {
		return Recipe{}, err
	}
	out.CanManage = v.User != nil && v.id() == r.OwnerID
	return out, nil
}

// GetRecipe returns a recipe the viewer may see. A forbidden recipe looks like a missing one.
func (s *Service) GetRecipe(ctx context.Context, v Viewer, id int64) (Recipe, error) {
	q := s.Store.Queries
	r, err := q.GetRecipe(ctx, id)
	if isNoRows(err) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}
	ok, err := s.canView(ctx, q, v, r)
	if err != nil {
		return Recipe{}, err
	}
	if !ok {
		return Recipe{}, ErrNotFound
	}
	return s.buildRecipe(ctx, q, v, r)
}

func (s *Service) statsFor(ctx context.Context, q *sqlcgen.Queries, v Viewer, keys []sqlcgen.ListVisibleRecipeKeysRow) (map[int64]sqlcgen.RecipeStatsRow, error) {
	out := map[int64]sqlcgen.RecipeStatsRow{}
	if len(keys) == 0 {
		return out, nil
	}
	ids := make([]int64, len(keys))
	for i, k := range keys {
		ids[i] = k.ID
	}
	rows, err := q.RecipeStats(ctx, sqlcgen.RecipeStatsParams{UserID: v.id(), Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.RecipeID] = r
	}
	return out, nil
}

// narrowRecipes applies the filters only kitshn uses. They run after the permission check
// in ListVisibleRecipeKeys, so they can only ever remove recipes from what's visible.
func (s *Service) narrowRecipes(ctx context.Context, q *sqlcgen.Queries, v Viewer, keys []sqlcgen.ListVisibleRecipeKeysRow, f RecipeQuery) ([]sqlcgen.ListVisibleRecipeKeysRow, error) {
	keep := func(allowed map[int64]bool) {
		out := keys[:0:0]
		for _, k := range keys {
			if allowed[k.ID] {
				out = append(out, k)
			}
		}
		keys = out
	}
	if len(f.AnyKeyword) > 0 && len(keys) > 0 {
		ids := make([]int64, len(keys))
		for i, k := range keys {
			ids[i] = k.ID
		}
		links, err := q.ListRecipeKeywordsFor(ctx, ids)
		if err != nil {
			return nil, err
		}
		want := map[int64]bool{}
		for _, id := range f.AnyKeyword {
			want[id] = true
		}
		allowed := map[int64]bool{}
		for _, l := range links {
			if want[l.KeywordID] {
				allowed[l.RecipeID] = true
			}
		}
		keep(allowed)
	}
	if len(f.AnyFood) > 0 {
		allowed := map[int64]bool{}
		for _, food := range f.AnyFood {
			ids, err := q.ListRecipeIDsUsingFood(ctx, food)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				allowed[id] = true
			}
		}
		keep(allowed)
	}
	if len(f.AnyBook) > 0 {
		allowed := map[int64]bool{}
		for _, book := range f.AnyBook {
			if _, err := s.bookAccess(ctx, q, v, book); err != nil {
				continue
			}
			rows, err := q.ListBookEntries(ctx, book)
			if err != nil {
				return nil, err
			}
			for _, e := range rows {
				allowed[e.RecipeID] = true
			}
		}
		keep(allowed)
	}
	if f.RatingMin > 0 || f.RatingMax > 0 || f.CookedMin > 0 || f.CookedMax > 0 {
		stats, err := s.statsFor(ctx, q, v, keys)
		if err != nil {
			return nil, err
		}
		allowed := map[int64]bool{}
		for _, k := range keys {
			st := stats[k.ID]
			if (f.RatingMin > 0 && st.Rating < float64(f.RatingMin)) || (f.RatingMax > 0 && st.Rating > float64(f.RatingMax)) ||
				(f.CookedMin > 0 && st.TimesCooked < f.CookedMin) || (f.CookedMax > 0 && st.TimesCooked > f.CookedMax) {
				continue
			}
			allowed[k.ID] = true
		}
		keep(allowed)
	}
	return keys, nil
}

// RecipeQuery filters a recipe list.
type RecipeQuery struct {
	Q, Sort          string
	KeywordID, Owner int64
	Limit, Offset    int64
	// AnyKeyword, AnyFood and AnyBook keep recipes matching at least one of the IDs.
	AnyKeyword, AnyFood, AnyBook []int64
	// RatingMin and the like keep a range. Zero means no limit.
	RatingMin, RatingMax, CookedMin, CookedMax int64
}

// ListRecipes returns what the viewer can reach: their own, their household's, shared with them.
func (s *Service) ListRecipes(ctx context.Context, v Viewer, f RecipeQuery) (Page[RecipeSummary], error) {
	limit, offset := clampPage(f.Limit, f.Offset)
	q := s.Store.Queries
	keys, err := q.ListVisibleRecipeKeys(ctx, sqlcgen.ListVisibleRecipeKeysParams{
		ViewerID: v.id(), HouseholdID: v.User.HouseholdID, Q: text.EscapeLike(text.Normalize(f.Q)), KeywordID: f.KeywordID, OwnerFilter: f.Owner,
	})
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	if keys, err = s.narrowRecipes(ctx, q, v, keys, f); err != nil {
		return Page[RecipeSummary]{}, err
	}
	var stats map[int64]sqlcgen.RecipeStatsRow
	if strings.Contains(f.Sort, "rating") || strings.Contains(f.Sort, "cooked") || strings.Contains(f.Sort, "times") {
		if stats, err = s.statsFor(ctx, q, v, keys); err != nil {
			return Page[RecipeSummary]{}, err
		}
	}
	switch f.Sort {
	case "name":
		sort.SliceStable(keys, func(i, j int) bool { return text.Normalize(keys[i].Name) < text.Normalize(keys[j].Name) })
	case "-name":
		sort.SliceStable(keys, func(i, j int) bool { return text.Normalize(keys[i].Name) > text.Normalize(keys[j].Name) })
	case "created":
		sort.SliceStable(keys, func(i, j int) bool { return keys[i].CreatedAt > keys[j].CreatedAt })
	case "-created":
		sort.SliceStable(keys, func(i, j int) bool { return keys[i].CreatedAt < keys[j].CreatedAt })
	case "rating":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].Rating < stats[keys[j].ID].Rating })
	case "-rating":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].Rating > stats[keys[j].ID].Rating })
	case "times":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].TimesCooked < stats[keys[j].ID].TimesCooked })
	case "-times":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].TimesCooked > stats[keys[j].ID].TimesCooked })
	case "cooked":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].LastCooked < stats[keys[j].ID].LastCooked })
	case "-cooked":
		sort.SliceStable(keys, func(i, j int) bool { return stats[keys[i].ID].LastCooked > stats[keys[j].ID].LastCooked })
	case "random":
		rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	default:
		sort.SliceStable(keys, func(i, j int) bool { return keys[i].UpdatedAt > keys[j].UpdatedAt })
	}
	total := int64(len(keys))
	if offset >= total {
		return Page[RecipeSummary]{Items: []RecipeSummary{}, Total: total}, nil
	}
	end := min(offset+limit, total)
	ids := make([]int64, 0, end-offset)
	pos := map[int64]int{}
	for i, k := range keys[offset:end] {
		ids = append(ids, k.ID)
		pos[k.ID] = i
	}
	rows, err := q.ListRecipesByIDs(ctx, ids)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	sort.Slice(rows, func(i, j int) bool { return pos[rows[i].ID] < pos[rows[j].ID] })
	items, err := s.summaries(ctx, q, v, rows)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	return Page[RecipeSummary]{Items: items, Total: total}, nil
}

func validateRecipe(in *RecipeInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return invalid("name_required", "Please enter a name")
	}
	if len(in.Name) > 200 {
		return invalid("name_too_long", "This name is too long")
	}
	if in.Servings < 1 {
		in.Servings = 1
	}
	if in.WorkMinutes < 0 || in.WaitMinutes < 0 {
		return invalid("bad_time", "Times can't be negative")
	}
	switch in.Visibility {
	case "":
		in.Visibility = "household"
	case "private", "household", "public":
	default:
		return invalid("bad_visibility", "Unknown visibility")
	}
	if u := strings.TrimSpace(in.SourceURL); u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return invalid("bad_url", "The source must be a web address")
	}
	for i := range in.Steps {
		if in.Steps[i].Minutes < 0 {
			return invalid("bad_time", "Times can't be negative")
		}
		for j := range in.Steps[i].Ingredients {
			ing := &in.Steps[i].Ingredients[j]
			if ing.FoodID <= 0 && strings.TrimSpace(ing.FoodName) == "" {
				return invalid("ingredient_required", "Each ingredient needs a name")
			}
			if ing.Amount != nil && *ing.Amount < 0 {
				return invalid("bad_amount", "Amounts can't be negative")
			}
		}
	}
	return nil
}

func (s *Service) resolveFood(ctx context.Context, q *sqlcgen.Queries, v Viewer, ing IngredientInput) (int64, error) {
	if ing.FoodID > 0 {
		if _, err := q.GetFood(ctx, ing.FoodID); isNoRows(err) {
			return 0, invalid("unknown_food", "This ingredient no longer exists")
		} else if err != nil {
			return 0, err
		}
		return ing.FoodID, nil
	}
	name := strings.TrimSpace(ing.FoodName)
	if id, err := q.FindFoodIDByName(ctx, name); err == nil {
		return id, nil
	} else if !isNoRows(err) {
		return 0, err
	}
	f, err := q.CreateFood(ctx, sqlcgen.CreateFoodParams{CreatedBy: nullID(v.id())})
	if err != nil {
		return 0, err
	}
	err = q.UpsertFoodName(ctx, sqlcgen.UpsertFoodNameParams{FoodID: f.ID, Locale: v.locale(), Name: name, SearchName: text.Normalize(name)})
	return f.ID, err
}

func (s *Service) resolveUnit(ctx context.Context, q *sqlcgen.Queries, v Viewer, ing IngredientInput) (sql.NullInt64, error) {
	if ing.UnitID > 0 {
		if _, err := q.GetUnit(ctx, ing.UnitID); isNoRows(err) {
			return sql.NullInt64{}, invalid("unknown_unit", "This unit no longer exists")
		} else if err != nil {
			return sql.NullInt64{}, err
		}
		return sql64(ing.UnitID), nil
	}
	name := strings.TrimSpace(ing.UnitName)
	if name == "" {
		return sql.NullInt64{}, nil
	}
	if id, err := q.FindUnitIDByName(ctx, name); err == nil {
		return sql64(id), nil
	} else if !isNoRows(err) {
		return sql.NullInt64{}, err
	}
	u, err := q.CreateUnit(ctx, nullID(v.id()))
	if err != nil {
		return sql.NullInt64{}, err
	}
	err = q.UpsertUnitName(ctx, sqlcgen.UpsertUnitNameParams{UnitID: u.ID, Locale: v.locale(), Name: name, SearchName: text.Normalize(name)})
	return sql64(u.ID), err
}

func (s *Service) resolveKeywords(ctx context.Context, q *sqlcgen.Queries, v Viewer, names []string) ([]int64, error) {
	var ids []int64
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		norm := text.Normalize(n)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		id, err := q.FindKeywordBySearchName(ctx, norm)
		if isNoRows(err) {
			k, err := q.CreateKeyword(ctx)
			if err != nil {
				return nil, err
			}
			id = k.ID
			if err := q.UpsertKeywordName(ctx, sqlcgen.UpsertKeywordNameParams{KeywordID: id, Locale: v.locale(), Name: n, SearchName: norm}); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// writeContent replaces steps, ingredients and keywords and returns the recipe's search text.
// It runs inside the save transaction, and new foods, units and keywords are created there
// too, so a failed save leaves nothing behind.
func (s *Service) writeContent(ctx context.Context, q *sqlcgen.Queries, v Viewer, recipeID int64, in RecipeInput) (string, error) {
	// Steps keep their IDs from one save to the next, because kitshn holds on to them.
	oldSteps, err := q.ListSteps(ctx, recipeID)
	if err != nil {
		return "", err
	}
	for i := len(in.Steps); i < len(oldSteps); i++ {
		if err := q.DeleteStep(ctx, oldSteps[i].ID); err != nil {
			return "", err
		}
	}
	if err := q.DeleteRecipeKeywords(ctx, recipeID); err != nil {
		return "", err
	}
	parts := []string{in.Name, in.Description}
	var foodIDs []int64
	for i, st := range in.Steps {
		var stepID int64
		if i < len(oldSteps) {
			stepID = oldSteps[i].ID
			if err := q.DeleteStepIngredients(ctx, stepID); err != nil {
				return "", err
			}
			if err := q.UpdateStep(ctx, sqlcgen.UpdateStepParams{ID: stepID, Position: int64(i), Name: strings.TrimSpace(st.Name), Instruction: st.Instruction, Minutes: st.Minutes}); err != nil {
				return "", err
			}
		} else {
			stepID, err = q.CreateStep(ctx, sqlcgen.CreateStepParams{RecipeID: recipeID, Position: int64(i), Name: strings.TrimSpace(st.Name), Instruction: st.Instruction, Minutes: st.Minutes})
			if err != nil {
				return "", err
			}
		}
		parts = append(parts, st.Name, st.Instruction)
		for j, ing := range st.Ingredients {
			foodID, err := s.resolveFood(ctx, q, v, ing)
			if err != nil {
				return "", err
			}
			unit, err := s.resolveUnit(ctx, q, v, ing)
			if err != nil {
				return "", err
			}
			var amount sql.NullFloat64
			if ing.Amount != nil {
				amount = sql.NullFloat64{Float64: *ing.Amount, Valid: true}
			}
			foodIDs = append(foodIDs, foodID)
			if err := q.CreateIngredient(ctx, sqlcgen.CreateIngredientParams{StepID: stepID, Position: int64(j), FoodID: foodID, UnitID: unit, Amount: amount, Note: strings.TrimSpace(ing.Note)}); err != nil {
				return "", err
			}
			parts = append(parts, ing.Note)
		}
	}
	kwIDs, err := s.resolveKeywords(ctx, q, v, in.Keywords)
	if err != nil {
		return "", err
	}
	for _, id := range kwIDs {
		if err := q.AddRecipeKeyword(ctx, sqlcgen.AddRecipeKeywordParams{RecipeID: recipeID, KeywordID: id}); err != nil {
			return "", err
		}
	}
	parts = append(parts, in.Keywords...)
	if len(foodIDs) > 0 { // every language's name, so a recipe is found in any of them
		names, err := q.ListAllFoodSearchNames(ctx, foodIDs)
		if err != nil {
			return "", err
		}
		for _, n := range names {
			parts = append(parts, n.SearchName)
		}
	}
	return text.Normalize(strings.Join(parts, " ")), nil
}

func (s *Service) checkPublicAllowed(ctx context.Context, q *sqlcgen.Queries, visibility string) error {
	if visibility != "public" {
		return nil
	}
	v, err := q.GetSetting(ctx, "allow_public_recipes")
	if err != nil {
		return err
	}
	if v != "true" {
		return fail(http.StatusForbidden, "public_disabled", "Public recipes are turned off on this server")
	}
	return nil
}

// CreateRecipe saves a whole recipe in one transaction.
func (s *Service) CreateRecipe(ctx context.Context, v Viewer, in RecipeInput) (Recipe, error) {
	if err := validateRecipe(&in); err != nil {
		return Recipe{}, err
	}
	var id int64
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := s.checkPublicAllowed(ctx, q, in.Visibility); err != nil {
			return err
		}
		r, err := q.CreateRecipe(ctx, sqlcgen.CreateRecipeParams{
			OwnerID: v.id(), Name: in.Name, Description: in.Description, WorkMinutes: in.WorkMinutes, WaitMinutes: in.WaitMinutes,
			Servings: in.Servings, ServingsText: in.ServingsText, SourceUrl: strings.TrimSpace(in.SourceURL), Visibility: in.Visibility,
		})
		if err != nil {
			return err
		}
		id = r.ID
		search, err := s.writeContent(ctx, q, v, id, in)
		if err != nil {
			return err
		}
		_, err = q.UpdateRecipe(ctx, sqlcgen.UpdateRecipeParams{
			Name: in.Name, Description: in.Description, WorkMinutes: in.WorkMinutes, WaitMinutes: in.WaitMinutes, Servings: in.Servings,
			ServingsText: in.ServingsText, SourceUrl: strings.TrimSpace(in.SourceURL), Visibility: in.Visibility, SearchText: search, ID: id, BaseVersion: r.Version,
		})
		return err
	})
	if err != nil {
		return Recipe{}, err
	}
	return s.GetRecipe(ctx, v, id)
}

// UpdateRecipe saves a whole recipe in one transaction. The previous state is kept as a
// version, and an edit based on an outdated version is rejected instead of overwriting.
func (s *Service) UpdateRecipe(ctx context.Context, v Viewer, id int64, in RecipeInput) (Recipe, error) {
	if err := validateRecipe(&in); err != nil {
		return Recipe{}, err
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		r, err := q.GetRecipe(ctx, id)
		if isNoRows(err) || (err == nil && r.DeletedAt.Valid) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		ok, err := canEdit(ctx, q, v, r)
		if err != nil {
			return err
		}
		if !ok {
			if seen, err := s.canView(ctx, q, v, r); err != nil {
				return err
			} else if !seen {
				return ErrNotFound
			}
			return ErrForbidden
		}
		// Only the owner may change who can see a recipe.
		if v.id() != r.OwnerID && in.Visibility != r.Visibility {
			return ErrForbidden
		}
		if err := s.checkPublicAllowed(ctx, q, in.Visibility); err != nil {
			if r.Visibility != "public" {
				return err
			}
		}
		if in.BaseVersion != r.Version {
			return ErrConflict
		}
		if err := s.saveVersion(ctx, q, v, r); err != nil {
			return err
		}
		search, err := s.writeContent(ctx, q, v, id, in)
		if err != nil {
			return err
		}
		n, err := q.UpdateRecipe(ctx, sqlcgen.UpdateRecipeParams{
			Name: in.Name, Description: in.Description, WorkMinutes: in.WorkMinutes, WaitMinutes: in.WaitMinutes, Servings: in.Servings,
			ServingsText: in.ServingsText, SourceUrl: strings.TrimSpace(in.SourceURL), Visibility: in.Visibility, SearchText: search, ID: id, BaseVersion: in.BaseVersion,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		return Recipe{}, err
	}
	return s.GetRecipe(ctx, v, id)
}

func (s *Service) saveVersion(ctx context.Context, q *sqlcgen.Queries, v Viewer, r sqlcgen.Recipe) error {
	doc, err := s.buildRecipe(ctx, q, v, r)
	if err != nil {
		return err
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if err := q.CreateRecipeVersion(ctx, sqlcgen.CreateRecipeVersionParams{RecipeID: r.ID, Version: r.Version, Document: string(b), SavedBy: nullID(v.id())}); err != nil {
		return err
	}
	return q.PruneRecipeVersions(ctx, r.ID)
}

// DeleteRecipe moves a recipe to the trash. Only the owner can do this.
func (s *Service) DeleteRecipe(ctx context.Context, v Viewer, id int64) error {
	return s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, r sqlcgen.Recipe) error {
		return q.SoftDeleteRecipe(ctx, id)
	})
}

func (s *Service) RestoreRecipe(ctx context.Context, v Viewer, id int64) error {
	return s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, r sqlcgen.Recipe) error {
		return q.RestoreRecipe(ctx, id)
	})
}

// PurgeRecipe permanently removes a recipe that is already in the trash.
func (s *Service) PurgeRecipe(ctx context.Context, v Viewer, id int64) error {
	var photo string
	err := s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, r sqlcgen.Recipe) error {
		if !r.DeletedAt.Valid {
			return fail(http.StatusConflict, "not_in_trash", "Move this recipe to the trash first")
		}
		photo = r.Photo.String
		return q.PurgeRecipe(ctx, id)
	})
	if err != nil {
		return err
	}
	if photo != "" {
		s.removePhotos([]string{photo})
	}
	return nil
}

func (s *Service) ownerAction(ctx context.Context, v Viewer, id int64, fn func(*sqlcgen.Queries, sqlcgen.Recipe) error) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		r, err := q.GetRecipe(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if r.OwnerID != v.id() {
			if seen, err := s.canView(ctx, q, v, r); err != nil {
				return err
			} else if !seen {
				return ErrNotFound
			}
			return ErrForbidden
		}
		return fn(q, r)
	})
}

func (s *Service) ListTrash(ctx context.Context, v Viewer) ([]RecipeSummary, error) {
	rows, err := s.Store.Queries.ListTrash(ctx, v.id())
	if err != nil {
		return nil, err
	}
	return s.summaries(ctx, s.Store.Queries, v, rows)
}

// SetPhoto stores the new image first, then points the recipe at it, and removes the old
// file only after that succeeded. A failure at any stage keeps the previous photo.
func (s *Service) SetPhoto(ctx context.Context, v Viewer, id int64, r io.Reader) (Recipe, error) {
	if err := s.checkEdit(ctx, v, id); err != nil {
		return Recipe{}, err
	}
	name, err := s.Media.Save(r)
	if errors.Is(err, media.ErrInvalidImage) {
		return Recipe{}, invalid("invalid_image", err.Error())
	}
	if err != nil {
		return Recipe{}, err
	}
	old, err := s.replacePhoto(ctx, v, id, name)
	if err != nil {
		s.removePhotos([]string{name})
		return Recipe{}, err
	}
	if old != "" {
		s.removePhotos([]string{old})
	}
	return s.GetRecipe(ctx, v, id)
}

func (s *Service) DeletePhoto(ctx context.Context, v Viewer, id int64) error {
	old, err := s.replacePhoto(ctx, v, id, "")
	if err != nil {
		return err
	}
	if old != "" {
		s.removePhotos([]string{old})
	}
	return nil
}

func (s *Service) replacePhoto(ctx context.Context, v Viewer, id int64, name string) (string, error) {
	var old string
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		r, err := q.GetRecipe(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		ok, err := canEdit(ctx, q, v, r)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		old = r.Photo.String
		return q.SetRecipePhoto(ctx, sqlcgen.SetRecipePhotoParams{Photo: nullStr(name), ID: id})
	})
	return old, err
}

func (s *Service) checkEdit(ctx context.Context, v Viewer, id int64) error {
	q := s.Store.Queries
	r, err := q.GetRecipe(ctx, id)
	if isNoRows(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	ok, err := canEdit(ctx, q, v, r)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// CanSeePhoto decides whether a request may load a photo file.
func (s *Service) CanSeePhoto(ctx context.Context, v Viewer, file, shareToken string) (bool, error) {
	q := s.Store.Queries
	r, err := q.GetRecipeByPhoto(ctx, nullStr(media.OriginalName(file)))
	if isNoRows(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if shareToken != "" {
		n, err := q.CountValidShareLink(ctx, sqlcgen.CountValidShareLinkParams{Token: shareToken, RecipeID: r.ID, ExpiresAt: nullStr(s.stamp(s.Now()))})
		if err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
		if covered, err := s.collectionCovers(ctx, shareToken, r); err != nil || covered {
			return covered, err
		}
	}
	return s.canView(ctx, q, v, r)
}

// --- versions ---

func (s *Service) ListVersions(ctx context.Context, v Viewer, id int64) ([]RecipeVersion, error) {
	if err := s.checkEdit(ctx, v, id); err != nil {
		return nil, err
	}
	rows, err := s.Store.Queries.ListRecipeVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]RecipeVersion, len(rows))
	for i, r := range rows {
		out[i] = RecipeVersion{ID: r.ID, Version: r.Version, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// RestoreVersion saves an old version as the newest one, so the current state stays in history.
func (s *Service) RestoreVersion(ctx context.Context, v Viewer, id, versionID int64) (Recipe, error) {
	if err := s.checkEdit(ctx, v, id); err != nil {
		return Recipe{}, err
	}
	row, err := s.Store.Queries.GetRecipeVersion(ctx, sqlcgen.GetRecipeVersionParams{RecipeID: id, ID: versionID})
	if isNoRows(err) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}
	var doc Recipe
	if err := json.Unmarshal([]byte(row.Document), &doc); err != nil {
		return Recipe{}, fmt.Errorf("read saved version: %w", err)
	}
	cur, err := s.Store.Queries.GetRecipe(ctx, id)
	if err != nil {
		return Recipe{}, err
	}
	in := RecipeInput{
		Name: doc.Name, Description: doc.Description, WorkMinutes: doc.WorkMinutes, WaitMinutes: doc.WaitMinutes, Servings: doc.Servings,
		ServingsText: doc.ServingsText, SourceURL: doc.SourceURL, Visibility: cur.Visibility, BaseVersion: cur.Version,
	}
	for _, k := range doc.Keywords {
		in.Keywords = append(in.Keywords, k.Name)
	}
	for _, st := range doc.Steps {
		si := StepInput{Name: st.Name, Instruction: st.Instruction, Minutes: st.Minutes}
		for _, ing := range st.Ingredients {
			ii := IngredientInput{FoodID: ing.Food.ID, FoodName: ing.Food.Name, Amount: ing.Amount, Note: ing.Note}
			if ing.Unit != nil {
				ii.UnitID = ing.Unit.ID
				ii.UnitName = ing.Unit.Name
			}
			si.Ingredients = append(si.Ingredients, ii)
		}
		in.Steps = append(in.Steps, si)
	}
	return s.UpdateRecipe(ctx, v, id, in)
}

// --- sharing ---

func (s *Service) GetSharing(ctx context.Context, v Viewer, id int64) (Sharing, error) {
	var out Sharing
	err := s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, _ sqlcgen.Recipe) error {
		users, err := q.ListRecipeShareUsers(ctx, id)
		if err != nil {
			return err
		}
		links, err := q.ListShareLinks(ctx, id)
		if err != nil {
			return err
		}
		out = Sharing{Users: make([]Person, len(users)), Links: make([]ShareLink, len(links))}
		for i, u := range users {
			out.Users[i] = toPerson(u)
		}
		for i, l := range links {
			out.Links[i] = s.toShareLink(l)
		}
		return nil
	})
	return out, err
}

func (s *Service) toShareLink(l sqlcgen.ShareLink) ShareLink {
	return ShareLink{ID: l.ID, URL: s.Cfg.PublicURL + "/s/" + l.Token, ExpiresAt: l.ExpiresAt.String, CreatedAt: l.CreatedAt}
}

// ShareWithUser shares by exact handle, or with an existing friend or household member by ID.
func (s *Service) ShareWithUser(ctx context.Context, v Viewer, id int64, handle string) error {
	u, err := s.lookupHandle(ctx, v, handle)
	if err != nil {
		return err
	}
	return s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, r sqlcgen.Recipe) error {
		if r.DeletedAt.Valid {
			return ErrNotFound
		}
		return q.AddRecipeShare(ctx, sqlcgen.AddRecipeShareParams{RecipeID: id, UserID: u.ID})
	})
}

func (s *Service) UnshareWithUser(ctx context.Context, v Viewer, id, userID int64) error {
	return s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, _ sqlcgen.Recipe) error {
		return q.RemoveRecipeShare(ctx, sqlcgen.RemoveRecipeShareParams{RecipeID: id, UserID: userID})
	})
}

func (s *Service) CreateShareLink(ctx context.Context, v Viewer, id, expiresInDays int64) (ShareLink, error) {
	token, err := auth.NewToken()
	if err != nil {
		return ShareLink{}, err
	}
	var out ShareLink
	err = s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, r sqlcgen.Recipe) error {
		if r.DeletedAt.Valid {
			return ErrNotFound
		}
		p := sqlcgen.CreateShareLinkParams{Token: token, RecipeID: id, CreatedBy: v.id()}
		if expiresInDays > 0 {
			p.ExpiresAt = nullStr(s.stamp(s.Now().AddDate(0, 0, int(expiresInDays))))
		}
		l, err := q.CreateShareLink(ctx, p)
		if err != nil {
			return err
		}
		out = s.toShareLink(l)
		return nil
	})
	return out, err
}

func (s *Service) RevokeShareLink(ctx context.Context, v Viewer, id, linkID int64) error {
	return s.ownerAction(ctx, v, id, func(q *sqlcgen.Queries, _ sqlcgen.Recipe) error {
		return q.DeleteShareLink(ctx, sqlcgen.DeleteShareLinkParams{ID: linkID, RecipeID: id})
	})
}

// anonymous strips everything that identifies or relates to a specific account.
func anonymous(r Recipe) Recipe {
	r.Owner = nil
	r.CanEdit = false
	r.CanManage = false
	r.Rating = nil
	r.LastCooked = ""
	r.TimesCooked = 0
	return r
}

// GetSharedRecipe opens a share link without any account.
func (s *Service) GetSharedRecipe(ctx context.Context, locale, token string) (Recipe, error) {
	q := s.Store.Queries
	l, err := q.GetShareLinkByToken(ctx, token)
	if isNoRows(err) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}
	if l.ExpiresAt.Valid && l.ExpiresAt.String <= s.stamp(s.Now()) {
		return Recipe{}, ErrNotFound
	}
	r, err := q.GetRecipe(ctx, l.RecipeID)
	if err != nil || r.DeletedAt.Valid {
		return Recipe{}, ErrNotFound
	}
	doc, err := s.buildRecipe(ctx, q, Viewer{Locale: locale}, r)
	if err != nil {
		return Recipe{}, err
	}
	doc.PhotoURL, doc.ThumbURL = photoURLs(r.Photo.String, token)
	return anonymous(doc), nil
}

func (s *Service) ListPublicRecipes(ctx context.Context, locale, query string, limit, offset int64) (Page[RecipeSummary], error) {
	allowed, err := s.settingBool(ctx, "allow_public_recipes")
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	if !allowed {
		return Page[RecipeSummary]{Items: []RecipeSummary{}}, nil
	}
	limit, offset = clampPage(limit, offset)
	q := s.Store.Queries
	n := text.EscapeLike(text.Normalize(query))
	rows, err := q.ListPublicRecipes(ctx, sqlcgen.ListPublicRecipesParams{Q: n, Lim: limit, Off: offset})
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	total, err := q.CountPublicRecipes(ctx, n)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	items, err := s.summaries(ctx, q, Viewer{Locale: locale}, rows)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	return Page[RecipeSummary]{Items: items, Total: total}, nil
}

func (s *Service) GetPublicRecipe(ctx context.Context, locale string, id int64) (Recipe, error) {
	doc, err := s.GetRecipe(ctx, Viewer{Locale: locale}, id)
	if err != nil {
		return Recipe{}, err
	}
	return anonymous(doc), nil
}

// --- cook log ---

func (s *Service) AddCookLog(ctx context.Context, v Viewer, id int64, in CookLogInput) (CookLogEntry, error) {
	if _, err := s.GetRecipe(ctx, v, id); err != nil {
		return CookLogEntry{}, err
	}
	if in.Rating != nil && (*in.Rating < 1 || *in.Rating > 5) {
		return CookLogEntry{}, invalid("bad_rating", "Ratings go from 1 to 5")
	}
	p := sqlcgen.CreateCookLogParams{RecipeID: id, UserID: v.id(), Comment: strings.TrimSpace(in.Comment)}
	if in.Servings != nil {
		p.Servings = sql64(*in.Servings)
	}
	if in.Rating != nil {
		p.Rating = sql64(*in.Rating)
	}
	var row sqlcgen.CookLog
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) (err error) {
		row, err = q.CreateCookLog(ctx, p)
		return err
	})
	if err != nil {
		return CookLogEntry{}, err
	}
	return toCookLog(row), nil
}

func toCookLog(r sqlcgen.CookLog) CookLogEntry {
	e := CookLogEntry{ID: r.ID, Comment: r.Comment, CreatedAt: r.CreatedAt}
	if r.Servings.Valid {
		v := r.Servings.Int64
		e.Servings = &v
	}
	if r.Rating.Valid {
		v := r.Rating.Int64
		e.Rating = &v
	}
	return e
}

func (s *Service) ListCookLog(ctx context.Context, v Viewer, id int64) ([]CookLogEntry, error) {
	if _, err := s.GetRecipe(ctx, v, id); err != nil {
		return nil, err
	}
	rows, err := s.Store.Queries.ListCookLog(ctx, sqlcgen.ListCookLogParams{RecipeID: id, UserID: v.id()})
	if err != nil {
		return nil, err
	}
	out := make([]CookLogEntry, len(rows))
	for i, r := range rows {
		out[i] = toCookLog(r)
	}
	return out, nil
}

// ExportData is one user's data in an open format that needs no Wokko to read.
type ExportData struct {
	Format     string   `json:"format"`
	ExportedAt string   `json:"exported_at"`
	User       Person   `json:"user"`
	Recipes    []Recipe `json:"recipes"`
}

// Export gathers everything a user owns and the photo files that go with it.
func (s *Service) Export(ctx context.Context, v Viewer) (ExportData, []string, error) {
	q := s.Store.Queries
	rows, err := q.ListRecipesByOwner(ctx, v.id())
	if err != nil {
		return ExportData{}, nil, err
	}
	out := ExportData{Format: "wokko-export-1", ExportedAt: s.stamp(s.Now()), User: toPerson(*v.User), Recipes: []Recipe{}}
	var photos []string
	for _, r := range rows {
		if r.DeletedAt.Valid {
			continue
		}
		doc, err := s.buildRecipe(ctx, q, v, r)
		if err != nil {
			return ExportData{}, nil, err
		}
		out.Recipes = append(out.Recipes, doc)
		if r.Photo.Valid {
			photos = append(photos, r.Photo.String)
		}
	}
	return out, photos, nil
}

// PurgeExpiredTrash removes recipes that stayed in the trash past the retention period.
func (s *Service) PurgeExpiredTrash(ctx context.Context) error {
	cutoff := s.stamp(s.Now().Add(-time.Duration(s.Cfg.TrashRetentionDays) * 24 * time.Hour))
	var photos []string
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		rows, err := q.ListExpiredTrash(ctx, nullStr(cutoff))
		if err != nil {
			return err
		}
		for _, r := range rows {
			if r.Photo.Valid {
				photos = append(photos, r.Photo.String)
			}
			if err := q.PurgeRecipe(ctx, r.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.removePhotos(photos)
	return nil
}

// GetStep returns one step if the viewer may see its recipe, and its position in the recipe.
// StepRecipe finds the recipe a step belongs to, if the viewer may see it.
func (s *Service) StepRecipe(ctx context.Context, v Viewer, stepID int64) (Recipe, error) {
	st, err := s.Store.Queries.GetStep(ctx, stepID)
	if isNoRows(err) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}
	return s.GetRecipe(ctx, v, st.RecipeID)
}

func (s *Service) GetStep(ctx context.Context, v Viewer, id int64) (Step, int, error) {
	st, err := s.Store.Queries.GetStep(ctx, id)
	if isNoRows(err) {
		return Step{}, 0, ErrNotFound
	}
	if err != nil {
		return Step{}, 0, err
	}
	doc, err := s.GetRecipe(ctx, v, st.RecipeID)
	if err != nil {
		return Step{}, 0, err
	}
	for i, x := range doc.Steps {
		if x.ID == id {
			return x, i, nil
		}
	}
	return Step{}, 0, ErrNotFound
}
