package app

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/importer"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

var defaultMealTypes = map[string][]string{
	"en": {"Breakfast", "Lunch", "Dinner", "Snack"},
	"de": {"Frühstück", "Mittagessen", "Abendessen", "Snack"},
}

const dateLayout = "2006-01-02"

func toMealType(t sqlcgen.MealType) MealType {
	return MealType{ID: t.ID, Name: t.Name, Position: t.Position, Time: t.Time, Color: t.Color}
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("name_required", "Please enter a name")
	}
	if len(name) > 200 {
		return "", invalid("name_too_long", "This name is too long")
	}
	return name, nil
}

// ListMealTypes gives a household its starting meal types the first time they're needed.
func (s *Service) ListMealTypes(ctx context.Context, v Viewer) ([]MealType, error) {
	rows, err := s.Store.Queries.ListMealTypes(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		names, ok := defaultMealTypes[strings.SplitN(v.locale(), "-", 2)[0]]
		if !ok {
			names = defaultMealTypes["en"]
		}
		err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
			existing, err := q.ListMealTypes(ctx, v.User.HouseholdID)
			if err != nil {
				return err
			}
			if len(existing) > 0 { // someone else in the household got here first
				rows = existing
				return nil
			}
			for i, n := range names {
				t, err := q.CreateMealType(ctx, sqlcgen.CreateMealTypeParams{HouseholdID: v.User.HouseholdID, Name: n, Position: int64(i), CreatedBy: nullID(v.id())})
				if err != nil {
					return err
				}
				rows = append(rows, t)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]MealType, len(rows))
	for i, t := range rows {
		out[i] = toMealType(t)
	}
	return out, nil
}

func (s *Service) mealTypeAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.MealType, error) {
	t, err := q.GetMealType(ctx, id)
	if isNoRows(err) || (err == nil && t.HouseholdID != v.User.HouseholdID) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Service) GetMealType(ctx context.Context, v Viewer, id int64) (MealType, error) {
	t, err := s.mealTypeAccess(ctx, s.Store.Queries, v, id)
	return toMealType(t), err
}

func (s *Service) CreateMealType(ctx context.Context, v Viewer, in MealTypeInput) (MealType, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return MealType{}, err
	}
	var out sqlcgen.MealType
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		out, err = q.CreateMealType(ctx, sqlcgen.CreateMealTypeParams{HouseholdID: v.User.HouseholdID, Name: name, Position: in.Position, Time: in.Time, Color: in.Color, CreatedBy: nullID(v.id())})
		return err
	})
	return toMealType(out), err
}

func (s *Service) UpdateMealType(ctx context.Context, v Viewer, id int64, in MealTypeInput) (MealType, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return MealType{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.mealTypeAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.UpdateMealType(ctx, sqlcgen.UpdateMealTypeParams{Name: name, Position: in.Position, Time: in.Time, Color: in.Color, ID: id})
	})
	if err != nil {
		return MealType{}, err
	}
	return s.GetMealType(ctx, v, id)
}

// DeleteMealType refuses while plans use the type, so deleting it can't remove planned meals.
func (s *Service) DeleteMealType(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.mealTypeAccess(ctx, q, v, id); err != nil {
			return err
		}
		n, err := q.CountMealTypeUses(ctx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return fail(http.StatusConflict, "in_use", "This is still used by planned meals")
		}
		return q.DeleteMealType(ctx, id)
	})
}

func cleanDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) > 10 {
		s = s[:10] // kitshn sends a time of day as well
	}
	if _, err := time.Parse(dateLayout, s); err != nil {
		return "", invalid("bad_date", "This date isn't valid")
	}
	return s, nil
}

