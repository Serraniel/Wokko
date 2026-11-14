package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/serraniel/wokko/backend/internal/app"
)

type ingredientArgs struct {
	FoodID int64    `json:"food_id,omitempty" jsonschema:"id of a known food from search_foods. Preferred over food, because it always means that exact entry"`
	Food   string   `json:"food,omitempty" jsonschema:"food name, used when there is no food_id. Copy a name exactly from search_foods. A name that isn't in the catalog becomes a new food"`
	UnitID int64    `json:"unit_id,omitempty" jsonschema:"id of a known unit from search_units. Preferred over unit"`
	Unit   string   `json:"unit,omitempty" jsonschema:"unit name or abbreviation, such as g or tbsp, used when there is no unit_id. Leave both out for whole items, such as 2 eggs"`
	Amount *float64 `json:"amount,omitempty" jsonschema:"amount for the recipe's servings. Leave out for 'to taste'"`
	Note   string   `json:"note,omitempty" jsonschema:"a note, such as finely chopped"`
}

type stepArgs struct {
	Name        string           `json:"name,omitempty" jsonschema:"optional step title"`
	Instruction string           `json:"instruction" jsonschema:"what to do in this step"`
	Minutes     int64            `json:"minutes,omitempty" jsonschema:"how long this step takes"`
	Ingredients []ingredientArgs `json:"ingredients,omitempty" jsonschema:"ingredients used in this step"`
}

type recipeArgs struct {
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Servings     int64      `json:"servings,omitempty" jsonschema:"how many the amounts make"`
	ServingsText string     `json:"servings_text,omitempty" jsonschema:"what servings counts when it isn't portions, such as loaf or cookies. Leave out for portions"`
	WorkMinutes  int64      `json:"work_minutes,omitempty"`
	WaitMinutes  int64      `json:"wait_minutes,omitempty"`
	SourceURL    string     `json:"source_url,omitempty"`
	Keywords     []string   `json:"keywords,omitempty" jsonschema:"tags. Reuse names from search_tags, a new name becomes a new tag"`
	Steps        []stepArgs `json:"steps,omitempty"`
}

type createArgs struct {
	recipeArgs
	ConfirmedNew []string `json:"confirmed_new,omitempty" jsonschema:"new food and unit names the person agreed to add, only after a save asked for confirmation"`
	Household    bool     `json:"share_with_household,omitempty" jsonschema:"also show it to the person's household. Otherwise only they can see it"`
}

type updateArgs struct {
	recipeArgs
	ConfirmedNew []string `json:"confirmed_new,omitempty" jsonschema:"new food and unit names the person agreed to add, only after a save asked for confirmation"`
	ID           int64    `json:"id"`
	BaseVersion  int64    `json:"base_version" jsonschema:"the version from get_recipe, so a newer edit isn't overwritten"`
}

func (a recipeArgs) input() app.RecipeInput {
	in := app.RecipeInput{Name: a.Name, Description: a.Description, Servings: a.Servings, ServingsText: a.ServingsText, WorkMinutes: a.WorkMinutes, WaitMinutes: a.WaitMinutes, SourceURL: a.SourceURL, Keywords: a.Keywords, Steps: []app.StepInput{}}
	if in.Keywords == nil {
		in.Keywords = []string{}
	}
	for _, st := range a.Steps {
		out := app.StepInput{Name: st.Name, Instruction: st.Instruction, Minutes: st.Minutes, Ingredients: []app.IngredientInput{}}
		for _, i := range st.Ingredients {
			out.Ingredients = append(out.Ingredients, app.IngredientInput{FoodID: i.FoodID, FoodName: i.Food, UnitID: i.UnitID, UnitName: i.Unit, Amount: i.Amount, Note: i.Note})
		}
		in.Steps = append(in.Steps, out)
	}
	return in
}

type idArgs struct {
	ID int64 `json:"id"`
}

type searchArgs struct {
	Query   string `json:"query,omitempty" jsonschema:"words to find in names, ingredients, tags and steps"`
	Keyword int64  `json:"keyword_id,omitempty"`
	Limit   int64  `json:"limit,omitempty"`
	Offset  int64  `json:"offset,omitempty"`
}

type catalogArgs struct {
	Query   string   `json:"query,omitempty" jsonschema:"part of a name, in any language"`
	Queries []string `json:"queries,omitempty" jsonschema:"several names at once, such as every ingredient of a recipe. Each gets its own matches"`
	Limit   int64    `json:"limit,omitempty" jsonschema:"matches per name, 10 if left out"`
}

type tagArgs struct {
	Query string `json:"query,omitempty" jsonschema:"part of a tag name"`
	Limit int64  `json:"limit,omitempty"`
}

