package app

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// Export formats. Wokko's own keeps everything a recipe has. schema.org is what other apps
// read, laid out like Nextcloud Cookbook's export, which Tandoor and Mealie import too.
const (
	FormatWokko  = "wokko"
	FormatSchema = "schema"
)

// ExportData is recipes in an open format that needs no Wokko to read.
type ExportData struct {
	Format     string         `json:"format"`
	ExportedAt string         `json:"exported_at"`
	User       *Person        `json:"user,omitempty"`
	Recipes    []ExportRecipe `json:"recipes"`
}

// ExportRecipe is a recipe as the API shows it, plus where its photo is inside the archive.
type ExportRecipe struct {
	Recipe
	Photo string `json:"photo,omitempty"`
}

// ExportFormat checks a requested format, with Wokko's own as the default.
func ExportFormat(raw string) (string, error) {
	switch raw {
	case "", FormatWokko:
		return FormatWokko, nil
	case FormatSchema:
		return FormatSchema, nil
	}
	return "", invalid("bad_request", "Unknown export format")
}

func (s *Service) exportOf(user *Person, recipes []Recipe) ExportData {
	out := ExportData{Format: "wokko-export-1", ExportedAt: s.stamp(s.Now()), User: user, Recipes: []ExportRecipe{}}
	for _, r := range recipes {
		item := ExportRecipe{Recipe: r}
		if file := photoFile(r.PhotoURL); file != "" {
			item.Photo = "photos/" + file
		}
		out.Recipes = append(out.Recipes, item)
	}
	return out
}

// photoFile is the stored photo behind a recipe's photo address.
func photoFile(url string) string {
	if url == "" {
		return ""
	}
	return path.Base(strings.SplitN(url, "?", 2)[0])
}

// ExportScope exports the viewer's own recipes ("mine"), or every recipe they can see ("all"):
// theirs, their household's and those shared with them. Other people's private recipes never
// leave through here, not even for an admin. A whole server moves with a backup instead.
func (s *Service) ExportScope(ctx context.Context, v Viewer, scope string) (ExportData, error) {
	switch scope {
	case "", "mine":
		return s.Export(ctx, v)
	case "all":
	default:
		return ExportData{}, invalid("bad_request", "Unknown export scope")
	}
	q := s.Store.Queries
	keys, err := q.ListVisibleRecipeKeys(ctx, sqlcgen.ListVisibleRecipeKeysParams{ViewerID: v.id(), HouseholdID: v.User.HouseholdID})
	if err != nil {
		return ExportData{}, err
	}
	ids := make([]int64, len(keys))
	for i, k := range keys {
		ids[i] = k.ID
	}
	rows, err := q.ListRecipesByIDs(ctx, ids)
	if err != nil {
		return ExportData{}, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	recipes := make([]Recipe, 0, len(rows))
	for _, r := range rows {
		doc, err := s.buildRecipe(ctx, q, v, r)
		if err != nil {
			return ExportData{}, err
		}
		recipes = append(recipes, doc)
	}
	person := toPerson(*v.User)
	return s.exportOf(&person, recipes), nil
}

// ExportRecipe exports one recipe the viewer may see.
func (s *Service) ExportRecipe(ctx context.Context, v Viewer, id int64) (ExportData, error) {
	r, err := s.GetRecipe(ctx, v, id)
	if err != nil {
		return ExportData{}, err
	}
	return s.exportOf(nil, []Recipe{r}), nil
}

// ExportSharedRecipe exports the recipe behind a share link, for anyone holding the link.
func (s *Service) ExportSharedRecipe(ctx context.Context, locale, token string) (ExportData, error) {
	r, err := s.GetSharedRecipe(ctx, locale, token)
	if err != nil {
		return ExportData{}, err
	}
	return s.exportOf(nil, []Recipe{r}), nil
}

// ExportPublicRecipe exports a recipe from the public gallery.
func (s *Service) ExportPublicRecipe(ctx context.Context, locale string, id int64) (ExportData, error) {
	r, err := s.GetPublicRecipe(ctx, locale, id)
	if err != nil {
		return ExportData{}, err
	}
	return s.exportOf(nil, []Recipe{r}), nil
}

// WriteExport writes the recipes as a ZIP in the chosen format.
func (s *Service) WriteExport(w io.Writer, format string, d ExportData) error {
	zw := zip.NewWriter(w)
	var err error
	if format == FormatSchema {
		err = s.writeSchema(zw, d)
	} else {
		err = s.writeWokko(zw, d)
	}
	if err != nil {
		return err
	}
	return zw.Close()
}

func (s *Service) writeWokko(zw *zip.Writer, d ExportData) error {
	f, err := zw.Create("recipes.json")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return err
	}
	written := map[string]bool{}
	for _, r := range d.Recipes {
		if r.Photo == "" || written[r.Photo] {
			continue
		}
		written[r.Photo] = true
		if err := s.copyPhoto(zw, photoFile(r.Photo), r.Photo); err != nil {
			return err
		}
	}
	return nil
}