func (in *MealPlanInput) clean() error {
	from, err := cleanDate(in.FromDate)
	if err != nil {
		return err
	}
	in.FromDate = from
	if strings.TrimSpace(in.ToDate) == "" {
		in.ToDate = from
	} else if in.ToDate, err = cleanDate(in.ToDate); err != nil {
		return err
	}
	if in.ToDate < in.FromDate {
		return invalid("bad_date", "This date isn't valid")
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Note = strings.TrimSpace(in.Note)
	if in.Title == "" && in.RecipeID == 0 {
		return invalid("plan_empty", "Choose a recipe or enter a title")
	}
	if in.Servings <= 0 {
		in.Servings = 1
	}
	return nil
}

func (s *Service) toMealPlan(ctx context.Context, q *sqlcgen.Queries, v Viewer, p sqlcgen.MealPlan) (MealPlan, error) {
	out := MealPlan{ID: p.ID, Title: p.Title, Note: p.Note, Servings: p.Servings, FromDate: p.FromDate, ToDate: p.ToDate, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
	t, err := q.GetMealType(ctx, p.MealTypeID)
	if err != nil {
		return out, err
	}
	out.MealType = toMealType(t)
	if p.CreatedBy.Valid {
		u, err := q.GetUser(ctx, p.CreatedBy.Int64)
		if err != nil {
			return out, err
		}
		person := toPerson(u)
		out.CreatedBy = &person
	}
	switch {
	case p.RecipeID.Valid:
		r, err := q.GetRecipe(ctx, p.RecipeID.Int64)
		if err != nil {
			return out, err
		}
		ok, err := s.canView(ctx, q, v, r)
		if err != nil {
			return out, err
		}
		if !ok || r.DeletedAt.Valid {
			out.RecipeHidden = true
			break
		}
		items, err := s.summaries(ctx, q, v, []sqlcgen.Recipe{r})
		if err != nil {
			return out, err
		}
		out.Recipe = &items[0]
	case out.Title == "":
		out.Title = p.RecipeName // the recipe is gone for good, and its name is all that's left
	}
	return out, nil
}

func (s *Service) ListMealPlans(ctx context.Context, v Viewer, from, to string, mealType int64) ([]MealPlan, error) {
	q := s.Store.Queries
	rows, err := q.ListMealPlans(ctx, sqlcgen.ListMealPlansParams{HouseholdID: v.User.HouseholdID, FromDate: from, ToDate: to, MealTypeID: mealType})
	if err != nil {
		return nil, err
	}
	out := make([]MealPlan, len(rows))
	for i, p := range rows {
		if out[i], err = s.toMealPlan(ctx, q, v, p); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) mealPlanAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.MealPlan, error) {
	p, err := q.GetMealPlan(ctx, id)
	if isNoRows(err) || (err == nil && p.HouseholdID != v.User.HouseholdID) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Service) GetMealPlan(ctx context.Context, v Viewer, id int64) (MealPlan, error) {
	q := s.Store.Queries
	p, err := s.mealPlanAccess(ctx, q, v, id)
	if err != nil {
		return MealPlan{}, err
	}
	return s.toMealPlan(ctx, q, v, p)
}

// planRecipe checks that a plan may point at a recipe, and returns its id and name.
func (s *Service) planRecipe(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sql.NullInt64, string, error) {
	if id == 0 {
		return sql.NullInt64{}, "", nil
	}
	r, err := q.GetRecipe(ctx, id)
	if isNoRows(err) {
		return sql.NullInt64{}, "", ErrNotFound
	}
	if err != nil {
		return sql.NullInt64{}, "", err
	}
	ok, err := s.canView(ctx, q, v, r)
	if err != nil {
		return sql.NullInt64{}, "", err
	}
	if !ok || r.DeletedAt.Valid {
		return sql.NullInt64{}, "", ErrNotFound
	}
	return sql64(id), r.Name, nil
}

func (s *Service) CreateMealPlan(ctx context.Context, v Viewer, in MealPlanInput) (MealPlan, error) {
	if err := in.clean(); err != nil {
		return MealPlan{}, err
	}
	var id int64
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.mealTypeAccess(ctx, q, v, in.MealTypeID); err != nil {
			return err
		}
		recipe, name, err := s.planRecipe(ctx, q, v, in.RecipeID)
		if err != nil {
			return err
		}
		p, err := q.CreateMealPlan(ctx, sqlcgen.CreateMealPlanParams{HouseholdID: v.User.HouseholdID, CreatedBy: nullID(v.id()), RecipeID: recipe, RecipeName: name, MealTypeID: in.MealTypeID, Title: in.Title, Note: in.Note, Servings: in.Servings, FromDate: in.FromDate, ToDate: in.ToDate})
		if err != nil {
			return err
		}
		id = p.ID
		if in.AddToShopping && recipe.Valid {
			_, err = s.addRecipeToShopping(ctx, q, v, ShoppingAddRecipe{RecipeID: recipe.Int64, Servings: in.Servings, MealPlanID: id})
		}
		return err
	})
	if err != nil {
		return MealPlan{}, err
	}
	return s.GetMealPlan(ctx, v, id)
}

func (s *Service) UpdateMealPlan(ctx context.Context, v Viewer, id int64, in MealPlanInput) (MealPlan, error) {
	if err := in.clean(); err != nil {
		return MealPlan{}, err
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		p, err := s.mealPlanAccess(ctx, q, v, id)
		if err != nil {
			return err
		}
		if _, err := s.mealTypeAccess(ctx, q, v, in.MealTypeID); err != nil {
			return err
		}
		recipe, name := p.RecipeID, p.RecipeName
		// Leaving a recipe in place is always fine, even one this person can't open.
		if !p.RecipeID.Valid || p.RecipeID.Int64 != in.RecipeID {
			if recipe, name, err = s.planRecipe(ctx, q, v, in.RecipeID); err != nil {
				return err
			}
		}
		return q.UpdateMealPlan(ctx, sqlcgen.UpdateMealPlanParams{RecipeID: recipe, RecipeName: name, MealTypeID: in.MealTypeID, Title: in.Title, Note: in.Note, Servings: in.Servings, FromDate: in.FromDate, ToDate: in.ToDate, ID: id})
	})
	if err != nil {
		return MealPlan{}, err
	}
	return s.GetMealPlan(ctx, v, id)
}

