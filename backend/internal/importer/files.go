package importer

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Limits for uploaded archives. A small zip can unpack to something enormous, so every read
// counts against a budget instead of trusting the sizes an archive claims.
const (
	MaxFileBytes   = 512 << 20
	maxEntryBytes  = 64 << 20
	maxTotalBytes  = 1 << 30
	maxEntries     = 20000
	maxFileRecipes = 5000
	maxZipDepth    = 2
)

var (
	ErrFileTooBig  = errors.New("the file unpacks to more than the limit")
	ErrFileUnknown = errors.New("no recipes found in this file")
)

// FileIngredient has either structured fields or, for sources with plain text, a Line.
type FileIngredient struct {
	Food, Unit, Note, Line string
	Amount                 *float64
}

type FileStep struct {
	Name, Instruction string
	Minutes           int64
	Ingredients       []FileIngredient
}

// FileRecipe is one recipe read from another app's file.
type FileRecipe struct {
	Format                                  string
	Name, Description, ServingsText, Source string
	Servings, WorkMinutes, WaitMinutes      int64
	Keywords                                []string
	Steps                                   []FileStep
	Photo                                   []byte
}

type fileReader struct {
	budget  int64
	entries int
	out     []FileRecipe
}

func (f *fileReader) read(r io.Reader) ([]byte, error) {
	limit := min(f.budget, maxEntryBytes)
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrFileTooBig
	}
	f.budget -= int64(len(data))
	return data, nil
}

// ReadFile finds recipes in an uploaded file, whichever app it came from.
func ReadFile(data io.ReaderAt, size int64, name string) ([]FileRecipe, error) {
	f := &fileReader{budget: maxTotalBytes}
	head := make([]byte, 4)
	if _, err := data.ReadAt(head, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, ErrFileUnknown
	}
	var err error
	switch {
	case bytes.HasPrefix(head, []byte("PK")):
		var archive *zip.Reader
		if archive, err = zip.NewReader(data, size); err != nil {
			return nil, ErrFileUnknown
		}
		err = f.readZip(archive, 0)
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}):
		var body []byte
		if body, err = f.read(io.NewSectionReader(data, 0, size)); err == nil {
			err = f.readGzip(body, nil)
		}
	default:
		var body []byte
		if body, err = f.read(io.NewSectionReader(data, 0, size)); err == nil {
			f.readJSON(body, nil)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(f.out) == 0 {
		return nil, ErrFileUnknown
	}
	return f.out, nil
}

var photoName = regexp.MustCompile(`(?i)^(image|original|full|photo|cover)\.(jpe?g|png|webp)$`)

func (f *fileReader) readZip(archive *zip.Reader, depth int) error {
	byName := map[string]*zip.File{}
	for _, e := range archive.File {
		byName[e.Name] = e
	}
	// A recipe's photo sits beside its JSON, or in an images folder below it.
	photoFor := func(jsonPath string) []byte {
		dir := path.Dir(jsonPath)
		for _, e := range archive.File {
			entryDir := path.Dir(e.Name)
			if (entryDir == dir || entryDir == path.Join(dir, "images")) && photoName.MatchString(path.Base(e.Name)) {
				if data, err := f.open(e); err == nil {
					return data
				}
				return nil
			}
		}
		return nil
	}
	for _, e := range archive.File {
		f.entries++
		if f.entries > maxEntries || len(f.out) >= maxFileRecipes {
			return ErrFileTooBig
		}
		name := strings.ToLower(path.Base(e.Name))
		if e.FileInfo().IsDir() || strings.HasPrefix(e.Name, "__MACOSX/") || strings.HasPrefix(name, ".") {
			continue
		}
		switch {
		case strings.HasSuffix(name, ".zip"):
			if depth >= maxZipDepth {
				continue
			}
			data, err := f.open(e)
			if err != nil {
				return err
			}
			inner, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				continue // One broken inner archive doesn't sink the others.
			}
			if err := f.readZip(inner, depth+1); err != nil {
				return err
			}
		case strings.HasSuffix(name, ".paprikarecipe"):
			data, err := f.open(e)
			if err != nil {
				return err
			}
			if err := f.readGzip(data, nil); err != nil && errors.Is(err, ErrFileTooBig) {
				return err
			}
		case strings.HasSuffix(name, ".json"):
			data, err := f.open(e)
			if err != nil {
				return err
			}
			f.readJSON(data, func() []byte { return photoFor(e.Name) })
		}
	}
	return nil
}

