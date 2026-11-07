package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/serraniel/wokko/backend/internal/importer"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

const maxImportText = 200_000

// ImportRecipe reads a recipe into a draft for the editor. It saves nothing, and new foods
// and units only appear when the person saves the recipe.
func (s *Service) ImportRecipe(ctx context.Context, in ImportRequest) (ImportedRecipe, error) {
	out, parsed, err := s.importDraft(ctx, in)
	if err != nil {
		return ImportedRecipe{}, err
	}
	if parsed.ImageURL != "" {
		// A recipe without its photo is still a good import, so failing here is fine.
		if data, mime, err := s.Fetcher.Image(ctx, parsed.ImageURL); err == nil {
			out.ImageData, out.ImageType = base64.StdEncoding.EncodeToString(data), mime
		} else {
			s.Log.Info("imported recipe without its photo", "error", err)
		}
	}
	return out, nil
}

// ImportForKitshn reads a draft and hands back the photo's address, which kitshn sends back
// when it saves the recipe.
func (s *Service) ImportForKitshn(ctx context.Context, url string) (ImportedRecipe, string, error) {
	out, parsed, err := s.importDraft(ctx, ImportRequest{URL: url})
	if err != nil {
		return ImportedRecipe{}, "", err
	}
	return out, parsed.ImageURL, nil
}

// SetPhotoFromURL downloads a photo through the import's safe fetcher, then treats it as an upload.
func (s *Service) SetPhotoFromURL(ctx context.Context, v Viewer, id int64, url string) (Recipe, error) {
	data, _, err := s.Fetcher.Image(ctx, url)
	if err != nil {
		return Recipe{}, importError(err)
	}
	return s.SetPhoto(ctx, v, id, bytes.NewReader(data))
}

func (s *Service) importDraft(ctx context.Context, in ImportRequest) (ImportedRecipe, *importer.Recipe, error) {
	var (
		parsed *importer.Recipe
		err    error
	)
	pasted := strings.TrimSpace(in.Text)
	switch {
	case pasted != "":
		if len(pasted) > maxImportText {
			return ImportedRecipe{}, nil, invalid("import_too_long", "This text is too long to import")
		}
		lower := strings.ToLower(pasted[:min(len(pasted), 2000)])
		if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
			parsed, err = importer.ParseHTML([]byte(pasted), "text/html; charset=utf-8", strings.TrimSpace(in.URL))
		} else {
			parsed, err = importer.ParseText(pasted)
		}
	case strings.TrimSpace(in.URL) != "":
		var (
			body        []byte
			contentType string
			final       string
		)
		body, contentType, final, err = s.Fetcher.Page(ctx, in.URL)
		if err == nil {
			parsed, err = importer.ParseHTML(body, contentType, final)
		}
	default:
		return ImportedRecipe{}, nil, invalid("import_empty", "Enter a link or paste a recipe")
	}
	if err != nil {
		return ImportedRecipe{}, nil, importError(err)
	}

	units, err := s.unitSpellings(ctx)
	if err != nil {
		return ImportedRecipe{}, nil, err
	}
	return ImportedRecipe{Recipe: toRecipeInput(parsed, units), Guessed: parsed.Guessed}, parsed, nil
}

func importError(err error) error {
	switch {
	case errors.Is(err, importer.ErrBadURL):
		return invalid("import_bad_url", "This doesn't look like a web address")
	case errors.Is(err, importer.ErrPrivate):
		return invalid("import_private", "Wokko only imports from public websites")
	case errors.Is(err, importer.ErrBlocked):
		return fail(http.StatusBadGateway, "import_blocked", "This site doesn't allow automatic imports")
	case errors.Is(err, importer.ErrUnreachable):
		return fail(http.StatusBadGateway, "import_unreachable", "This site could not be reached")
	case errors.Is(err, importer.ErrNotPage):
		return invalid("import_not_page", "This address isn't a recipe page")
	case errors.Is(err, importer.ErrNoRecipe):
		return fail(http.StatusUnprocessableEntity, "import_no_recipe", "No recipe found")
	}
	return err
}

func (s *Service) unitSpellings(ctx context.Context) (map[string]string, error) {
	rows, err := s.Store.Queries.ListAllUnitSpellings(ctx)
	if err != nil {
		return nil, err
	}
	units := make(map[string]string, len(rows)*3)
	for _, r := range rows {
		for _, spelling := range []string{r.Name, r.Plural.String, r.Abbreviation.String} {
			if spelling != "" {
				units[strings.ToLower(spelling)] = r.Name
			}
		}
	}
	return units, nil
}