func (s *Service) DeleteMealPlan(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.mealPlanAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.DeleteMealPlan(ctx, id)
	})
}

// --- supermarkets ---

func toCategory(c sqlcgen.SupermarketCategory) SupermarketCategory {
	return SupermarketCategory{ID: c.ID, Name: c.Name, Description: c.Description}
}

func (s *Service) ListCategories(ctx context.Context, v Viewer) ([]SupermarketCategory, error) {
	rows, err := s.Store.Queries.ListCategories(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	out := make([]SupermarketCategory, len(rows))
	for i, c := range rows {
		out[i] = toCategory(c)
	}
	return out, nil
}

func (s *Service) categoryAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.SupermarketCategory, error) {
	c, err := q.GetCategory(ctx, id)
	if isNoRows(err) || (err == nil && c.HouseholdID != v.User.HouseholdID) {
		return c, ErrNotFound
	}
	return c, err
}

func (s *Service) GetCategory(ctx context.Context, v Viewer, id int64) (SupermarketCategory, error) {
	c, err := s.categoryAccess(ctx, s.Store.Queries, v, id)
	return toCategory(c), err
}

func (s *Service) CreateCategory(ctx context.Context, v Viewer, in CategoryInput) (SupermarketCategory, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return SupermarketCategory{}, err
	}
	var out sqlcgen.SupermarketCategory
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		out, err = q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{HouseholdID: v.User.HouseholdID, Name: name, Description: in.Description})
		return err
	})
	return toCategory(out), err
}

func (s *Service) UpdateCategory(ctx context.Context, v Viewer, id int64, in CategoryInput) (SupermarketCategory, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return SupermarketCategory{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.categoryAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{Name: name, Description: in.Description, ID: id})
	})
	if err != nil {
		return SupermarketCategory{}, err
	}
	return s.GetCategory(ctx, v, id)
}

func (s *Service) DeleteCategory(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.categoryAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.DeleteCategory(ctx, id)
	})
}

// SetFoodCategory sorts a food into a category, for this household only.
func (s *Service) SetFoodCategory(ctx context.Context, v Viewer, foodID, categoryID int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := q.GetFood(ctx, foodID); isNoRows(err) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if categoryID == 0 {
			return q.ClearFoodCategory(ctx, sqlcgen.ClearFoodCategoryParams{HouseholdID: v.User.HouseholdID, FoodID: foodID})
		}
		if _, err := s.categoryAccess(ctx, q, v, categoryID); err != nil {
			return err
		}
		return q.SetFoodCategory(ctx, sqlcgen.SetFoodCategoryParams{HouseholdID: v.User.HouseholdID, FoodID: foodID, CategoryID: categoryID})
	})
}

// FoodCategories maps food IDs to this household's categories.
func (s *Service) FoodCategories(ctx context.Context, v Viewer) (map[int64]SupermarketCategory, error) {
	return s.foodCategories(ctx, s.Store.Queries, v)
}

func (s *Service) foodCategories(ctx context.Context, q *sqlcgen.Queries, v Viewer) (map[int64]SupermarketCategory, error) {
	links, err := q.ListFoodCategories(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	cats, err := q.ListCategories(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	byID := map[int64]SupermarketCategory{}
	for _, c := range cats {
		byID[c.ID] = toCategory(c)
	}
	out := map[int64]SupermarketCategory{}
	for _, l := range links {
		out[l.FoodID] = byID[l.CategoryID]
	}
	return out, nil
}

func (s *Service) toSupermarket(ctx context.Context, q *sqlcgen.Queries, m sqlcgen.Supermarket) (Supermarket, error) {
	out := Supermarket{ID: m.ID, Name: m.Name, Description: m.Description, Categories: []SupermarketOrderEntry{}}
	rows, err := q.ListSupermarketOrder(ctx, m.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		c, err := q.GetCategory(ctx, r.CategoryID)
		if err != nil {
			return out, err
		}
		out.Categories = append(out.Categories, SupermarketOrderEntry{ID: r.ID, Category: toCategory(c), Position: r.Position})
	}
	return out, nil
}

func (s *Service) ListSupermarkets(ctx context.Context, v Viewer) ([]Supermarket, error) {
	q := s.Store.Queries
	rows, err := q.ListSupermarkets(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	out := make([]Supermarket, len(rows))
	for i, m := range rows {
		if out[i], err = s.toSupermarket(ctx, q, m); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) supermarketAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.Supermarket, error) {
	m, err := q.GetSupermarket(ctx, id)
	if isNoRows(err) || (err == nil && m.HouseholdID != v.User.HouseholdID) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Service) GetSupermarket(ctx context.Context, v Viewer, id int64) (Supermarket, error) {
	q := s.Store.Queries
	m, err := s.supermarketAccess(ctx, q, v, id)
	if err != nil {
		return Supermarket{}, err
	}
	return s.toSupermarket(ctx, q, m)
}

func (s *Service) CreateSupermarket(ctx context.Context, v Viewer, in CategoryInput) (Supermarket, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return Supermarket{}, err
	}
	var id int64
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		m, err := q.CreateSupermarket(ctx, sqlcgen.CreateSupermarketParams{HouseholdID: v.User.HouseholdID, Name: name, Description: in.Description})
		id = m.ID
		return err
	})
	if err != nil {
		return Supermarket{}, err
	}
	return s.GetSupermarket(ctx, v, id)
}

