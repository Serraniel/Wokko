package kitshn

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/media"
)

// writeRoutes covers everything kitshn changes and the planning screens. Every call goes
// through package app, so it gets the same transactions, permissions and trash as the web app.
func (l *Layer) writeRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/recipe/{$}", l.authed(l.createRecipe))
	m.HandleFunc("PATCH /api/recipe/{id}/{$}", l.authed(l.patchRecipe))
	m.HandleFunc("DELETE /api/recipe/{id}/{$}", l.authed(l.deleteRecipe))
	m.HandleFunc("PUT /api/recipe/{id}/image/{$}", l.authed(l.recipeImage))
	m.HandleFunc("PUT /api/recipe/{id}/shopping/{$}", l.authed(l.recipeShopping))
	m.HandleFunc("GET /api/share-link/{id}", l.authed(l.shareLink))
	m.HandleFunc("GET /api/share-link/{id}/{$}", l.authed(l.shareLink))
	m.HandleFunc("PATCH /api/step/{id}/{$}", l.authed(l.changeStep))
	m.HandleFunc("PUT /api/step/{id}/{$}", l.authed(l.changeStep))
	m.HandleFunc("DELETE /api/step/{id}/{$}", l.authed(l.deleteStep))
	m.HandleFunc("POST /api/recipe-from-source/{$}", l.authed(l.fromSource))

	m.HandleFunc("POST /api/food/{$}", l.authed(l.createFood))
	m.HandleFunc("PATCH /api/food/{id}/{$}", l.authed(l.patchFood))
	m.HandleFunc("DELETE /api/food/{id}/{$}", l.authed(l.deleteFood))
	m.HandleFunc("POST /api/unit/{$}", l.authed(l.createUnit))
	m.HandleFunc("DELETE /api/unit/{id}/{$}", l.authed(l.deleteUnit))
	m.HandleFunc("POST /api/keyword/{$}", l.authed(l.createKeyword))

	m.HandleFunc("GET /api/meal-type/{$}", l.authed(l.listMealTypes))
	m.HandleFunc("POST /api/meal-type/{$}", l.authed(l.createMealType))
	m.HandleFunc("PATCH /api/meal-type/{id}/{$}", l.authed(l.patchMealType))
	m.HandleFunc("DELETE /api/meal-type/{id}/{$}", l.authed(l.deleteMealType))
	m.HandleFunc("GET /api/meal-plan/{$}", l.authed(l.listMealPlans))
	m.HandleFunc("POST /api/meal-plan/{$}", l.authed(l.createMealPlan))
	m.HandleFunc("GET /api/meal-plan/{id}/{$}", l.authed(l.getMealPlan))
	m.HandleFunc("PATCH /api/meal-plan/{id}/{$}", l.authed(l.patchMealPlan))
	m.HandleFunc("DELETE /api/meal-plan/{id}/{$}", l.authed(l.deleteMealPlan))

	m.HandleFunc("GET /api/recipe-book/{$}", l.authed(l.listBooks))
	m.HandleFunc("POST /api/recipe-book/{$}", l.authed(l.createBook))
	m.HandleFunc("GET /api/recipe-book/{id}/{$}", l.authed(l.getBook))
	m.HandleFunc("PATCH /api/recipe-book/{id}/{$}", l.authed(l.patchBook))
	m.HandleFunc("DELETE /api/recipe-book/{id}/{$}", l.authed(l.deleteBook))
	m.HandleFunc("GET /api/recipe-book-entry/{$}", l.authed(l.listBookEntries))
	m.HandleFunc("POST /api/recipe-book-entry/{$}", l.authed(l.createBookEntry))
	m.HandleFunc("DELETE /api/recipe-book-entry/{id}/{$}", l.authed(l.deleteBookEntry))

	m.HandleFunc("GET /api/shopping-list-entry/{$}", l.authed(l.listShopping))
	m.HandleFunc("POST /api/shopping-list-entry/{$}", l.authed(l.createShopping))
	m.HandleFunc("POST /api/shopping-list-entry/bulk/{$}", l.authed(l.bulkShopping))
	m.HandleFunc("PATCH /api/shopping-list-entry/{id}/{$}", l.authed(l.patchShopping))
	m.HandleFunc("DELETE /api/shopping-list-entry/{id}/{$}", l.authed(l.deleteShopping))
	m.HandleFunc("POST /api/shopping-list-recipe/{$}", l.authed(l.createShoppingRecipe))
	m.HandleFunc("POST /api/shopping-list-recipe/{id}/bulk_create_entries/{$}", l.authed(l.bulkShoppingRecipe))
	m.HandleFunc("DELETE /api/shopping-list-recipe/{id}/{$}", l.authed(l.deleteShoppingRecipe))

	m.HandleFunc("GET /api/supermarket/{$}", l.authed(l.listSupermarkets))
	m.HandleFunc("POST /api/supermarket/{$}", l.authed(l.createSupermarket))
	m.HandleFunc("GET /api/supermarket/{id}/{$}", l.authed(l.getSupermarket))
	m.HandleFunc("DELETE /api/supermarket/{id}/{$}", l.authed(l.deleteSupermarket))
	m.HandleFunc("GET /api/supermarket-category/{$}", l.authed(l.listCategories))
	m.HandleFunc("POST /api/supermarket-category/{$}", l.authed(l.createCategory))
	m.HandleFunc("GET /api/supermarket-category/{id}/{$}", l.authed(l.getCategory))
	m.HandleFunc("DELETE /api/supermarket-category/{id}/{$}", l.authed(l.deleteCategory))
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(v); err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "bad_request", Message: "The request could not be read"}
	}
	return nil
}

func noContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func idList(q url.Values, names ...string) []int64 {
	var out []int64
	for _, name := range names {
		for _, raw := range q[name] {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				out = append(out, id)
			}
		}
	}
	return out
}

// pageOf cuts a complete list down to the page kitshn asked for.
func pageOf[T any](r *http.Request, items []T) paged[T] {
	p := readPage(r)
	start := min(p.offset(), int64(len(items)))
	end := min(start+p.size, int64(len(items)))
	return toPaged(r, p, int64(len(items)), items[start:end])
}

// --- nutrition, as Tandoor's properties ---

type propertyType struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
	Order       int64  `json:"order"`
}

type property struct {
	ID             int64        `json:"id"`
	PropertyAmount float64      `json:"property_amount"`
	PropertyType   propertyType `json:"property_type"`
}

type foodProperty struct {
	ID           int64          `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Unit         string         `json:"unit"`
	Order        int64          `json:"order"`
	FoodValues   map[string]any `json:"food_values"`
	TotalValue   float64        `json:"total_value"`
	MissingValue bool           `json:"missing_value"`
}

func (l *Layer) fullRecipe(r *http.Request, v app.Viewer, rec app.Recipe) recipe {
	out := toRecipe(rec)
	n, err := l.svc.RecipeNutritionFor(r.Context(), v.Locale, rec)
	if err != nil {
		l.log.Warn("could not add nutrition for kitshn", "recipe", rec.ID, "error", err)
		return out
	}
	if n.Counted == 0 {
		return out
	}
	for _, x := range n.Nutrients {
		out.FoodProperties[strconv.FormatInt(x.Nutrient.ID, 10)] = foodProperty{
			ID: x.Nutrient.ID, Name: x.Nutrient.Name, Unit: x.Nutrient.Unit, Order: x.Nutrient.Position,
			FoodValues: map[string]any{}, TotalValue: x.Total, MissingValue: n.Counted < n.Total,
		}
	}
	return out
}

func (l *Layer) foodFor(r *http.Request, v app.Viewer, f app.FoodRef) food {
	out := toFood(f)
	cats, err := l.svc.FoodCategories(r.Context(), v)
	if err != nil {
		l.log.Warn("could not add a shopping category for kitshn", "error", err)
		return out
	}
	if c, ok := cats[f.ID]; ok {
		cat := toCategoryOut(c)
		out.SupermarketCategory = &cat
	}
	return out
}

// --- recipes ---

type namedRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ingredientIn struct {
	Food     *namedRef `json:"food"`
	Unit     *namedRef `json:"unit"`
	Amount   *float64  `json:"amount"`
	Note     *string   `json:"note"`
	NoAmount bool      `json:"no_amount"`
}

type stepIn struct {
	Name        *string         `json:"name"`
	Instruction *string         `json:"instruction"`
	Time        *int64          `json:"time"`
	Ingredients *[]ingredientIn `json:"ingredients"`
}

type recipeIn struct {
	Name         *string     `json:"name"`
	Description  *string     `json:"description"`
	Keywords     *[]namedRef `json:"keywords"`
	WorkingTime  *int64      `json:"working_time"`
	WaitingTime  *int64      `json:"waiting_time"`
	SourceURL    *string     `json:"source_url"`
	Servings     *int64      `json:"servings"`
	ServingsText *string     `json:"servings_text"`
	Steps        *[]stepIn   `json:"steps"`
	Image        *string     `json:"image"`
	ImageURL     *string     `json:"image_url"`
}

func set[T any](target *T, value *T) {
	if value != nil {
		*target = *value
	}
}

func toIngredients(in []ingredientIn) []app.IngredientInput {
	out := []app.IngredientInput{}
	for _, i := range in {
		if i.Food == nil || (i.Food.ID <= 0 && strings.TrimSpace(i.Food.Name) == "") {
			continue // Tandoor's section headers have no food, and Wokko has no such rows.
		}
		item := app.IngredientInput{FoodName: i.Food.Name}
		if i.Food.ID > 0 {
			item.FoodID = i.Food.ID
		}
		if i.Unit != nil {
			item.UnitName = i.Unit.Name
			if i.Unit.ID > 0 {
				item.UnitID = i.Unit.ID
			}
		}
		set(&item.Note, i.Note)
		// Tandoor writes "no amount" as a flag with 0 beside it.
		if i.Amount != nil && !i.NoAmount && *i.Amount != 0 {
			item.Amount = i.Amount
		}
		out = append(out, item)
	}
	return out
}

func applySteps(in []stepIn) []app.StepInput {
	out := make([]app.StepInput, 0, len(in))
	for _, st := range in {
		item := app.StepInput{Ingredients: []app.IngredientInput{}}
		set(&item.Name, st.Name)
		set(&item.Instruction, st.Instruction)
		set(&item.Minutes, st.Time)
		if st.Ingredients != nil {
			item.Ingredients = toIngredients(*st.Ingredients)
		}
		out = append(out, item)
	}
	return out
}

func inputFrom(rec app.Recipe) app.RecipeInput {
	in := app.RecipeInput{
		Name: rec.Name, Description: rec.Description, WorkMinutes: rec.WorkMinutes, WaitMinutes: rec.WaitMinutes,
		Servings: rec.Servings, ServingsText: rec.ServingsText, SourceURL: rec.SourceURL, Visibility: rec.Visibility,
		BaseVersion: rec.Version, Keywords: []string{}, Steps: []app.StepInput{},
	}
	for _, k := range rec.Keywords {
		in.Keywords = append(in.Keywords, k.Name)
	}
	for _, st := range rec.Steps {
		in.Steps = append(in.Steps, stepInput(st))
	}
	return in
}

func stepInput(st app.Step) app.StepInput {
	out := app.StepInput{Name: st.Name, Instruction: st.Instruction, Minutes: st.Minutes, Ingredients: []app.IngredientInput{}}
	for _, i := range st.Ingredients {
		item := app.IngredientInput{FoodID: i.Food.ID, Amount: i.Amount, Note: i.Note}
		if i.Unit != nil {
			item.UnitID = i.Unit.ID
		}
		out.Ingredients = append(out.Ingredients, item)
	}
	return out
}

func (in recipeIn) applyTo(out *app.RecipeInput) {
	set(&out.Name, in.Name)
	set(&out.Description, in.Description)
	set(&out.WorkMinutes, in.WorkingTime)
	set(&out.WaitMinutes, in.WaitingTime)
	set(&out.SourceURL, in.SourceURL)
	set(&out.Servings, in.Servings)
	set(&out.ServingsText, in.ServingsText)
	if in.Keywords != nil {
		out.Keywords = []string{}
		for _, k := range *in.Keywords {
			out.Keywords = append(out.Keywords, k.Name)
		}
	}
	if in.Steps != nil {
		out.Steps = applySteps(*in.Steps)
	}
}

func (l *Layer) createRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in recipeIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	draft := app.RecipeInput{Visibility: "household", Servings: 1, Keywords: []string{}, Steps: []app.StepInput{}}
	in.applyTo(&draft)
	rec, err := l.svc.CreateRecipe(r.Context(), v, draft)
	if err != nil {
		return err
	}
	// A failed photo download shouldn't lose the recipe someone just imported.
	image := ""
	if in.Image != nil && strings.HasPrefix(*in.Image, "http") {
		image = *in.Image
	} else if in.ImageURL != nil && strings.HasPrefix(*in.ImageURL, "http") {
		image = *in.ImageURL
	}
	if image != "" {
		if withPhoto, err := l.svc.SetPhotoFromURL(r.Context(), v, rec.ID, image); err == nil {
			rec = withPhoto
		} else {
			l.log.Info("kitshn import saved without its photo", "error", err)
		}
	}
	writeJSON(w, http.StatusCreated, l.fullRecipe(r, v, rec))
	return nil
}

// patchRecipe changes only what was sent. kitshn doesn't track versions, so it edits the
// latest one. The version it replaces stays in the recipe's history.
func (l *Layer) patchRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in recipeIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	rec, err := l.svc.GetRecipe(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := inputFrom(rec)
	in.applyTo(&draft)
	saved, err := l.svc.UpdateRecipe(r.Context(), v, id, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l.fullRecipe(r, v, saved))
	return nil
}

func (l *Layer) deleteRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteRecipe(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) recipeImage(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "invalid_image", Message: "Please choose a photo no larger than 20 MB"}
	}
	var rec app.Recipe
	if link := r.FormValue("image_url"); link != "" {
		rec, err = l.svc.SetPhotoFromURL(r.Context(), v, id, link)
	} else {
		file, _, ferr := r.FormFile("image")
		if ferr != nil {
			return &app.Error{Status: http.StatusBadRequest, Code: "invalid_image", Message: "Please choose a photo no larger than 20 MB"}
		}
		defer file.Close()
		rec, err = l.svc.SetPhoto(r.Context(), v, id, file)
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"image": rec.PhotoURL})
	return nil
}

func (l *Layer) recipeShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		Ingredients []int64   `json:"ingredients"`
		Mealplan    *namedRef `json:"mealplan"`
		Servings    float64   `json:"servings"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	add := app.ShoppingAddRecipe{RecipeID: id, Servings: in.Servings, IngredientIDs: in.Ingredients}
	if in.Mealplan != nil {
		add.MealPlanID = in.Mealplan.ID
	}
	if _, err := l.svc.AddRecipeToShopping(r.Context(), v, add); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) shareLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	link, err := l.svc.CreateShareLink(r.Context(), v, id, 0)
	if err != nil {
		return err
	}
	address := link.URL
	if strings.HasPrefix(address, "/") {
		address = scheme(r) + "://" + r.Host + address
	}
	writeJSON(w, http.StatusOK, map[string]any{"pk": id, "share": link.ID, "link": address})
	return nil
}