func toRecipeInput(r *importer.Recipe, units map[string]string) RecipeInput {
	in := RecipeInput{
		Name:        firstRunes(r.Name, 200),
		Description: r.Description,
		WorkMinutes: r.WorkMinutes,
		WaitMinutes: r.WaitMinutes,
		Servings:    r.Servings,
		SourceURL:   r.SourceURL,
		Visibility:  "private", // Someone else's text and photo: sharing it stays a deliberate choice.
		Keywords:    r.Keywords,
		Steps:       []StepInput{},
	}
	if in.Servings == 0 {
		in.Servings = 4
	}
	if in.Keywords == nil {
		in.Keywords = []string{}
	}
	ingredients := []IngredientInput{}
	for _, l := range r.Ingredients {
		p := importer.ParseIngredient(l, units)
		ingredients = append(ingredients, IngredientInput{FoodName: p.Food, UnitName: p.Unit, Amount: p.Amount, Note: p.Note})
	}
	// Recipe data doesn't say which step uses what, so everything starts in the first step.
	for _, st := range r.Steps {
		in.Steps = append(in.Steps, StepInput{Name: st.Name, Instruction: st.Text, Ingredients: []IngredientInput{}})
	}
	if len(in.Steps) == 0 {
		in.Steps = append(in.Steps, StepInput{Ingredients: []IngredientInput{}})
	}
	in.Steps[0].Ingredients = ingredients
	return in
}

// ImportFile reads another app's export. With dryRun it only reports what's inside. Otherwise
// each recipe is saved on its own, private and skipping likely duplicates, so one bad recipe
// doesn't cost the others and importing twice is harmless.
func (s *Service) ImportFile(ctx context.Context, v Viewer, data io.ReaderAt, size int64, name string, dryRun bool) (FileImportResult, error) {
	recipes, err := importer.ReadFile(data, size, name)
	if errors.Is(err, importer.ErrFileTooBig) {
		return FileImportResult{}, fail(http.StatusRequestEntityTooLarge, "import_file_too_big", "This file is too big to import")
	}
	if err != nil {
		return FileImportResult{}, fail(http.StatusUnprocessableEntity, "import_file_unknown", "No recipes found in this file")
	}
	units, err := s.unitSpellings(ctx)
	if err != nil {
		return FileImportResult{}, err
	}
	keys, err := s.Store.Queries.ListVisibleRecipeKeys(ctx, sqlcgen.ListVisibleRecipeKeysParams{ViewerID: v.id(), HouseholdID: v.User.HouseholdID})
	if err != nil {
		return FileImportResult{}, err
	}
	known := map[string]bool{}
	for _, k := range keys {
		known[text.Normalize(k.Name)] = true
	}
	out := FileImportResult{Items: make([]FileImportItem, 0, len(recipes))}
	for _, r := range recipes {
		in := fileRecipeInput(r, units)
		item := FileImportItem{Name: in.Name, Format: r.Format, Steps: int64(len(in.Steps)), HasPhoto: len(r.Photo) > 0}
		for _, st := range in.Steps {
			item.Ingredients += int64(len(st.Ingredients))
		}
		key := text.Normalize(in.Name)
		item.Duplicate = known[key]
		known[key] = true
		switch {
		case item.Duplicate:
			out.Skipped++
		case !dryRun:
			saved, err := s.CreateRecipe(ctx, v, in)
			if err != nil {
				var ae *Error
				if !errors.As(err, &ae) {
					return out, err // A problem with the server, not with this recipe.
				}
				item.Error = ae.Code
				out.Failed++
				break
			}
			item.RecipeID = saved.ID
			out.Imported++
			if len(r.Photo) > 0 {
				if _, err := s.SetPhoto(ctx, v, saved.ID, bytes.NewReader(r.Photo)); err != nil {
					item.PhotoFailed = true // The recipe matters more than its picture.
					s.Log.Info("imported recipe without its photo", "recipe", saved.ID, "error", err)
				}
			}
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func fileRecipeInput(r importer.FileRecipe, units map[string]string) RecipeInput {
	in := RecipeInput{
		Name: firstRunes(strings.TrimSpace(r.Name), 200), Description: r.Description, WorkMinutes: r.WorkMinutes, WaitMinutes: r.WaitMinutes,
		Servings: r.Servings, ServingsText: firstRunes(r.ServingsText, 32), Visibility: "private", Keywords: []string{}, Steps: []StepInput{},
	}
	if in.Servings <= 0 {
		in.Servings = 4
	}
	if strings.HasPrefix(r.Source, "http://") || strings.HasPrefix(r.Source, "https://") {
		in.SourceURL = r.Source
	}
	for _, k := range r.Keywords {
		if k = strings.TrimSpace(k); k != "" && len(in.Keywords) < 20 {
			in.Keywords = append(in.Keywords, firstRunes(k, 60))
		}
	}
	for _, st := range r.Steps {
		step := StepInput{Name: st.Name, Instruction: st.Instruction, Minutes: max(st.Minutes, 0), Ingredients: []IngredientInput{}}
		for _, i := range st.Ingredients {
			ing := IngredientInput{FoodName: i.Food, UnitName: i.Unit, Amount: i.Amount, Note: i.Note}
			if i.Line != "" {
				p := importer.ParseIngredient(i.Line, units)
				ing = IngredientInput{FoodName: p.Food, UnitName: p.Unit, Amount: p.Amount, Note: p.Note}
			}
			if strings.TrimSpace(ing.FoodName) != "" {
				ing.FoodName = firstRunes(strings.TrimSpace(ing.FoodName), 200)
				step.Ingredients = append(step.Ingredients, ing)
			}
		}
		in.Steps = append(in.Steps, step)
	}
	return in
}

func firstRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