func (s *Service) UpdateSupermarket(ctx context.Context, v Viewer, id int64, in CategoryInput) (Supermarket, error) {
	name, err := cleanName(in.Name)
	if err != nil {
		return Supermarket{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.supermarketAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.UpdateSupermarket(ctx, sqlcgen.UpdateSupermarketParams{Name: name, Description: in.Description, ID: id})
	})
	if err != nil {
		return Supermarket{}, err
	}
	return s.GetSupermarket(ctx, v, id)
}

func (s *Service) DeleteSupermarket(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.supermarketAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.DeleteSupermarket(ctx, id)
	})
}

// SetSupermarketOrder replaces the walking order through a shop.
func (s *Service) SetSupermarketOrder(ctx context.Context, v Viewer, id int64, categories []int64) (Supermarket, error) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.supermarketAccess(ctx, q, v, id); err != nil {
			return err
		}
		if err := q.ClearSupermarketOrder(ctx, id); err != nil {
			return err
		}
		for i, c := range categories {
			if _, err := s.categoryAccess(ctx, q, v, c); err != nil {
				return err
			}
			if _, err := q.AddSupermarketOrder(ctx, sqlcgen.AddSupermarketOrderParams{SupermarketID: id, CategoryID: c, Position: int64(i)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Supermarket{}, err
	}
	return s.GetSupermarket(ctx, v, id)
}

// --- shopping list ---

func (s *Service) ListShopping(ctx context.Context, v Viewer) ([]ShoppingItem, error) {
	return s.shoppingItems(ctx, s.Store.Queries, v, 0)
}

func (s *Service) shoppingItems(ctx context.Context, q *sqlcgen.Queries, v Viewer, only int64) ([]ShoppingItem, error) {
	rows, err := q.ListShoppingItems(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	var foods, units []int64
	for _, r := range rows {
		foods = append(foods, r.FoodID)
		if r.UnitID.Valid {
			units = append(units, r.UnitID.Int64)
		}
	}
	foodMap, err := foodRefs(ctx, q, foods, v.locale())
	if err != nil {
		return nil, err
	}
	unitMap, err := unitRefs(ctx, q, units, v.locale())
	if err != nil {
		return nil, err
	}
	cats, err := s.foodCategories(ctx, q, v)
	if err != nil {
		return nil, err
	}
	srs, err := q.ListShoppingRecipes(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	recipes := map[int64]*ShoppingRecipe{}
	for _, r := range srs {
		recipes[r.ID] = &ShoppingRecipe{ID: r.ID, Name: r.Name, Servings: r.Servings, RecipeID: r.RecipeID.Int64, MealPlanID: r.MealPlanID.Int64}
	}
	people := map[int64]*Person{}
	out := []ShoppingItem{}
	for _, r := range rows {
		if only != 0 && r.ID != only {
			continue
		}
		item := ShoppingItem{ID: r.ID, Food: foodMap[r.FoodID], Note: r.Note, Checked: r.CheckedAt.Valid, Position: r.Position, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
		if r.UnitID.Valid {
			u := unitMap[r.UnitID.Int64]
			item.Unit = &u
		}
		if r.Amount.Valid {
			a := r.Amount.Float64
			item.Amount = &a
		}
		if c, ok := cats[r.FoodID]; ok {
			item.Category = &c
		}
		if r.ShoppingRecipeID.Valid {
			item.Recipe = recipes[r.ShoppingRecipeID.Int64]
		}
		if r.CreatedBy.Valid {
			p, ok := people[r.CreatedBy.Int64]
			if !ok {
				u, err := q.GetUser(ctx, r.CreatedBy.Int64)
				if err != nil {
					return nil, err
				}
				person := toPerson(u)
				p = &person
				people[r.CreatedBy.Int64] = p
			}
			item.CreatedBy = p
		}
		out = append(out, item)
	}
	return out, nil
}

// dropEmptyGroups removes shopping-recipe groups whose last item is gone.
func dropEmptyGroups(ctx context.Context, q *sqlcgen.Queries, household int64) error {
	ids, err := q.ListUnusedShoppingRecipes(ctx, household)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := q.DeleteShoppingRecipe(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkStamp(checked bool) sql.NullString {
	if !checked {
		return sql.NullString{}
	}
	return nullStr(s.stamp(s.Now()))
}

func (s *Service) CreateShoppingItem(ctx context.Context, v Viewer, in ShoppingItemInput) (ShoppingItem, error) {
	return s.createShoppingItem(ctx, v, in, 0)
}

// createShoppingItem also takes the shopping-recipe group an item belongs to, which only
// kitshn sets by hand.
func (s *Service) createShoppingItem(ctx context.Context, v Viewer, in ShoppingItemInput, group int64) (ShoppingItem, error) {
	if line := strings.TrimSpace(in.Line); line != "" {
		units, err := s.unitSpellings(ctx)
		if err != nil {
			return ShoppingItem{}, err
		}
		p := importer.ParseIngredient(line, units)
		in.FoodName, in.UnitName, in.Amount, in.Note = p.Food, p.Unit, p.Amount, p.Note
	}
	if in.FoodID <= 0 && strings.TrimSpace(in.FoodName) == "" {
		return ShoppingItem{}, invalid("ingredient_required", "Each ingredient needs a name")
	}
	if in.Amount != nil && *in.Amount < 0 {
		return ShoppingItem{}, invalid("bad_amount", "Amounts can't be negative")
	}
	var out ShoppingItem
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		ing := IngredientInput{FoodID: in.FoodID, FoodName: in.FoodName, UnitID: in.UnitID, UnitName: in.UnitName}
		food, err := s.resolveFood(ctx, q, v, ing)
		if err != nil {
			return err
		}
		unit, err := s.resolveUnit(ctx, q, v, ing)
		if err != nil {
			return err
		}
		pos, err := q.NextShoppingPosition(ctx, v.User.HouseholdID)
		if err != nil {
			return err
		}
		var amount sql.NullFloat64
		if in.Amount != nil {
			amount = sql.NullFloat64{Float64: *in.Amount, Valid: true}
		}
		var groupID sql.NullInt64
		if group != 0 {
			g, err := q.GetShoppingRecipe(ctx, group)
			if isNoRows(err) || (err == nil && g.HouseholdID != v.User.HouseholdID) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			groupID = sql64(group)
		}
		item, err := q.CreateShoppingItem(ctx, sqlcgen.CreateShoppingItemParams{HouseholdID: v.User.HouseholdID, ShoppingRecipeID: groupID, FoodID: food, UnitID: unit, Amount: amount, Note: strings.TrimSpace(in.Note), Position: pos, CreatedBy: nullID(v.id()), CheckedAt: s.checkStamp(in.Checked)})
		if err != nil {
			return err
		}
		items, err := s.shoppingItems(ctx, q, v, item.ID)
		if err != nil {
			return err
		}
		out = items[0]
		return nil
	})
	return out, err
}

func (s *Service) shoppingAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.ShoppingItem, error) {
	it, err := q.GetShoppingItem(ctx, id)
	if isNoRows(err) || (err == nil && it.HouseholdID != v.User.HouseholdID) {
		return it, ErrNotFound
	}
	return it, err
}

func (s *Service) GetShoppingItem(ctx context.Context, v Viewer, id int64) (ShoppingItem, error) {
	q := s.Store.Queries
	if _, err := s.shoppingAccess(ctx, q, v, id); err != nil {
		return ShoppingItem{}, err
	}
	items, err := s.shoppingItems(ctx, q, v, id)
	if err != nil {
		return ShoppingItem{}, err
	}
	return items[0], nil
}

func (s *Service) UpdateShoppingItem(ctx context.Context, v Viewer, id int64, in ShoppingItemInput) (ShoppingItem, error) {
	if in.Amount != nil && *in.Amount < 0 {
		return ShoppingItem{}, invalid("bad_amount", "Amounts can't be negative")
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		it, err := s.shoppingAccess(ctx, q, v, id)
		if err != nil {
			return err
		}
		ing := IngredientInput{FoodID: in.FoodID, FoodName: in.FoodName, UnitID: in.UnitID, UnitName: in.UnitName}
		food := it.FoodID
		if in.FoodID > 0 || strings.TrimSpace(in.FoodName) != "" {
			if food, err = s.resolveFood(ctx, q, v, ing); err != nil {
				return err
			}
		}
		unit, err := s.resolveUnit(ctx, q, v, ing)
		if err != nil {
			return err
		}
		var amount sql.NullFloat64
		if in.Amount != nil {
			amount = sql.NullFloat64{Float64: *in.Amount, Valid: true}
		}
		checked := it.CheckedAt
		if in.Checked != it.CheckedAt.Valid {
			checked = s.checkStamp(in.Checked)
		}
		return q.UpdateShoppingItem(ctx, sqlcgen.UpdateShoppingItemParams{FoodID: food, UnitID: unit, Amount: amount, Note: strings.TrimSpace(in.Note), Position: it.Position, CheckedAt: checked, ID: id})
	})
	if err != nil {
		return ShoppingItem{}, err
	}
	return s.GetShoppingItem(ctx, v, id)
}

func (s *Service) DeleteShoppingItem(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.shoppingAccess(ctx, q, v, id); err != nil {
			return err
		}
		if err := q.DeleteShoppingItem(ctx, id); err != nil {
			return err
		}
		return dropEmptyGroups(ctx, q, v.User.HouseholdID)
	})
}