func (l *Layer) changeStep(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in stepIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	rec, err := l.svc.StepRecipe(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := inputFrom(rec)
	index := slices.IndexFunc(rec.Steps, func(st app.Step) bool { return st.ID == id })
	if index < 0 {
		return app.ErrNotFound
	}
	target := &draft.Steps[index]
	set(&target.Name, in.Name)
	set(&target.Instruction, in.Instruction)
	set(&target.Minutes, in.Time)
	if in.Ingredients != nil {
		target.Ingredients = toIngredients(*in.Ingredients)
	}
	if _, err := l.svc.UpdateRecipe(r.Context(), v, rec.ID, draft); err != nil {
		return err
	}
	st, order, err := l.svc.GetStep(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toStep(st, order))
	return nil
}

func (l *Layer) deleteStep(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	rec, err := l.svc.StepRecipe(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := inputFrom(rec)
	index := slices.IndexFunc(rec.Steps, func(st app.Step) bool { return st.ID == id })
	if index < 0 {
		return app.ErrNotFound
	}
	// Later steps move up, each taking the ID of the position it lands on.
	draft.Steps = slices.Delete(draft.Steps, index, index+1)
	if _, err := l.svc.UpdateRecipe(r.Context(), v, rec.ID, draft); err != nil {
		return err
	}
	return noContent(w)
}

// fromSource answers with error:true when it can't import, which is how kitshn expects a
// failed import to look.
func (l *Layer) fromSource(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		URL  string `json:"url"`
		Data string `json:"data"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	draft, image, err := l.svc.ImportForKitshn(r.Context(), in.URL)
	if err != nil {
		var msg = "This recipe could not be imported"
		if ae, ok := err.(*app.Error); ok {
			msg = ae.Message
		} else {
			l.log.Error("kitshn import failed", "error", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{"error": true, "msg": msg, "images": []string{}, "duplicates": []any{}})
		return nil
	}
	steps := []map[string]any{}
	for _, st := range draft.Recipe.Steps {
		ings := []map[string]any{}
		for _, i := range st.Ingredients {
			amount := 0.0
			if i.Amount != nil {
				amount = *i.Amount
			}
			var unitOut any
			if i.UnitName != "" {
				unitOut = map[string]string{"name": i.UnitName}
			}
			ings = append(ings, map[string]any{"amount": amount, "food": map[string]string{"name": i.FoodName}, "unit": unitOut, "note": i.Note, "original_text": strings.TrimSpace(i.UnitName + " " + i.FoodName)})
		}
		steps = append(steps, map[string]any{"instruction": st.Instruction, "ingredients": ings, "show_ingredients_table": true})
	}
	keywords := []map[string]string{}
	for _, k := range draft.Recipe.Keywords {
		keywords = append(keywords, map[string]string{"label": k, "name": k})
	}
	images := []string{}
	if image != "" {
		images = append(images, image)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recipe": map[string]any{
			"steps": steps, "internal": true, "source_url": draft.Recipe.SourceURL, "name": draft.Recipe.Name, "description": draft.Recipe.Description,
			"servings": draft.Recipe.Servings, "servings_text": "", "working_time": draft.Recipe.WorkMinutes, "waiting_time": draft.Recipe.WaitMinutes,
			"image_url": image, "keywords": keywords,
		},
		"recipe_id": nil, "images": images, "error": false, "msg": "", "duplicates": []any{},
	})
	return nil
}

// --- catalog ---

func (l *Layer) createFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in namedRef
	if err := decode(w, r, &in); err != nil {
		return err
	}
	f, err := l.svc.CreateFood(r.Context(), v, app.FoodInput{Translations: []app.Translation{{Locale: v.Locale, Name: in.Name}}})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, l.foodFor(r, v, f.FoodRef))
	return nil
}

// patchFood only changes the shopping category. The shared catalog's names are managed in
// the web app, where every language can be seen at once.
func (l *Layer) patchFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		SupermarketCategory *namedRef `json:"supermarket_category"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	var category int64
	if in.SupermarketCategory != nil {
		category = in.SupermarketCategory.ID
	}
	if err := l.svc.SetFoodCategory(r.Context(), v, id, category); err != nil {
		return err
	}
	f, err := l.svc.GetFood(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l.foodFor(r, v, f.FoodRef))
	return nil
}

func (l *Layer) deleteFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteFood(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) createUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in namedRef
	if err := decode(w, r, &in); err != nil {
		return err
	}
	u, err := l.svc.CreateUnit(r.Context(), v, app.UnitInput{Translations: []app.Translation{{Locale: v.Locale, Name: in.Name}}})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toUnit(u.UnitRef))
	return nil
}