type matches[T any] struct {
	Query string `json:"query"`
	Total int64  `json:"total"`
	Items []T    `json:"items"`
}

// search looks up each name the assistant asks about, so a whole ingredient list takes one call.
func search[T any](in catalogArgs, fn func(query string, limit int64) (app.Page[T], error)) (any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	if len(in.Queries) == 0 {
		return fn(in.Query, limit)
	}
	if len(in.Queries) > 50 {
		return nil, &app.Error{Status: 400, Code: "too_many_queries", Message: "Search at most 50 names at once"}
	}
	out := []matches[T]{}
	for _, q := range in.Queries {
		page, err := fn(q, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, matches[T]{Query: q, Total: page.Total, Items: page.Items})
	}
	return out, nil
}

type photoArgs struct {
	ID       int64  `json:"id" jsonschema:"the recipe"`
	ImageURL string `json:"image_url" jsonschema:"public address of a photo, such as the image of the page the recipe came from"`
}

type nameArgs struct {
	Locale       string `json:"locale" jsonschema:"language code, such as en or de"`
	Name         string `json:"name"`
	Plural       string `json:"plural,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty" jsonschema:"units only, such as tbsp"`
}

func (a namesArgs) translations() []app.Translation {
	out := []app.Translation{}
	for _, t := range a.Translations {
		out = append(out, app.Translation{Locale: t.Locale, Name: t.Name, Plural: t.Plural, Abbreviation: t.Abbreviation})
	}
	return out
}

type namesArgs struct {
	ID           int64      `json:"id" jsonschema:"the entry from search_foods or search_units. Only works where can_edit is true"`
	Aliases      []string   `json:"aliases,omitempty" jsonschema:"other spellings or synonyms, such as a regional name"`
	Translations []nameArgs `json:"translations,omitempty" jsonschema:"names in other languages, with locale such as en or de. A language that already has a name keeps it and only gains a missing plural or abbreviation"`
}

type urlArgs struct {
	URL string `json:"url" jsonschema:"address of a recipe page"`
}

type rangeArgs struct {
	From string `json:"from,omitempty" jsonschema:"first date, YYYY-MM-DD"`
	To   string `json:"to,omitempty" jsonschema:"last date, YYYY-MM-DD"`
}

type planArgs struct {
	Date       string  `json:"date" jsonschema:"YYYY-MM-DD"`
	ToDate     string  `json:"to_date,omitempty" jsonschema:"last day, for leftovers or a longer plan"`
	MealTypeID int64   `json:"meal_type_id" jsonschema:"from list_meal_types"`
	RecipeID   int64   `json:"recipe_id,omitempty"`
	Title      string  `json:"title,omitempty" jsonschema:"used when there's no recipe, such as Eating out"`
	Servings   float64 `json:"servings,omitempty"`
	Note       string  `json:"note,omitempty"`
}

type shoppingArgs struct {
	Items []string `json:"items" jsonschema:"one line each, such as 2 kg potatoes"`
}

type shoppingRecipeArgs struct {
	RecipeID int64   `json:"recipe_id"`
	Servings float64 `json:"servings,omitempty" jsonschema:"scales the amounts. Defaults to the recipe's servings"`
}

type checkArgs struct {
	IDs     []int64 `json:"ids"`
	Checked bool    `json:"checked"`
}

type bookEntryArgs struct {
	BookID   int64 `json:"book_id"`
	RecipeID int64 `json:"recipe_id"`
}

type bookArgs struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// plainError makes failures readable for the assistant, without internal details.
func plainError(err error) error {
	var ae *app.Error
	if errors.As(err, &ae) {
		return fmt.Errorf("%s (%s)", ae.Message, ae.Code)
	}
	return errors.New("something went wrong on the server")
}

type result = *mcp.CallToolResult

// read registers a tool that changes nothing.
func read[In any](l *Layer, s *mcp.Server, name, description string, fn func(context.Context, In) (any, error)) {
	yes := true
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: yes}}, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (result, any, error) {
		out, err := fn(ctx, in)
		if err != nil {
			return nil, nil, plainError(err)
		}
		return nil, wrap(out), nil
	})
}

// wrap gives every result an object at the top, which MCP's structured output requires.
func wrap(out any) any { return map[string]any{"result": out} }

// write registers a tool that changes something, and records it for the person to see.
func write[In any](l *Layer, s *mcp.Server, a app.TokenAccess, name, description string, fn func(context.Context, In) (any, string, error)) {
	destructive := false
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive}}, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (result, any, error) {
		out, target, err := fn(ctx, in)
		if err != nil {
			return nil, nil, plainError(err)
		}
		l.svc.LogAssistantActivity(ctx, a, name, target)
		return nil, wrap(out), nil
	})
}