// CheckShopping ticks or unticks items in one go. IDs from other households do nothing.
func (s *Service) CheckShopping(ctx context.Context, v Viewer, in ShoppingCheckInput) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		stamp := s.checkStamp(in.Checked)
		for _, id := range in.IDs {
			if err := q.SetShoppingItemChecked(ctx, sqlcgen.SetShoppingItemCheckedParams{CheckedAt: stamp, ID: id, HouseholdID: v.User.HouseholdID}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ClearCheckedShopping(ctx context.Context, v Viewer) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.DeleteCheckedShoppingItems(ctx, v.User.HouseholdID); err != nil {
			return err
		}
		return dropEmptyGroups(ctx, q, v.User.HouseholdID)
	})
}

func (s *Service) AddRecipeToShopping(ctx context.Context, v Viewer, in ShoppingAddRecipe) (ShoppingRecipe, error) {
	var out ShoppingRecipe
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		var err error
		out, err = s.addRecipeToShopping(ctx, q, v, in)
		return err
	})
	return out, err
}

func (s *Service) addRecipeToShopping(ctx context.Context, q *sqlcgen.Queries, v Viewer, in ShoppingAddRecipe) (ShoppingRecipe, error) {
	r, err := q.GetRecipe(ctx, in.RecipeID)
	if isNoRows(err) {
		return ShoppingRecipe{}, ErrNotFound
	}
	if err != nil {
		return ShoppingRecipe{}, err
	}
	ok, err := s.canView(ctx, q, v, r)
	if err != nil {
		return ShoppingRecipe{}, err
	}
	if !ok || r.DeletedAt.Valid {
		return ShoppingRecipe{}, ErrNotFound
	}
	servings := in.Servings
	if servings <= 0 {
		servings = float64(r.Servings)
	}
	var plan sql.NullInt64
	if in.MealPlanID != 0 {
		if _, err := s.mealPlanAccess(ctx, q, v, in.MealPlanID); err != nil {
			return ShoppingRecipe{}, err
		}
		plan = sql64(in.MealPlanID)
	}
	group, err := q.CreateShoppingRecipe(ctx, sqlcgen.CreateShoppingRecipeParams{HouseholdID: v.User.HouseholdID, RecipeID: sql64(r.ID), MealPlanID: plan, Name: r.Name, Servings: servings, CreatedBy: nullID(v.id())})
	if err != nil {
		return ShoppingRecipe{}, err
	}
	wanted := map[int64]bool{}
	for _, id := range in.IngredientIDs {
		wanted[id] = true
	}
	ings, err := q.ListIngredientsForRecipe(ctx, r.ID)
	if err != nil {
		return ShoppingRecipe{}, err
	}
	factor := 1.0
	if r.Servings > 0 {
		factor = servings / float64(r.Servings)
	}
	pos, err := q.NextShoppingPosition(ctx, v.User.HouseholdID)
	if err != nil {
		return ShoppingRecipe{}, err
	}
	for _, i := range ings {
		if len(wanted) > 0 && !wanted[i.ID] {
			continue
		}
		amount := i.Amount
		if amount.Valid {
			amount.Float64 *= factor
		}
		if _, err := q.CreateShoppingItem(ctx, sqlcgen.CreateShoppingItemParams{HouseholdID: v.User.HouseholdID, ShoppingRecipeID: sql64(group.ID), FoodID: i.FoodID, UnitID: i.UnitID, Amount: amount, Note: i.Note, Position: pos, CreatedBy: nullID(v.id())}); err != nil {
			return ShoppingRecipe{}, err
		}
		pos++
	}
	return ShoppingRecipe{ID: group.ID, Name: group.Name, Servings: group.Servings, RecipeID: r.ID, MealPlanID: plan.Int64}, nil
}

