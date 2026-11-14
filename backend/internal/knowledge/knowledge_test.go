package knowledge

import (
	"slices"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/text"
)

var languages = []string{"en", "de"}

func load(t *testing.T) *Pack {
	t.Helper()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPackIsComplete checks the shipped data: every text exists in every language, every
// reference points somewhere, and every entry names a source and has examples.
func TestPackIsComplete(t *testing.T) {
	p := load(t)
	allergens := map[string]bool{}
	for _, a := range p.Allergens {
		allergens[a.ID] = true
		for _, l := range languages {
			if a.Names[l] == "" {
				t.Errorf("allergen %s has no %s name", a.ID, l)
			}
		}
	}
	if len(p.Allergens) != 14 {
		t.Errorf("the EU lists 14 allergens, the pack has %d", len(p.Allergens))
	}
	for _, c := range p.Classes {
		for _, l := range languages {
			if c.Names[l] == "" {
				t.Errorf("class %s has no %s name", c.ID, l)
			}
		}
		for _, a := range append(slices.Clone(c.Allergens), c.MayContain...) {
			if !allergens[a] {
				t.Errorf("class %s names unknown allergen %s", c.ID, a)
			}
		}
		if i := strings.LastIndex(c.ID, "."); i > 0 && p.classes[c.ID[:i]] == nil {
			t.Errorf("class %s has no parent class %s", c.ID, c.ID[:i])
		}
	}
	used := map[string]bool{}
	for _, e := range p.Entries {
		if e.Kind != "sauce" && e.Kind != "technique" && e.Kind != "problem" && e.Kind != "safety" {
			t.Errorf("%s: unknown kind %q", e.ID, e.Kind)
		}
		if (e.Kind == "problem" || e.Kind == "safety") && e.Severity != "info" && e.Severity != "warning" {
			t.Errorf("%s: a problem needs severity info or warning", e.ID)
		}
		if !strings.HasPrefix(e.ID, e.Kind+".") {
			t.Errorf("%s: the id should start with %s.", e.ID, e.Kind)
		}
		if e.Parent != "" && p.entries[e.Parent] == nil {
			t.Errorf("%s: unknown parent %s", e.ID, e.Parent)
		}
		for _, l := range languages {
			if e.Names[l] == "" || e.Explain[l] == "" {
				t.Errorf("%s: missing %s text", e.ID, l)
			}
			for i, tip := range e.Tips {
				if tip[l] == "" {
					t.Errorf("%s: tip %d has no %s text", e.ID, i+1, l)
				}
			}
		}
		if len(e.Sources) == 0 {
			t.Errorf("%s: name a source", e.ID)
		}
		if e.Check == "" && len(e.Examples.Match) == 0 {
			t.Errorf("%s: needs at least one example it matches", e.ID)
		}
		checkCond(t, p, e.ID, e.Detect)
		for _, x := range append(slices.Clone(e.Examples.Match), e.Examples.NoMatch...) {
			used[x] = true
			if _, ok := p.Examples[x]; !ok {
				t.Errorf("%s: no example recipe %s", e.ID, x)
			}
		}
	}
	for name := range p.Examples {
		if !used[name] {
			t.Errorf("example %s isn't used by any entry", name)
		}
	}
}

func checkCond(t *testing.T, p *Pack, id string, c Cond) {
	t.Helper()
	if c.Class != "" && p.classes[c.Class] == nil {
		t.Errorf("%s: unknown class %s", id, c.Class)
	}
	if c.Entry != "" && p.entries[c.Entry] == nil {
		t.Errorf("%s: unknown entry %s", id, c.Entry)
	}
	if c.Entry == id {
		t.Errorf("%s refers to itself", id)
	}
	for _, list := range [][]Cond{c.All, c.Any, c.Then} {
		for _, sub := range list {
			checkCond(t, p, id, sub)
		}
	}
	if c.Not != nil {
		checkCond(t, p, id, *c.Not)
	}
}

// TestEntriesMatchTheirExamples runs every entry against the recipes it names. This is what keeps
// a new entry from quietly breaking an older one.
func TestEntriesMatchTheirExamples(t *testing.T) {
	p := load(t)
	results := map[string]map[string]bool{}
	for name, r := range p.Examples {
		res := p.Analyze(r, "en")
		found := map[string]bool{}
		for _, list := range [][]Finding{res.Sauces, res.Techniques, res.Problems, res.Safety} {
			for _, f := range list {
				found[f.ID] = true
			}
		}
		results[name] = found
	}
	for _, e := range p.Entries {
		for _, x := range e.Examples.Match {
			if !results[x][e.ID] {
				t.Errorf("%s should be found in %s", e.ID, x)
			}
		}
		for _, x := range e.Examples.NoMatch {
			if results[x][e.ID] {
				t.Errorf("%s should not be found in %s", e.ID, x)
			}
		}
	}
}

// TestWordsDontCollide makes sure each word belongs to one class, so classification can't
// depend on the order of the files.
func TestWordsDontCollide(t *testing.T) {
	p := load(t)
	owner := map[string]string{}
	for _, cw := range p.words {
		if o, ok := owner[cw.word]; ok && o != cw.class {
			t.Errorf("%q belongs to both %s and %s", cw.word, o, cw.class)
		}
		owner[cw.word] = cw.class
	}
}

func TestFoodClasses(t *testing.T) {
	p := load(t)
	cases := map[string]string{
		"Hühnerbrühe":         "liquid.stock.poultry",
		"Erdnussbutter":       "peanut",
		"Kokosmilch":          "fruit.coconut",
		"Hafermilch":          "drink.plant",
		"Butter":              "fat.butter",
		"Kräuterbutter":       "fat.butter",
		"Frühlingszwiebeln":   "veg.onion",
		"Paniermehl":          "grain.gluten",
		"Weizenmehl Type 405": "grain.gluten.flour",
		"Mandelmehl":          "nut",
		"Schweinefleisch":     "meat",
		"lemon rind":          "fruit.citrus_zest",
		"peanut butter":       "peanut",
		"chicken broth":       "liquid.stock.poultry",
		"Muskatnuss":          "spice",
		"Selleriesalz":        "veg.celery",
		"Glasnudeln":          "grain.free",
	}
	for name, want := range cases {
		got := p.ClassesOf([]string{name})
		if !slices.Contains(got, want) || len(got) != 1 {
			t.Errorf("%s: got %v, want only %s", name, got, want)
		}
	}
	if got := p.ClassesOf([]string{"Dingsbums"}); len(got) != 0 {
		t.Errorf("unknown food: %v", got)
	}
	// Aliases and translations count too.
	if got := p.ClassesOf([]string{"Wokko-Spezial", "Sahne"}); !slices.Contains(got, "dairy.cream") {
		t.Errorf("alias: %v", got)
	}
}

func TestFricasseeInsights(t *testing.T) {
	p := load(t)
	res := p.Analyze(p.Examples["huehnerfrikassee"], "de")
	var allergens []string
	for _, a := range res.Allergens {
		if !a.Maybe {
			allergens = append(allergens, a.ID)
		}
	}
	slices.Sort(allergens)
	// Worcestershire sauce brings fish, which is easy to miss.
	want := []string{"celery", "eggs", "fish", "gluten", "milk"}
	if !slices.Equal(allergens, want) {
		t.Errorf("allergens: %v, want %v", allergens, want)
	}
	if res.Diet.Vegetarian == nil || *res.Diet.Vegetarian {
		t.Errorf("vegetarian: %v", res.Diet.Vegetarian)
	}
	if len(res.Problems) != 0 || len(res.Safety) != 0 {
		t.Errorf("problems: %+v %+v", res.Problems, res.Safety)
	}
	var sauce *Finding
	for i := range res.Sauces {
		if res.Sauces[i].ID == "sauce.allemande" {
			sauce = &res.Sauces[i]
		}
	}
	if sauce == nil || sauce.Title != "Sauce allemande" || !strings.Contains(sauce.Text, "Frikassee") {
		t.Fatalf("sauces: %+v", res.Sauces)
	}
	for _, f := range res.Techniques {
		if f.ID == "technique.roux" && !slices.Equal(f.Steps, []int{5}) {
			t.Errorf("roux steps: %v", f.Steps)
		}
	}
}

func TestDiets(t *testing.T) {
	p := load(t)
	dal := Recipe{Steps: []Step{{Text: "Alles 20 Minuten köcheln.", Ingredients: []Ingredient{{Names: []string{"Rote Linsen"}}, {Names: []string{"Zwiebel"}}, {Names: []string{"Kokosmilch"}}, {Names: []string{"Kurkuma"}}, {Names: []string{"Gemüsebrühe"}}}}}}
	res := p.Analyze(dal, "en")
	// Vegetable stock may contain celery, which is shown as "check the label", not as fact.
	if res.Diet.Vegan == nil || !*res.Diet.Vegan || len(res.Allergens) != 1 || !res.Allergens[0].Maybe {
		t.Errorf("dal: %+v", res)
	}
	// Plain "stock" might be made from meat, so Wokko doesn't claim anything.
	dal.Steps[0].Ingredients[4] = Ingredient{Names: []string{"Brühe"}}
	if res := p.Analyze(dal, "en"); res.Diet.Vegetarian != nil {
		t.Errorf("plain stock: %v", *res.Diet.Vegetarian)
	}
	// Traditional parmesan is made with calf rennet; honey isn't vegan.
	pesto := Recipe{Steps: []Step{{Ingredients: []Ingredient{{Names: []string{"Parmesan"}}, {Names: []string{"Honig"}}}}}}
	res = p.Analyze(pesto, "en")
	if res.Diet.Vegetarian == nil || *res.Diet.Vegetarian || !slices.Contains(res.Diet.Because, "Parmesan") {
		t.Errorf("parmesan: %+v", res.Diet)
	}
	unknown := Recipe{Steps: []Step{{Ingredients: []Ingredient{{Names: []string{"Dingsbums"}}, {Names: []string{"Zwiebel"}}}}}}
	res = p.Analyze(unknown, "en")
	if res.Diet.Vegetarian != nil || !slices.Equal(res.Unclassified, []string{"Dingsbums"}) {
		t.Errorf("unknown: %+v", res)
	}
}

func TestTimesDisagree(t *testing.T) {
	p := load(t)
	r := Recipe{WorkMinutes: 20, Steps: []Step{{Text: "Kochen", Minutes: 60}}}
	res := p.Analyze(r, "en")
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0].Text, "60 minutes") || !strings.Contains(res.Problems[0].Text, "20") {
		t.Errorf("times: %+v", res.Problems)
	}
	r.WorkMinutes = 55
	if res := p.Analyze(r, "en"); len(res.Problems) != 0 {
		t.Errorf("close enough: %+v", res.Problems)
	}
}

func TestWordsAreNormalized(t *testing.T) {
	p := load(t)
	for _, cw := range p.words {
		if cw.word != text.Normalize(cw.word) {
			t.Errorf("%q", cw.word)
		}
	}
}