// writeSchema puts each recipe in its own folder, with recipe.json and full.jpg.
func (s *Service) writeSchema(zw *zip.Writer, d ExportData) error {
	used := map[string]int{}
	for _, r := range d.Recipes {
		dir := folderName(r.Name)
		if n := used[dir]; n > 0 {
			used[dir] = n + 1
			dir = fmt.Sprintf("%s (%d)", dir, n+1)
		} else {
			used[dir] = 1
		}
		f, err := zw.Create(dir + "/recipe.json")
		if err != nil {
			return err
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(schemaRecipe(r)); err != nil {
			return err
		}
		if r.Photo != "" {
			if err := s.copyPhoto(zw, photoFile(r.Photo), dir+"/full.jpg"); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyPhoto adds a stored photo to the archive. A missing file is left out, as a recipe matters
// more than its picture.
func (s *Service) copyPhoto(zw *zip.Writer, file, name string) error {
	p, valid := s.Media.Path(file)
	if !valid {
		return nil
	}
	src, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

// folderName makes a recipe name safe as a folder name in any system.
func folderName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case strings.ContainsRune(`/\:*?"<>|`, r) || unicode.IsControl(r):
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), ". ")
	if r := []rune(out); len(r) > 80 {
		out = string(r[:80])
	}
	if out == "" {
		out = "Recipe"
	}
	return out
}

// FileName is a download name for an export, built from plain letters so every browser keeps it.
func FileName(d ExportData, format string) string {
	base := "wokko-recipes"
	if len(d.Recipes) == 1 {
		var b strings.Builder
		for _, r := range strings.ToLower(d.Recipes[0].Name) {
			switch {
			case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
				b.WriteRune(r)
			case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
				b.WriteRune('-')
			}
		}
		if slug := strings.Trim(b.String(), "-"); slug != "" {
			base = slug
		} else {
			base = "recipe"
		}
	}
	if format == FormatSchema {
		return base + "-schema.zip"
	}
	return base + "-wokko.zip"
}

func schemaRecipe(r ExportRecipe) map[string]any {
	out := map[string]any{"@context": "https://schema.org", "@type": "Recipe", "name": r.Name}
	if r.Description != "" {
		out["description"] = r.Description
	}
	yield := strconv.FormatInt(r.Servings, 10)
	if r.ServingsText != "" {
		yield += " " + r.ServingsText
	}
	out["recipeYield"] = yield
	if r.WorkMinutes > 0 {
		out["prepTime"] = fmt.Sprintf("PT%dM", r.WorkMinutes)
	}
	if r.WaitMinutes > 0 {
		out["cookTime"] = fmt.Sprintf("PT%dM", r.WaitMinutes)
	}
	if total := r.WorkMinutes + r.WaitMinutes; total > 0 {
		out["totalTime"] = fmt.Sprintf("PT%dM", total)
	}
	if r.SourceURL != "" {
		out["url"] = r.SourceURL
	}
	if len(r.Keywords) > 0 {
		names := make([]string, len(r.Keywords))
		for i, k := range r.Keywords {
			names[i] = k.Name
		}
		out["keywords"] = strings.Join(names, ", ")
	}
	if r.Photo != "" {
		out["image"] = "full.jpg"
	}
	ingredients := []string{}
	steps := []map[string]any{}
	for _, st := range r.Steps {
		for _, i := range st.Ingredients {
			ingredients = append(ingredients, ingredientLine(i))
		}
		if strings.TrimSpace(st.Instruction) == "" && st.Name == "" {
			continue
		}
		step := map[string]any{"@type": "HowToStep", "text": st.Instruction}
		if st.Name != "" {
			step["name"] = st.Name
		}
		steps = append(steps, step)
	}
	out["recipeIngredient"] = ingredients
	out["recipeInstructions"] = steps
	if r.CreatedAt != "" {
		out["dateCreated"] = r.CreatedAt
	}
	return out
}

// ingredientLine writes an ingredient the way recipe sites do, such as "200 g lentils, red".
func ingredientLine(i Ingredient) string {
	var parts []string
	many := i.Amount != nil && *i.Amount != 1
	if i.Amount != nil {
		parts = append(parts, strconv.FormatFloat(*i.Amount, 'f', -1, 64))
	}
	if i.Unit != nil {
		switch {
		case i.Unit.Abbreviation != "":
			parts = append(parts, i.Unit.Abbreviation)
		case many && i.Unit.Plural != "":
			parts = append(parts, i.Unit.Plural)
		default:
			parts = append(parts, i.Unit.Name)
		}
	}
	food := i.Food.Name
	if many && i.Unit == nil && i.Food.Plural != "" {
		food = i.Food.Plural
	}
	parts = append(parts, food)
	line := strings.Join(parts, " ")
	if i.Note != "" {
		line += ", " + i.Note
	}
	return line
}