// server builds the tools this token may use. A tool beyond its scope doesn't exist for it.
//
// Some things are deliberately never offered: finding or listing other people, making a
// recipe public, account and admin actions, and permanent deletion.
func (l *Layer) server(a app.TokenAccess) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "wokko", Title: "Wokko", Version: l.svc.Cfg.Version}, &mcp.ServerOptions{Instructions: instructionsFor(a)})
	v, svc := a.Viewer, l.svc

	read(l, s, "search_recipes", "Find recipes this person can see. Without a query it lists the most recently changed.", func(ctx context.Context, in searchArgs) (any, error) {
		return svc.ListRecipes(ctx, v, app.RecipeQuery{Q: in.Query, KeywordID: in.Keyword, Limit: in.Limit, Offset: in.Offset})
	})
	read(l, s, "get_recipe", "Read a whole recipe, with its steps, ingredients and version.", func(ctx context.Context, in idArgs) (any, error) {
		return svc.GetRecipe(ctx, v, in.ID)
	})
	read(l, s, "get_recipe_nutrition", "Estimate a recipe's nutrition. 'missing' lists ingredients that couldn't be counted.", func(ctx context.Context, in idArgs) (any, error) {
		return svc.GetRecipeNutrition(ctx, v, in.ID)
	})
	read(l, s, "search_foods", "Find known ingredients before writing a recipe. Matches part of any name, in every language and every alias, and returns each food's id, translations and aliases. Pass every ingredient in queries at once. When a name finds nothing, try again with the singular, the word stem, a synonym, a regional name or the name in another language (Frühlingszwiebel, Lauchzwiebel, spring onion, scallion), then use the food_id you found.", func(ctx context.Context, in catalogArgs) (any, error) {
		return search(in, func(q string, n int64) (app.Page[app.Food], error) { return svc.ListFoods(ctx, v, q, n, 0) })
	})
	read(l, s, "search_units", "Find known units of measure by name, plural, abbreviation or alias, in every language. Use the unit_id you find. Try the abbreviation and the full name (EL, Esslöffel, tbsp).", func(ctx context.Context, in catalogArgs) (any, error) {
		return search(in, func(q string, n int64) (app.Page[app.Unit], error) { return svc.ListUnits(ctx, v, q, n, 0) })
	})
	read(l, s, "search_tags", "Find existing recipe tags, so a recipe reuses them instead of adding near-duplicates.", func(ctx context.Context, in tagArgs) (any, error) {
		return svc.ListKeywords(ctx, v, in.Query, in.Limit, 0)
	})
	read(l, s, "import_recipe_from_url", "Read a recipe from a public web page into a draft. This saves nothing: show it, then use create_recipe.", func(ctx context.Context, in urlArgs) (any, error) {
		out, _, err := svc.ImportForKitshn(ctx, in.URL)
		return out, err
	})
	read(l, s, "list_meal_types", "List the household's meal types, such as breakfast or dinner.", func(ctx context.Context, _ struct{}) (any, error) {
		return svc.ListMealTypes(ctx, v)
	})
	read(l, s, "list_meal_plans", "Read the household's meal plan for a date range.", func(ctx context.Context, in rangeArgs) (any, error) {
		return svc.ListMealPlans(ctx, v, in.From, in.To, 0)
	})
	read(l, s, "list_shopping", "Read the household's shopping list.", func(ctx context.Context, _ struct{}) (any, error) {
		return svc.ListShopping(ctx, v)
	})
	read(l, s, "list_books", "List the household's recipe books.", func(ctx context.Context, _ struct{}) (any, error) {
		return svc.ListBooks(ctx, v)
	})

	if a.Scope == "write" || a.Scope == "delete" {
		write(l, s, a, "create_recipe", "Save a new recipe, with steps that each list the ingredients they use. Look up every ingredient and unit with search_foods and search_units first and give their ids. It stays private unless share_with_household is set.", func(ctx context.Context, in createArgs) (any, string, error) {
			draft := in.input()
			draft.Visibility = "private"
			if in.Household {
				draft.Visibility = "household"
			}
			if err := svc.CheckAssistantNewEntries(ctx, v, draft, in.ConfirmedNew); err != nil {
				return nil, "", err
			}
			r, err := svc.CreateRecipe(ctx, v, draft)
			return r, r.Name, err
		})
		write(l, s, a, "update_recipe", "Replace a recipe's whole contents, including every step and ingredient, so send everything that should stay. Keep the food_id and unit_id from get_recipe. Its sharing and photo don't change, and the previous version stays in its history.", func(ctx context.Context, in updateArgs) (any, string, error) {
			cur, err := svc.GetRecipe(ctx, v, in.ID)
			if err != nil {
				return nil, "", err
			}
			draft := in.input()
			draft.Visibility = cur.Visibility
			draft.BaseVersion = in.BaseVersion
			if err := svc.CheckAssistantNewEntries(ctx, v, draft, in.ConfirmedNew); err != nil {
				return nil, "", err
			}
			r, err := svc.UpdateRecipe(ctx, v, in.ID, draft)
			return r, r.Name, err
		})
		write(l, s, a, "set_recipe_photo", "Give a recipe a photo from a public image address. It replaces the current photo.", func(ctx context.Context, in photoArgs) (any, string, error) {
			r, err := svc.SetPhotoFromURL(ctx, v, in.ID, in.ImageURL)
			return r, r.Name, err
		})
		write(l, s, a, "add_food_names", "Teach a known food another name: an alias, synonym or regional spelling, or a translation. Use it when a recipe says a food differently than the catalog, so later searches find it. Nothing is removed or renamed.", func(ctx context.Context, in namesArgs) (any, string, error) {
			f, err := svc.AddFoodNames(ctx, v, in.ID, in.translations(), in.Aliases)
			return f, f.Name, err
		})
		write(l, s, a, "add_unit_names", "Teach a known unit another name, abbreviation or translation, the same way as add_food_names.", func(ctx context.Context, in namesArgs) (any, string, error) {
			u, err := svc.AddUnitNames(ctx, v, in.ID, in.translations(), in.Aliases)
			return u, u.Name, err
		})
		write(l, s, a, "create_meal_plan", "Plan a meal, with a recipe or just a title.", func(ctx context.Context, in planArgs) (any, string, error) {
			p, err := svc.CreateMealPlan(ctx, v, app.MealPlanInput{Title: in.Title, Note: in.Note, Servings: in.Servings, FromDate: in.Date, ToDate: in.ToDate, MealTypeID: in.MealTypeID, RecipeID: in.RecipeID})
			return p, strings.TrimSpace(in.Date + " " + in.Title), err
		})
		write(l, s, a, "delete_meal_plan", "Remove a planned meal. The recipe itself isn't touched.", func(ctx context.Context, in idArgs) (any, string, error) {
			return map[string]bool{"ok": true}, fmt.Sprintf("plan %d", in.ID), svc.DeleteMealPlan(ctx, v, in.ID)
		})
		write(l, s, a, "add_to_shopping", "Add free-text items to the shopping list.", func(ctx context.Context, in shoppingArgs) (any, string, error) {
			added := []app.ShoppingItem{}
			for _, line := range in.Items {
				item, err := svc.CreateShoppingItem(ctx, v, app.ShoppingItemInput{Line: line})
				if err != nil {
					return nil, "", err
				}
				added = append(added, item)
			}
			return added, strings.Join(in.Items, ", "), nil
		})
		write(l, s, a, "add_recipe_to_shopping", "Add a recipe's ingredients to the shopping list, scaled to the servings.", func(ctx context.Context, in shoppingRecipeArgs) (any, string, error) {
			g, err := svc.AddRecipeToShopping(ctx, v, app.ShoppingAddRecipe{RecipeID: in.RecipeID, Servings: in.Servings})
			return g, g.Name, err
		})
		write(l, s, a, "check_shopping", "Tick or untick shopping items.", func(ctx context.Context, in checkArgs) (any, string, error) {
			return map[string]bool{"ok": true}, fmt.Sprintf("%d items", len(in.IDs)), svc.CheckShopping(ctx, v, app.ShoppingCheckInput{IDs: in.IDs, Checked: in.Checked})
		})
		write(l, s, a, "create_book", "Create a recipe book, which the household can see.", func(ctx context.Context, in bookArgs) (any, string, error) {
			b, err := svc.CreateBook(ctx, v, app.RecipeBookInput{Name: in.Name, Description: in.Description})
			return b, b.Name, err
		})
		write(l, s, a, "add_recipe_to_book", "Put a recipe into a book.", func(ctx context.Context, in bookEntryArgs) (any, string, error) {
			_, err := svc.AddToBook(ctx, v, in.BookID, in.RecipeID)
			return map[string]bool{"ok": true}, fmt.Sprintf("recipe %d into book %d", in.RecipeID, in.BookID), err
		})
	}
	if a.Scope == "delete" {
		write(l, s, a, "delete_recipe", "Move a recipe to the trash. The person can restore it for 30 days.", func(ctx context.Context, in idArgs) (any, string, error) {
			return map[string]bool{"ok": true}, fmt.Sprintf("recipe %d", in.ID), svc.DeleteRecipe(ctx, v, in.ID)
		})
	}
	return s
}