func (l *Layer) deleteUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteUnit(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) createKeyword(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in namedRef
	if err := decode(w, r, &in); err != nil {
		return err
	}
	k, err := l.svc.CreateKeyword(r.Context(), v, in.Name)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toKeyword(k))
	return nil
}

// --- meal plans ---

type mealType struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Order     int64   `json:"order"`
	Time      *string `json:"time"`
	Color     *string `json:"color"`
	CreatedBy int64   `json:"created_by"`
}

func toMealType(t app.MealType, v app.Viewer) mealType {
	return mealType{ID: t.ID, Name: t.Name, Order: t.Position, Time: strPtr(t.Time), Color: strPtr(t.Color), CreatedBy: v.User.ID}
}

func (l *Layer) listMealTypes(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.ListMealTypes(r.Context(), v)
	if err != nil {
		return err
	}
	out := make([]mealType, len(rows))
	for i, t := range rows {
		out[i] = toMealType(t, v)
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

type mealTypeIn struct {
	Name  *string `json:"name"`
	Order *int64  `json:"order"`
	Time  *string `json:"time"`
	Color *string `json:"color"`
}

func (in mealTypeIn) applyTo(out *app.MealTypeInput) {
	set(&out.Name, in.Name)
	set(&out.Position, in.Order)
	set(&out.Time, in.Time)
	set(&out.Color, in.Color)
}

func (l *Layer) createMealType(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in mealTypeIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	var draft app.MealTypeInput
	in.applyTo(&draft)
	t, err := l.svc.CreateMealType(r.Context(), v, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toMealType(t, v))
	return nil
}

func (l *Layer) patchMealType(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in mealTypeIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	cur, err := l.svc.GetMealType(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := app.MealTypeInput{Name: cur.Name, Position: cur.Position, Time: cur.Time, Color: cur.Color}
	in.applyTo(&draft)
	t, err := l.svc.UpdateMealType(r.Context(), v, id, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toMealType(t, v))
	return nil
}

func (l *Layer) deleteMealType(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteMealType(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

type mealPlan struct {
	ID           int64           `json:"id"`
	Title        string          `json:"title"`
	Recipe       *recipeOverview `json:"recipe"`
	Servings     float64         `json:"servings"`
	Note         string          `json:"note"`
	NoteMarkdown string          `json:"note_markdown"`
	FromDate     string          `json:"from_date"`
	ToDate       string          `json:"to_date"`
	MealType     mealType        `json:"meal_type"`
	CreatedBy    int64           `json:"created_by"`
	RecipeName   *string         `json:"recipe_name"`
	MealTypeName string          `json:"meal_type_name"`
	Shopping     bool            `json:"shopping"`
}

var hiddenRecipeTitle = map[string]string{"de": "Privates Rezept"}

func toMealPlan(p app.MealPlan, v app.Viewer) mealPlan {
	out := mealPlan{
		ID: p.ID, Title: p.Title, Servings: p.Servings, Note: p.Note, NoteMarkdown: p.Note,
		// Plain dates on purpose: kitshn reads them as they are. A time would be shifted by
		// the phone's time zone, which can move a meal to another day.
		FromDate: p.FromDate, ToDate: p.ToDate, MealType: toMealType(p.MealType, v), CreatedBy: v.User.ID, MealTypeName: p.MealType.Name,
	}
	if p.CreatedBy != nil {
		out.CreatedBy = p.CreatedBy.ID
	}
	if p.Recipe != nil {
		o := toOverview(*p.Recipe)
		out.Recipe = &o
		out.RecipeName = &p.Recipe.Name
	}
	if p.RecipeHidden && out.Title == "" {
		out.Title = "Private recipe"
		if t, ok := hiddenRecipeTitle[strings.SplitN(v.Locale, "-", 2)[0]]; ok {
			out.Title = t
		}
	}
	return out
}

// kitshnDate reads the date kitshn means. It sends local midnight converted to UTC, so
// 9 November in Germany arrives as 2026-11-08T23:00:00Z. Rounding to the nearest midnight
// gives the day the person chose, for time zones from UTC-11 to UTC+12.
func kitshnDate(raw string) string {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return raw // plain dates and anything odd go to the normal validation
	}
	t = t.UTC()
	if t.Hour() >= 12 {
		t = t.AddDate(0, 0, 1)
	}
	return t.Format("2006-01-02")
}

type mealPlanIn struct {
	Title       *string   `json:"title"`
	Recipe      *namedRef `json:"recipe"`
	Servings    *float64  `json:"servings"`
	Note        *string   `json:"note"`
	FromDate    *string   `json:"from_date"`
	ToDate      *string   `json:"to_date"`
	MealType    *namedRef `json:"meal_type"`
	AddShopping *bool     `json:"addshopping"`
}

func (in mealPlanIn) applyTo(out *app.MealPlanInput) {
	set(&out.Title, in.Title)
	set(&out.Servings, in.Servings)
	set(&out.Note, in.Note)
	set(&out.AddToShopping, in.AddShopping)
	if in.FromDate != nil {
		out.FromDate = kitshnDate(*in.FromDate)
	}
	if in.ToDate != nil {
		out.ToDate = kitshnDate(*in.ToDate)
	}
	if in.MealType != nil {
		out.MealTypeID = in.MealType.ID
	}
	if in.Recipe != nil {
		out.RecipeID = in.Recipe.ID
	}
}

func (l *Layer) listMealPlans(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	q := r.URL.Query()
	mealTypeID, _ := strconv.ParseInt(q.Get("meal_type"), 10, 64) // absent means all
	rows, err := l.svc.ListMealPlans(r.Context(), v, q.Get("from_date"), q.Get("to_date"), mealTypeID)
	if err != nil {
		return err
	}
	out := make([]mealPlan, len(rows))
	for i, p := range rows {
		out[i] = toMealPlan(p, v)
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createMealPlan(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in mealPlanIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	var draft app.MealPlanInput
	in.applyTo(&draft)
	p, err := l.svc.CreateMealPlan(r.Context(), v, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toMealPlan(p, v))
	return nil
}

func (l *Layer) getMealPlan(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	p, err := l.svc.GetMealPlan(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toMealPlan(p, v))
	return nil
}

func (l *Layer) patchMealPlan(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in mealPlanIn
	if err := decode(w, r, &in); err != nil {
		return err
	}
	cur, err := l.svc.GetMealPlan(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := app.MealPlanInput{Title: cur.Title, Note: cur.Note, Servings: cur.Servings, FromDate: cur.FromDate, ToDate: cur.ToDate, MealTypeID: cur.MealType.ID}
	if cur.Recipe != nil {
		draft.RecipeID = cur.Recipe.ID
	}
	in.applyTo(&draft)
	p, err := l.svc.UpdateMealPlan(r.Context(), v, id, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toMealPlan(p, v))
	return nil
}

func (l *Layer) deleteMealPlan(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteMealPlan(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

// --- recipe books ---

type recipeBook struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
	Filter      any    `json:"filter"`
	Shared      []user `json:"shared"`
}

func toBook(b app.RecipeBook) recipeBook {
	return recipeBook{ID: b.ID, Name: b.Name, Description: b.Description, Shared: []user{}}
}

type bookEntry struct {
	ID            int64          `json:"id"`
	Book          int64          `json:"book"`
	BookContent   recipeBook     `json:"book_content"`
	Recipe        int64          `json:"recipe"`
	RecipeContent recipeOverview `json:"recipe_content"`
}

func toBookEntry(e app.BookEntry) bookEntry {
	return bookEntry{ID: e.ID, Book: e.BookID, BookContent: toBook(e.Book), Recipe: e.Recipe.ID, RecipeContent: toOverview(e.Recipe)}
}

func (l *Layer) listBooks(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.ListBooks(r.Context(), v)
	if err != nil {
		return err
	}
	needle := strings.ToLower(r.URL.Query().Get("query"))
	out := []recipeBook{}
	for _, b := range rows {
		if needle == "" || strings.Contains(strings.ToLower(b.Name), needle) {
			out = append(out, toBook(b))
		}
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.RecipeBookInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	b, err := l.svc.CreateBook(r.Context(), v, in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toBook(b))
	return nil
}

func (l *Layer) getBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	b, err := l.svc.GetBook(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toBook(b.RecipeBook))
	return nil
}

func (l *Layer) patchBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	cur, err := l.svc.GetBook(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := app.RecipeBookInput{Name: cur.Name, Description: cur.Description}
	set(&draft.Name, in.Name)
	set(&draft.Description, in.Description)
	b, err := l.svc.UpdateBook(r.Context(), v, id, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toBook(b))
	return nil
}

func (l *Layer) deleteBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteBook(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) listBookEntries(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	q := r.URL.Query()
	book, _ := strconv.ParseInt(q.Get("book"), 10, 64)       // absent means any
	recipeID, _ := strconv.ParseInt(q.Get("recipe"), 10, 64) // absent means any
	rows, err := l.svc.ListBookEntries(r.Context(), v, book, recipeID)
	if err != nil {
		return err
	}
	out := make([]bookEntry, len(rows))
	for i, e := range rows {
		out[i] = toBookEntry(e)
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createBookEntry(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Book   int64 `json:"book"`
		Recipe int64 `json:"recipe"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	id, err := l.svc.AddToBook(r.Context(), v, in.Book, in.Recipe)
	if err != nil {
		return err
	}
	e, err := l.svc.GetBookEntry(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toBookEntry(e))
	return nil
}

func (l *Layer) deleteBookEntry(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteBookEntry(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

// --- shopping ---

type category struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toCategoryOut(c app.SupermarketCategory) category {
	return category{ID: c.ID, Name: c.Name, Description: c.Description}
}

type shoppingFood struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	PluralName          *string   `json:"plural_name"`
	SupermarketCategory *category `json:"supermarket_category"`
}

type shoppingPerson struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type shoppingRecipeData struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Recipe     *int64          `json:"recipe"`
	RecipeData *recipeOverview `json:"recipe_data"`
	Mealplan   *int64          `json:"mealplan"`
	MealPlan   *mealPlan       `json:"meal_plan_data"`
	Servings   float64         `json:"servings"`
}

type shoppingEntry struct {
	ID             int64               `json:"id"`
	ListRecipe     *int64              `json:"list_recipe"`
	ShoppingLists  []any               `json:"shopping_lists"`
	Food           shoppingFood        `json:"food"`
	Unit           *unit               `json:"unit"`
	Amount         float64             `json:"amount"`
	Order          int64               `json:"order"`
	Checked        bool                `json:"checked"`
	CreatedBy      shoppingPerson      `json:"created_by"`
	CreatedAt      string              `json:"created_at"`
	ListRecipeData *shoppingRecipeData `json:"list_recipe_data"`
}

func (l *Layer) toShoppingEntry(r *http.Request, v app.Viewer, i app.ShoppingItem) shoppingEntry {
	out := shoppingEntry{
		ID: i.ID, ShoppingLists: []any{}, Order: i.Position, Checked: i.Checked, CreatedAt: i.CreatedAt,
		Food: shoppingFood{ID: i.Food.ID, Name: i.Food.Name, PluralName: strPtr(i.Food.Plural)},
	}
	if i.Category != nil {
		c := toCategoryOut(*i.Category)
		out.Food.SupermarketCategory = &c
	}
	if i.Unit != nil {
		u := toUnit(*i.Unit)
		out.Unit = &u
	}
	if i.Amount != nil {
		out.Amount = *i.Amount
	}
	if i.CreatedBy != nil {
		out.CreatedBy = shoppingPerson{ID: i.CreatedBy.ID, Username: i.CreatedBy.Handle, DisplayName: i.CreatedBy.DisplayName}
	}
	if i.Recipe != nil {
		g := i.Recipe
		out.ListRecipe = &g.ID
		data := &shoppingRecipeData{ID: g.ID, Name: g.Name, Servings: g.Servings}
		if g.RecipeID != 0 {
			data.Recipe = &g.RecipeID
			if rec, err := l.svc.GetRecipe(r.Context(), v, g.RecipeID); err == nil {
				o := toOverview(rec.RecipeSummary)
				data.RecipeData = &o
			}
		}
		if g.MealPlanID != 0 {
			data.Mealplan = &g.MealPlanID
			if p, err := l.svc.GetMealPlan(r.Context(), v, g.MealPlanID); err == nil {
				m := toMealPlan(p, v)
				data.MealPlan = &m
			}
		}
		out.ListRecipeData = data
	}
	return out
}

func (l *Layer) listShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.ListShopping(r.Context(), v)
	if err != nil {
		return err
	}
	after := r.URL.Query().Get("updated_after")
	out := []shoppingEntry{}
	for _, i := range rows {
		if after == "" || i.UpdatedAt >= after {
			out = append(out, l.toShoppingEntry(r, v, i))
		}
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Amount     *float64  `json:"amount"`
		Checked    bool      `json:"checked"`
		ListRecipe int64     `json:"list_recipe"`
		Food       *namedRef `json:"food"`
		Unit       *namedRef `json:"unit"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	draft := app.ShoppingItemInput{Amount: in.Amount, Checked: in.Checked}
	if in.Food != nil {
		draft.FoodID, draft.FoodName = in.Food.ID, in.Food.Name
	}
	if in.Unit != nil {
		draft.UnitID, draft.UnitName = in.Unit.ID, in.Unit.Name
	}
	item, err := l.svc.CreateShoppingItemFor(r.Context(), v, draft, in.ListRecipe)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, l.toShoppingEntry(r, v, item))
	return nil
}

func (l *Layer) bulkShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.ShoppingCheckInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := l.svc.CheckShopping(r.Context(), v, in); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, in)
	return nil
}

func (l *Layer) patchShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		Amount  *float64        `json:"amount"`
		Checked *bool           `json:"checked"`
		Unit    json.RawMessage `json:"unit"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	cur, err := l.svc.GetShoppingItem(r.Context(), v, id)
	if err != nil {
		return err
	}
	draft := app.ShoppingItemInput{FoodID: cur.Food.ID, Amount: cur.Amount, Note: cur.Note, Checked: cur.Checked}
	if cur.Unit != nil {
		draft.UnitID = cur.Unit.ID
	}
	if in.Amount != nil {
		draft.Amount = in.Amount
	}
	set(&draft.Checked, in.Checked)
	if in.Unit != nil { // a JSON null clears the unit
		draft.UnitID, draft.UnitName = 0, ""
		var ref *namedRef
		if err := json.Unmarshal(in.Unit, &ref); err != nil {
			return &app.Error{Status: http.StatusBadRequest, Code: "bad_request", Message: "The request could not be read"}
		}
		if ref != nil {
			draft.UnitID, draft.UnitName = ref.ID, ref.Name
		}
	}
	item, err := l.svc.UpdateShoppingItem(r.Context(), v, id, draft)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l.toShoppingEntry(r, v, item))
	return nil
}

func (l *Layer) deleteShopping(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteShoppingItem(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) createShoppingRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Recipe   int64   `json:"recipe"`
		Servings float64 `json:"servings"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	g, err := l.svc.CreateShoppingGroup(r.Context(), v, in.Recipe, in.Servings)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": g.ID, "name": g.Name, "recipe": g.RecipeID, "servings": g.Servings})
	return nil
}

func (l *Layer) bulkShoppingRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		Entries []struct {
			Amount *float64 `json:"amount"`
			UnitID *int64   `json:"unit_id"`
			FoodID int64    `json:"food_id"`
		} `json:"entries"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	items := make([]app.ShoppingItemInput, 0, len(in.Entries))
	for _, e := range in.Entries {
		item := app.ShoppingItemInput{FoodID: e.FoodID, Amount: e.Amount}
		if e.UnitID != nil {
			item.UnitID = *e.UnitID
		}
		items = append(items, item)
	}
	if err := l.svc.CreateShoppingItemsInGroup(r.Context(), v, id, items); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) deleteShoppingRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteShoppingGroup(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

// --- supermarkets ---

type orderEntry struct {
	ID          int64    `json:"id"`
	Category    category `json:"category"`
	Supermarket int64    `json:"supermarket"`
	Order       int64    `json:"order"`
}

type supermarket struct {
	ID       int64        `json:"id"`
	Name     string       `json:"name"`
	Desc     string       `json:"description"`
	Category []orderEntry `json:"category_to_supermarket"`
}

func toSupermarket(m app.Supermarket) supermarket {
	out := supermarket{ID: m.ID, Name: m.Name, Desc: m.Description, Category: []orderEntry{}}
	for _, e := range m.Categories {
		out.Category = append(out.Category, orderEntry{ID: e.ID, Category: toCategoryOut(e.Category), Supermarket: m.ID, Order: e.Position})
	}
	return out
}

func (l *Layer) listSupermarkets(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.ListSupermarkets(r.Context(), v)
	if err != nil {
		return err
	}
	out := make([]supermarket, len(rows))
	for i, m := range rows {
		out[i] = toSupermarket(m)
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createSupermarket(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.CategoryInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	m, err := l.svc.CreateSupermarket(r.Context(), v, in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toSupermarket(m))
	return nil
}

func (l *Layer) getSupermarket(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	m, err := l.svc.GetSupermarket(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toSupermarket(m))
	return nil
}

func (l *Layer) deleteSupermarket(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteSupermarket(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}

func (l *Layer) listCategories(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.ListCategories(r.Context(), v)
	if err != nil {
		return err
	}
	out := make([]category, len(rows))
	for i, c := range rows {
		out[i] = toCategoryOut(c)
	}
	writeJSON(w, http.StatusOK, pageOf(r, out))
	return nil
}

func (l *Layer) createCategory(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.CategoryInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	c, err := l.svc.CreateCategory(r.Context(), v, in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toCategoryOut(c))
	return nil
}

func (l *Layer) getCategory(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	c, err := l.svc.GetCategory(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toCategoryOut(c))
	return nil
}

func (l *Layer) deleteCategory(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := l.svc.DeleteCategory(r.Context(), v, id); err != nil {
		return err
	}
	return noContent(w)
}