func (f *fileReader) open(e *zip.File) ([]byte, error) {
	r, err := e.Open()
	if err != nil {
		return nil, nil // A damaged entry is skipped, like any other unreadable one.
	}
	defer r.Close()
	data, err := f.read(r)
	if errors.Is(err, ErrFileTooBig) {
		return nil, err
	}
	if err != nil {
		return nil, nil
	}
	return data, nil
}

func (f *fileReader) readGzip(data []byte, photo func() []byte) error {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	defer r.Close()
	body, err := f.read(r)
	if err != nil {
		if errors.Is(err, ErrFileTooBig) {
			return err
		}
		return nil
	}
	f.readJSON(body, photo)
	return nil
}

// readJSON works out what a file is from its fields. Anything else in a zip, such as a
// settings file, is ignored.
func (f *fileReader) readJSON(data []byte, photo func() []byte) {
	var value any
	if json.Unmarshal(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), &value) != nil {
		return
	}
	for _, item := range asList(value) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		var r *FileRecipe
		switch {
		case isPaprika(m):
			r = fromPaprika(m)
		case isTandoor(m):
			r = fromTandoor(m)
		case isMealie(m):
			r = fromMealie(m)
		default:
			if node := findRecipeNode(m, 0); node != nil {
				r = fromSchema(node)
			}
		}
		if r == nil || strings.TrimSpace(r.Name) == "" {
			continue
		}
		if r.Photo == nil && photo != nil {
			r.Photo = photo()
		}
		f.out = append(f.out, *r)
	}
}

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// number also accepts text, because Tandoor writes decimals as "500.000".
func number(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		switch n := m[k].(type) {
		case float64:
			return n
		case string:
			if v, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
				return v
			}
		}
	}
	return 0
}

