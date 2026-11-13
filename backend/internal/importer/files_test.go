package importer

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type entry struct {
	name string
	data []byte
}

func makeZip(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func read(t *testing.T, data []byte) []FileRecipe {
	t.Helper()
	out, err := ReadFile(bytes.NewReader(data), int64(len(data)), "x")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const tandoorJSON = `{"name":"Gulasch","description":"Deftig","keywords":[{"name":"Rind","description":""}],"working_time":20,"waiting_time":90,"internal":true,"nutrition":null,"servings":6,"servings_text":"Portionen","source_url":"https://example.com/g",
"steps":[{"name":"Anbraten","instruction":"Braten.","time":10,"order":0,"show_as_header":false,"show_ingredients_table":true,"ingredients":[
{"food":{"name":"Rindfleisch","plural_name":null,"ignore_shopping":false,"supermarket_category":null},"unit":{"name":"g","plural_name":null,"description":""},"amount":"500.000","note":"","order":0,"is_header":false,"no_amount":false,"always_use_plural_unit":false,"always_use_plural_food":false}]},
{"name":"","instruction":"Schmoren.","time":80,"order":1,"ingredients":[{"food":{"name":"Salz"},"unit":null,"amount":0,"note":"nach Geschmack","order":0,"is_header":false,"no_amount":true},{"food":null,"unit":null,"amount":0,"note":"Für die Soße","order":1,"is_header":true,"no_amount":true},{"food":{"name":"Paprika"},"unit":{"name":"tbsp"},"amount":2.5,"note":"","order":2,"is_header":false,"no_amount":false}]}]}`

func TestTandoorExportIsAZipOfZips(t *testing.T) {
	inner := makeZip(t, entry{"recipe.json", []byte(tandoorJSON)}, entry{"image.jpg", []byte("JPEGDATA")})
	got := read(t, makeZip(t, entry{"12.zip", inner}, entry{"13.zip", makeZip(t, entry{"recipe.json", []byte(`{"name":"Ohne Bild","steps":[],"working_time":0,"keywords":[]}`)})}))
	if len(got) != 2 {
		t.Fatalf("recipes: %+v", got)
	}
	r := got[0]
	if r.Format != "tandoor" || r.Name != "Gulasch" || r.Servings != 6 || r.WorkMinutes != 20 || r.WaitMinutes != 90 || r.Source != "https://example.com/g" || r.Keywords[0] != "Rind" || string(r.Photo) != "JPEGDATA" {
		t.Fatalf("recipe: %+v", r)
	}
	if len(r.Steps) != 2 || r.Steps[0].Name != "Anbraten" || r.Steps[1].Minutes != 80 {
		t.Fatalf("steps: %+v", r.Steps)
	}
	second := r.Steps[1].Ingredients
	if len(second) != 2 || second[0].Food != "Salz" || second[0].Amount != nil || second[0].Note != "nach Geschmack" || *second[1].Amount != 2.5 || second[1].Unit != "tbsp" {
		t.Fatalf("ingredients, without the header row: %+v", second)
	}
	if got[1].Photo != nil {
		t.Fatal("a recipe's photo must not come from another recipe")
	}
}

// Tandoor's export writes amounts as text in some versions.
func TestTandoorAmountsAsText(t *testing.T) {
	got := read(t, makeZip(t, entry{"1.zip", makeZip(t, entry{"recipe.json", []byte(tandoorJSON)})}))
	if a := got[0].Steps[0].Ingredients[0].Amount; a == nil || *a != 500 {
		t.Fatalf("amount: %v", a)
	}
}

func TestMealieOldAndNewFieldNames(t *testing.T) {
	old := `{"name":"Curry","slug":"curry","description":"Scharf","recipe_yield":"4 servings","total_time":"1 hour 30 minutes","org_url":"https://e.com/c","tags":[{"name":"Indisch"}],"recipe_category":[{"name":"Dinner"}],
	"recipe_ingredient":[{"title":null,"note":"","unit":{"name":"cup"},"food":{"name":"rice"},"quantity":2.0},{"note":"1 pinch of salt","unit":null,"food":null,"quantity":1,"disable_amount":true}],
	"recipe_instructions":[{"title":"Start","text":"Cook."},{"title":"","text":"Eat."}],"notes":[{"title":"","text":"Better next day."}]}`
	got := read(t, makeZip(t, entry{"recipes/curry/curry.json", []byte(old)}, entry{"recipes/curry/images/original.webp", []byte("WEBP")}, entry{"recipes/curry/images/min-original.webp", []byte("no")}))
	r := got[0]
	if r.Format != "mealie" || r.Servings != 4 || r.WorkMinutes != 90 || r.Source != "https://e.com/c" || string(r.Photo) != "WEBP" || len(r.Keywords) != 2 || !strings.Contains(r.Description, "Better next day") {
		t.Fatalf("old: %+v", r)
	}
	ings := r.Steps[0].Ingredients
	if len(ings) != 2 || ings[0].Food != "rice" || *ings[0].Amount != 2 || ings[1].Line != "1 pinch of salt" || r.Steps[0].Name != "Start" {
		t.Fatalf("old ingredients: %+v", r.Steps)
	}
	modern := `{"name":"Soup","slug":"soup","recipeServings":3,"prepTime":"10 minutes","performTime":"PT20M","orgURL":"https://e.com/s","recipeIngredient":[{"quantity":1,"unit":{"name":"l"},"food":{"name":"water"},"note":"cold","display":"1 l water cold"}],"recipeInstructions":[{"text":"Boil."}]}`
	r = read(t, []byte(modern))[0]
	if r.Format != "mealie" || r.Servings != 3 || r.WorkMinutes != 10 || r.WaitMinutes != 20 || r.Steps[0].Ingredients[0].Note != "cold" {
		t.Fatalf("new: %+v", r)
	}
}

// The ingredients below have the exact fields of a real Mealie v3.30 export.
func TestMealieIngredientsWithoutASeparateFood(t *testing.T) {
	real := `{"name":"Echte Linsensuppe","slug":"echte-linsensuppe","description":"Wärmend.","recipe_servings":4.0,"recipe_yield":"4 Portionen","recipe_yield_quantity":0.0,"prep_time":"15 minutes","perform_time":"30 minutes","cook_time":null,"org_url":"https://example.com/linsen","tags":[],"recipe_category":[],
	"recipe_ingredient":[{"quantity":200.0,"unit":null,"food":null,"referenced_recipe":null,"note":"200 g rote Linsen","display":"200 200 g rote Linsen","title":null,"original_text":"200 g rote Linsen","substitutions":[]},
	{"quantity":1.0,"unit":null,"food":null,"note":"1 Zwiebel","display":"1 1 Zwiebel","title":null,"original_text":null},
	{"quantity":2.0,"unit":null,"food":null,"note":"Eier","display":"2 Eier","title":null,"original_text":null},
	{"quantity":1.0,"unit":{"name":"cup","abbreviation":"c"},"food":{"name":"rice"},"note":"","display":"1 cup rice","original_text":null}],
	"recipe_instructions":[{"id":"x","title":"Anbraten","summary":"","text":"Zwiebel anbraten.","ingredient_references":[]},{"title":"","text":"Linsen 20 Minuten kochen."}],"notes":[{"title":"Tipp","text":"Mit Zitrone."}]}`
	r := read(t, makeZip(t, entry{"recipes/echte-linsensuppe/echte-linsensuppe.json", []byte(real)}))[0]
	if r.Servings != 4 || r.WorkMinutes != 15 || r.WaitMinutes != 30 {
		t.Fatalf("fields: %+v", r)
	}
	got := r.Steps[0].Ingredients
	for i, want := range []string{"200 g rote Linsen", "1 Zwiebel", "2 Eier"} {
		if got[i].Line != want {
			t.Errorf("line %d: %q, want %q", i, got[i].Line, want)
		}
	}
	if got[3].Food != "rice" || *got[3].Amount != 1 || got[3].Unit != "cup" {
		t.Fatalf("structured: %+v", got[3])
	}
}

func paprikaEntry(t *testing.T, m map[string]any) []byte {
	t.Helper()
	raw, _ := json.Marshal(m)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(raw)
	_ = w.Close()
	return buf.Bytes()
}

func TestPaprikaIsAZipOfGzippedJSON(t *testing.T) {
	item := paprikaEntry(t, map[string]any{"name": "Brot", "servings": "1 Laib", "prep_time": "15 min", "cook_time": "1 hr", "description": "Knusprig", "notes": "Heiß servieren", "nutritional_info": "", "source": "Oma", "source_url": "https://e.com/b",
		"categories": []string{"Backen"}, "ingredients": "500 g Mehl\n\n1 TL Salz\n", "directions": "Kneten.\n\nBacken.", "photo_data": base64.StdEncoding.EncodeToString([]byte("PHOTO")), "image_url": nil})
	got := read(t, makeZip(t, entry{"Brot.paprikarecipe", item}, entry{"broken.paprikarecipe", []byte("not gzip")}))
	if len(got) != 1 {
		t.Fatalf("a broken entry must not stop the others: %+v", got)
	}
	r := got[0]
	if r.Format != "paprika" || r.Servings != 1 || r.WorkMinutes != 15 || r.WaitMinutes != 60 || string(r.Photo) != "PHOTO" || r.Source != "https://e.com/b" || !strings.Contains(r.Description, "Heiß") {
		t.Fatalf("recipe: %+v", r)
	}
	if len(r.Steps) != 2 || len(r.Steps[0].Ingredients) != 2 || r.Steps[0].Ingredients[0].Line != "500 g Mehl" {
		t.Fatalf("steps: %+v", r.Steps)
	}
	single := read(t, paprikaEntry(t, map[string]any{"name": "Allein", "ingredients": "x", "directions": "y"}))
	if single[0].Name != "Allein" {
		t.Fatalf("one gzipped recipe: %+v", single)
	}
}

func TestSchemaOrgFilesAndNextcloud(t *testing.T) {
	nc := `{"@context":"http://schema.org","@type":"Recipe","name":"Salat","url":"https://e.com/n","recipeYield":2,"recipeIngredient":["1 Gurke"],"recipeInstructions":["Schneiden.","Mischen."],"keywords":"frisch,leicht"}`
	got := read(t, makeZip(t, entry{"Salat/recipe.json", []byte(nc)}, entry{"Salat/full.jpg", []byte("FULL")}, entry{"Salat/thumb.jpg", []byte("no")}, entry{"settings.json", []byte(`{"theme":"dark"}`)}, entry{"__MACOSX/x.json", []byte(nc)}))
	if len(got) != 1 || got[0].Format != "schema.org" || string(got[0].Photo) != "FULL" || got[0].Servings != 2 || len(got[0].Steps) != 2 || got[0].Source != "https://e.com/n" {
		t.Fatalf("nextcloud: %+v", got)
	}
	list := read(t, []byte(`[{"@type":"Recipe","name":"A","recipeIngredient":["x"]},{"@graph":[{"@type":"WebSite"},{"@type":"Recipe","name":"B","recipeIngredient":["y"]}]},{"not":"a recipe"}]`))
	if len(list) != 2 || list[1].Name != "B" {
		t.Fatalf("list: %+v", list)
	}
}

func TestFilesWithoutRecipesAreRefused(t *testing.T) {
	for name, data := range map[string][]byte{"text": []byte("hello"), "json": []byte(`{"a":1}`), "broken zip": []byte("PK\x03\x04broken"), "empty zip": makeZip(t), "bad gzip": {0x1f, 0x8b, 0, 0}, "empty": {}} {
		if _, err := ReadFile(bytes.NewReader(data), int64(len(data)), "x"); !errors.Is(err, ErrFileUnknown) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestZipBombsStop(t *testing.T) {
	// 70 MB of zeros compress to almost nothing. The entry limit has to catch it.
	big := makeZip(t, entry{"bomb.json", make([]byte, maxEntryBytes+1024)})
	if len(big) > 1<<20 {
		t.Fatalf("test zip is too large to prove anything: %d", len(big))
	}
	if _, err := ReadFile(bytes.NewReader(big), int64(len(big)), "x"); !errors.Is(err, ErrFileTooBig) {
		t.Fatalf("bomb: %v", err)
	}
	// Nesting stops too: a zip inside a zip inside a zip inside a zip isn't followed.
	deep := makeZip(t, entry{"r.json", []byte(`{"@type":"Recipe","name":"Deep","recipeIngredient":["x"]}`)})
	for range 4 {
		deep = makeZip(t, entry{"n.zip", deep})
	}
	if _, err := ReadFile(bytes.NewReader(deep), int64(len(deep)), "x"); !errors.Is(err, ErrFileUnknown) {
		t.Fatalf("nesting: %v", err)
	}
}

func TestWokkoExportsKeepEverything(t *testing.T) {
	doc := `{"format":"wokko-export-1","recipes":[
	  {"name":"Linsensuppe","description":"Warm","servings":3,"servings_text":"bowls","work_minutes":15,"wait_minutes":30,
	   "source_url":"https://example.com/soup","keywords":[{"name":"soup"},{"name":"vegan"}],"photo":"photos/a.jpg",
	   "steps":[{"name":"Prep","instruction":"Chop.","minutes":5,"ingredients":[
	      {"food":{"name":"onion"},"unit":null,"amount":2,"note":"red"},
	      {"food":{"name":"lentils"},"unit":{"name":"gram"},"amount":200,"note":""}]},
	     {"name":"","instruction":"Cook.","minutes":25,"ingredients":[{"food":{"name":"salt"},"unit":null,"amount":null,"note":""}]}]},
	  {"name":"Old export","servings":2,"photo_url":"/media/b.jpg?share=x","steps":[]}]}`
	recipes := read(t, makeZip(t, entry{"recipes.json", []byte(doc)}, entry{"photos/a.jpg", []byte("A")}, entry{"photos/b.jpg", []byte("B")}))
	if len(recipes) != 2 {
		t.Fatalf("recipes: %+v", recipes)
	}
	r := recipes[0]
	if r.Format != "Wokko" || r.Servings != 3 || r.ServingsText != "bowls" || r.WorkMinutes != 15 || r.WaitMinutes != 30 || r.Source != "https://example.com/soup" || string(r.Photo) != "A" {
		t.Fatalf("recipe: %+v", r)
	}
	if strings.Join(r.Keywords, ",") != "soup,vegan" || len(r.Steps) != 2 || r.Steps[0].Name != "Prep" || r.Steps[0].Minutes != 5 || r.Steps[1].Minutes != 25 {
		t.Fatalf("keywords or steps: %+v", r)
	}
	onion, lentils, salt := r.Steps[0].Ingredients[0], r.Steps[0].Ingredients[1], r.Steps[1].Ingredients[0]
	if onion.Food != "onion" || onion.Unit != "" || *onion.Amount != 2 || onion.Note != "red" || lentils.Unit != "gram" || *lentils.Amount != 200 || salt.Amount != nil {
		t.Fatalf("ingredients: %+v %+v %+v", onion, lentils, salt)
	}
	// Exports from 0.2 and 0.3 only point at where the photo was served.
	if old := recipes[1]; string(old.Photo) != "B" || len(old.Steps) != 1 {
		t.Fatalf("old export: %+v", old)
	}
	// A photo path can't reach outside the archive's folder.
	evil := `{"format":"wokko-export-1","recipes":[{"name":"x","photo":"../secret.jpg","steps":[]}]}`
	if got := read(t, makeZip(t, entry{"in/recipes.json", []byte(evil)}, entry{"secret.jpg", []byte("S")})); got[0].Photo != nil {
		t.Fatal("photo outside the folder was read")
	}
	// A recipe file on its own, without photos, works too.
	if got := read(t, []byte(doc)); len(got) != 2 || got[0].Photo != nil {
		t.Fatalf("plain JSON: %+v", got)
	}
}