// CreateShoppingItemFor adds an item to a group of a recipe, as kitshn's list_recipe does.
func (s *Service) CreateShoppingItemFor(ctx context.Context, v Viewer, in ShoppingItemInput, group int64) (ShoppingItem, error) {
	return s.createShoppingItem(ctx, v, in, group)
}

func (s *Service) DeleteShoppingGroup(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		g, err := q.GetShoppingRecipe(ctx, id)
		if isNoRows(err) || (err == nil && g.HouseholdID != v.User.HouseholdID) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := q.DeleteShoppingItemsOfRecipe(ctx, sql64(id)); err != nil {
			return err
		}
		return q.DeleteShoppingRecipe(ctx, id)
	})
}

// RecipeNutritionFor is GetRecipeNutrition for a recipe already loaded.
func (s *Service) RecipeNutritionFor(ctx context.Context, locale string, rec Recipe) (RecipeNutrition, error) {
	return s.recipeNutrition(ctx, s.Store.Queries, locale, rec)
}

// CreateShoppingGroup and the methods below serve kitshn, which builds a recipe's shopping
// entries in two calls.
func (s *Service) CreateShoppingGroup(ctx context.Context, v Viewer, recipeID int64, servings float64) (ShoppingRecipe, error) {
	var out ShoppingRecipe
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		r, err := q.GetRecipe(ctx, recipeID)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if ok, err := s.canView(ctx, q, v, r); err != nil {
			return err
		} else if !ok || r.DeletedAt.Valid {
			return ErrNotFound
		}
		g, err := q.CreateShoppingRecipe(ctx, sqlcgen.CreateShoppingRecipeParams{HouseholdID: v.User.HouseholdID, RecipeID: sql64(recipeID), Name: r.Name, Servings: servings, CreatedBy: nullID(v.id())})
		if err != nil {
			return err
		}
		out = ShoppingRecipe{ID: g.ID, Name: g.Name, Servings: g.Servings, RecipeID: recipeID}
		return nil
	})
	return out, err
}