func field(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func nameOf(v any) string {
	if m, ok := v.(map[string]any); ok {
		return str(m, "name")
	}
	return ""
}

func names(values ...any) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range values {
		for _, e := range asList(v) {
			name := nameOf(e)
			if s, ok := e.(string); ok {
				name = strings.TrimSpace(s)
			}
			if key := strings.ToLower(name); name != "" && !seen[key] {
				seen[key] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// --- Tandoor: steps hold ingredients with food and unit objects ---

func isTandoor(m map[string]any) bool {
	_, steps := m["steps"].([]any)
	_, work := m["working_time"]
	return steps && work
}

func fromTandoor(m map[string]any) *FileRecipe {
	r := &FileRecipe{Format: "tandoor", Name: str(m, "name"), Description: str(m, "description"), ServingsText: str(m, "servings_text"), Source: str(m, "source_url"),
		Servings: int64(number(m, "servings")), WorkMinutes: int64(number(m, "working_time")), WaitMinutes: int64(number(m, "waiting_time")), Keywords: names(m["keywords"])}
	for _, raw := range asList(m["steps"]) {
		st, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		step := FileStep{Name: str(st, "name"), Instruction: str(st, "instruction"), Minutes: int64(number(st, "time"))}
		for _, rawIng := range asList(st["ingredients"]) {
			i, ok := rawIng.(map[string]any)
			if !ok {
				continue
			}
			food := nameOf(i["food"])
			if food == "" {
				continue // A section header in Tandoor, which Wokko doesn't have.
			}
			ing := FileIngredient{Food: food, Unit: nameOf(i["unit"]), Note: str(i, "note")}
			if a := number(i, "amount"); a > 0 && i["no_amount"] != true {
				ing.Amount = &a
			}
			step.Ingredients = append(step.Ingredients, ing)
		}
		r.Steps = append(r.Steps, step)
	}
	return r
}

// --- Mealie: snake_case in older versions, camelCase in newer ones ---

func isMealie(m map[string]any) bool {
	if _, ok := m["recipe_ingredient"]; ok {
		return true
	}
	list, ok := m["recipeIngredient"].([]any)
	if !ok || m["@type"] != nil {
		return false
	}
	if len(list) == 0 {
		return m["slug"] != nil
	}
	_, objects := list[0].(map[string]any)
	return objects
}

func fromMealie(m map[string]any) *FileRecipe {
	r := &FileRecipe{Format: "mealie", Name: str(m, "name"), Description: str(m, "description"), Source: str(m, "org_url", "orgURL"),
		Keywords: names(field(m, "recipe_category", "recipeCategory"), m["tags"])}
	yield := str(m, "recipe_yield", "recipeYield")
	r.Servings = int64(number(m, "recipe_servings", "recipeServings"))
	if r.Servings == 0 {
		r.Servings = firstNumber(yield)
	}
	r.WorkMinutes, r.WaitMinutes = minutes(str(m, "prep_time", "prepTime")), minutes(str(m, "perform_time", "performTime", "cook_time", "cookTime"))
	if r.WorkMinutes == 0 && r.WaitMinutes == 0 {
		r.WorkMinutes = minutes(str(m, "total_time", "totalTime"))
	}
	ingredients := []FileIngredient{}
	for _, raw := range asList(field(m, "recipe_ingredient", "recipeIngredient")) {
		i, ok := raw.(map[string]any)
		if !ok {
			if s, isText := raw.(string); isText && strings.TrimSpace(s) != "" {
				ingredients = append(ingredients, FileIngredient{Line: s})
			}
			continue
		}
		if food := nameOf(i["food"]); food != "" {
			ing := FileIngredient{Food: food, Unit: nameOf(i["unit"]), Note: str(i, "note")}
			if a := number(i, "quantity"); a > 0 {
				ing.Amount = &a
			}
			ingredients = append(ingredients, ing)
		} else if line := mealieLine(i); line != "" {
			ingredients = append(ingredients, FileIngredient{Line: line})
		}
	}
	for _, raw := range asList(field(m, "recipe_instructions", "recipeInstructions")) {
		st, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if text := str(st, "text"); text != "" {
			r.Steps = append(r.Steps, FileStep{Name: str(st, "title"), Instruction: text})
		}
	}
	notes := []string{}
	for _, raw := range asList(m["notes"]) {
		if n, ok := raw.(map[string]any); ok {
			if text := str(n, "text"); text != "" {
				notes = append(notes, text)
			}
		}
	}
	r.Description = joinText(r.Description, strings.Join(notes, "\n"))
	if len(r.Steps) == 0 {
		r.Steps = []FileStep{{}}
	}
	r.Steps[0].Ingredients = ingredients
	return r
}

// mealieLine reads an ingredient that has no separate food. Mealie's "display" repeats the
// quantity in front of the note ("200 200 g Linsen"), so it only helps when the note has none.
func mealieLine(i map[string]any) string {
	if line := str(i, "original_text", "originalText"); line != "" {
		return line
	}
	note := str(i, "note")
	if note != "" && startsNumRe.MatchString(note) {
		return note
	}
	if line := str(i, "display"); line != "" {
		return line
	}
	return note
}

// --- Paprika: plain text, with the photo inside ---

func isPaprika(m map[string]any) bool {
	_, ingredients := m["ingredients"].(string)
	_, directions := m["directions"].(string)
	return ingredients && directions
}

func fromPaprika(m map[string]any) *FileRecipe {
	yield := str(m, "servings")
	r := &FileRecipe{Format: "paprika", Name: str(m, "name"), Description: joinText(str(m, "description"), str(m, "notes")), Source: str(m, "source_url"),
		Servings: firstNumber(yield), WorkMinutes: minutes(str(m, "prep_time")), WaitMinutes: minutes(str(m, "cook_time")), Keywords: names(m["categories"])}
	if r.WorkMinutes == 0 && r.WaitMinutes == 0 {
		r.WorkMinutes = minutes(str(m, "total_time"))
	}
	if r.Source == "" {
		r.Source = str(m, "source")
	}
	ingredients := []FileIngredient{}
	for _, line := range strings.Split(str(m, "ingredients"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			ingredients = append(ingredients, FileIngredient{Line: line})
		}
	}
	for _, p := range splitParagraphs(clean(str(m, "directions"))) {
		r.Steps = append(r.Steps, FileStep{Instruction: p})
	}
	if len(r.Steps) == 0 {
		r.Steps = []FileStep{{}}
	}
	r.Steps[0].Ingredients = ingredients
	if raw := str(m, "photo_data"); raw != "" {
		if photo, err := base64.StdEncoding.DecodeString(raw); err == nil {
			r.Photo = photo
		}
	}
	return r
}

// --- schema.org: the recipe data websites publish, and Nextcloud Cookbook's files ---

func fromSchema(node map[string]any) *FileRecipe {
	p := readRecipe(node)
	r := &FileRecipe{Format: "schema.org", Name: p.Name, Description: p.Description, Servings: p.Servings, WorkMinutes: p.WorkMinutes, WaitMinutes: p.WaitMinutes, Keywords: p.Keywords}
	if u := str(node, "url"); strings.HasPrefix(u, "http") {
		r.Source = u
	}
	ingredients := []FileIngredient{}
	for _, line := range p.Ingredients {
		ingredients = append(ingredients, FileIngredient{Line: line})
	}
	for _, st := range p.Steps {
		r.Steps = append(r.Steps, FileStep{Name: st.Name, Instruction: st.Text})
	}
	if len(r.Steps) == 0 {
		r.Steps = []FileStep{{}}
	}
	r.Steps[0].Ingredients = ingredients
	return r
}

func joinText(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "\n\n" + b
}