// CreateShoppingItemsInGroup adds several items to a group in one transaction.
func (s *Service) CreateShoppingItemsInGroup(ctx context.Context, v Viewer, group int64, items []ShoppingItemInput) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		g, err := q.GetShoppingRecipe(ctx, group)
		if isNoRows(err) || (err == nil && g.HouseholdID != v.User.HouseholdID) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		pos, err := q.NextShoppingPosition(ctx, v.User.HouseholdID)
		if err != nil {
			return err
		}
		for _, in := range items {
			ing := IngredientInput{FoodID: in.FoodID, FoodName: in.FoodName, UnitID: in.UnitID, UnitName: in.UnitName}
			food, err := s.resolveFood(ctx, q, v, ing)
			if err != nil {
				return err
			}
			unit, err := s.resolveUnit(ctx, q, v, ing)
			if err != nil {
				return err
			}
			var amount sql.NullFloat64
			if in.Amount != nil {
				amount = sql.NullFloat64{Float64: *in.Amount, Valid: true}
			}
			if _, err := q.CreateShoppingItem(ctx, sqlcgen.CreateShoppingItemParams{HouseholdID: v.User.HouseholdID, ShoppingRecipeID: sql64(group), FoodID: food, UnitID: unit, Amount: amount, Note: in.Note, Position: pos, CreatedBy: nullID(v.id())}); err != nil {
				return err
			}
			pos++
		}
		return nil
	})
}

func (s *Service) GetShoppingGroup(ctx context.Context, v Viewer, id int64) (ShoppingRecipe, error) {
	g, err := s.Store.Queries.GetShoppingRecipe(ctx, id)
	if isNoRows(err) || (err == nil && g.HouseholdID != v.User.HouseholdID) {
		return ShoppingRecipe{}, ErrNotFound
	}
	if err != nil {
		return ShoppingRecipe{}, err
	}
	return ShoppingRecipe{ID: g.ID, Name: g.Name, Servings: g.Servings, RecipeID: g.RecipeID.Int64, MealPlanID: g.MealPlanID.Int64}, nil
}

// --- nutrition ---

func (s *Service) ListNutrients(ctx context.Context, v Viewer) ([]Nutrient, error) {
	return s.nutrients(ctx, s.Store.Queries, v.locale())
}

func (s *Service) nutrients(ctx context.Context, q *sqlcgen.Queries, locale string) ([]Nutrient, error) {
	rows, err := q.ListNutrients(ctx, locale)
	if err != nil {
		return nil, err
	}
	out := make([]Nutrient, len(rows))
	for i, n := range rows {
		out[i] = Nutrient{ID: n.ID, Name: n.Name, Unit: n.Unit, Position: n.Position}
	}
	return out, nil
}

func (s *Service) GetFoodNutrition(ctx context.Context, id int64) (FoodNutrition, error) {
	q := s.Store.Queries
	f, err := q.GetFood(ctx, id)
	if isNoRows(err) {
		return FoodNutrition{}, ErrNotFound
	}
	if err != nil {
		return FoodNutrition{}, err
	}
	out := FoodNutrition{Values: map[int64]float64{}}
	if f.PieceGrams.Valid {
		g := f.PieceGrams.Float64
		out.PieceGrams = &g
	}
	rows, err := q.ListFoodNutrients(ctx, []int64{id})
	if err != nil {
		return FoodNutrition{}, err
	}
	for _, r := range rows {
		out.Values[r.NutrientID] = r.Per100g
	}
	return out, nil
}

func (s *Service) SetFoodNutrition(ctx context.Context, v Viewer, id int64, in FoodNutrition) (FoodNutrition, error) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		f, err := q.GetFood(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !s.canEditCatalog(v, f.CreatedBy.Int64) {
			return ErrForbidden
		}
		var piece sql.NullFloat64
		if in.PieceGrams != nil {
			if *in.PieceGrams <= 0 {
				return invalid("bad_amount", "Amounts can't be negative")
			}
			piece = sql.NullFloat64{Float64: *in.PieceGrams, Valid: true}
		}
		if err := q.SetFoodPieceGrams(ctx, sqlcgen.SetFoodPieceGramsParams{PieceGrams: piece, ID: id}); err != nil {
			return err
		}
		if err := q.ClearFoodNutrients(ctx, id); err != nil {
			return err
		}
		for n, value := range in.Values {
			if value < 0 {
				return invalid("bad_amount", "Amounts can't be negative")
			}
			if err := q.SetFoodNutrient(ctx, sqlcgen.SetFoodNutrientParams{FoodID: id, NutrientID: n, Per100g: value}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return FoodNutrition{}, err
	}
	return s.GetFoodNutrition(ctx, id)
}

// pieceUnitID is the catalog's "piece". Having no unit at all counts as pieces too.
const pieceUnitID = 13

// GetRecipeNutrition adds up a recipe from what's known about its ingredients.
func (s *Service) GetRecipeNutrition(ctx context.Context, v Viewer, id int64) (RecipeNutrition, error) {
	rec, err := s.GetRecipe(ctx, v, id)
	if err != nil {
		return RecipeNutrition{}, err
	}
	return s.recipeNutrition(ctx, s.Store.Queries, v.locale(), rec)
}

func (s *Service) recipeNutrition(ctx context.Context, q *sqlcgen.Queries, locale string, rec Recipe) (RecipeNutrition, error) {
	nutrients, err := s.nutrients(ctx, q, locale)
	if err != nil {
		return RecipeNutrition{}, err
	}
	var foods, units []int64
	for _, st := range rec.Steps {
		for _, i := range st.Ingredients {
			foods = append(foods, i.Food.ID)
			if i.Unit != nil {
				units = append(units, i.Unit.ID)
			}
		}
	}
	out := RecipeNutrition{Nutrients: []RecipeNutrient{}, Missing: []string{}}
	totals := map[int64]float64{}
	if len(foods) > 0 {
		values := map[int64]map[int64]float64{}
		rows, err := q.ListFoodNutrients(ctx, foods)
		if err != nil {
			return out, err
		}
		for _, r := range rows {
			if values[r.FoodID] == nil {
				values[r.FoodID] = map[int64]float64{}
			}
			values[r.FoodID][r.NutrientID] = r.Per100g
		}
		pieces, err := q.ListFoodPieceGrams(ctx, foods)
		if err != nil {
			return out, err
		}
		pieceGrams := map[int64]float64{}
		for _, p := range pieces {
			if p.PieceGrams.Valid {
				pieceGrams[p.ID] = p.PieceGrams.Float64
			}
		}
		unitGrams := map[int64]float64{}
		if len(units) > 0 {
			rows, err := q.ListUnitGrams(ctx, units)
			if err != nil {
				return out, err
			}
			for _, u := range rows {
				if u.Grams.Valid {
					unitGrams[u.ID] = u.Grams.Float64
				}
			}
		}
		for _, st := range rec.Steps {
			for _, i := range st.Ingredients {
				out.Total++
				var grams float64
				known := i.Amount != nil && values[i.Food.ID] != nil
				if known {
					switch {
					case i.Unit != nil && unitGrams[i.Unit.ID] > 0:
						grams = *i.Amount * unitGrams[i.Unit.ID]
					case (i.Unit == nil || i.Unit.ID == pieceUnitID) && pieceGrams[i.Food.ID] > 0:
						grams = *i.Amount * pieceGrams[i.Food.ID]
					default:
						known = false
					}
				}
				if !known {
					out.Missing = append(out.Missing, i.Food.Name)
					continue
				}
				out.Counted++
				for n, per100 := range values[i.Food.ID] {
					totals[n] += per100 * grams / 100
				}
			}
		}
	}
	servings := float64(rec.Servings)
	if servings <= 0 {
		servings = 1
	}
	for _, n := range nutrients {
		out.Nutrients = append(out.Nutrients, RecipeNutrient{Nutrient: n, Total: totals[n.ID], PerServing: totals[n.ID] / servings})
	}
	sort.Strings(out.Missing)
	return out, nil
}
