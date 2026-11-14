// Package knowledge holds Wokko's built-in kitchen knowledge and reads recipes with it.
//
// The knowledge is curated data in data/, reviewed by people and shipped in the binary. Nothing
// here learns at runtime or calls a model: food classes come from word lists, and sauces,
// techniques and problems from declarative conditions over a recipe's foods and steps.
package knowledge

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/serraniel/wokko/backend/internal/text"
)

//go:embed data/*.json data/examples/*.json
var dataFS embed.FS

// Text is one piece of text in several languages, keyed by locale.
type Text map[string]string

func (t Text) pick(locale string) string {
	if s := t[locale]; s != "" {
		return s
	}
	return t["en"]
}

// Class is a kind of food, such as dairy.cream. A class implies its parents, so dairy.cream is
// also dairy, and it inherits their allergens and diet flags.
type Class struct {
	ID            string              `json:"id"`
	Names         Text                `json:"names,omitempty"`
	Words         map[string][]string `json:"words,omitempty"` // per locale, matched against food names
	Allergens     []string            `json:"allergens,omitempty"`
	MayContain    []string            `json:"may_contain,omitempty"` // depends on the product, such as celery in stock cubes
	NotVegetarian bool                `json:"not_vegetarian,omitempty"`
	NotVegan      bool                `json:"not_vegan,omitempty"`
	// DietUnknown marks foods that can go either way, such as plain "stock".
	DietUnknown bool `json:"diet_unknown,omitempty"`
}

// Allergen is one of the allergens Wokko points out.
type Allergen struct {
	ID    string `json:"id"`
	Names Text   `json:"names"`
}

// Cond is a condition over a recipe, or over a range of its steps.
type Cond struct {
	Class      string   `json:"class,omitempty"`       // an ingredient of this class, or a child class
	Words      []string `json:"words,omitempty"`       // a step's text contains one of these, normalized
	Entry      string   `json:"entry,omitempty"`       // another entry is found here
	All        []Cond   `json:"all,omitempty"`         // every one holds
	Any        []Cond   `json:"any,omitempty"`         // at least one holds
	Not        *Cond    `json:"not,omitempty"`         // doesn't hold
	Window     int      `json:"window,omitempty"`      // with all: within this many consecutive steps
	Then       []Cond   `json:"then,omitempty"`        // each holds in a later step than the one before
	MinMinutes int64    `json:"min_minutes,omitempty"` // a step takes at least this long
}

// Entry is one piece of knowledge: a sauce, a technique or a problem rule.
type Entry struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"` // sauce, technique or problem
	Parent   string   `json:"parent,omitempty"`
	Severity string   `json:"severity,omitempty"` // problems: info or warning
	Names    Text     `json:"names"`
	Explain  Text     `json:"explain"`
	Tips     []Text   `json:"tips,omitempty"`
	Detect   Cond     `json:"detect"`
	Check    string   `json:"check,omitempty"` // a check done in code instead of by Detect
	Sources  []Source `json:"sources"`
	Examples struct {
		Match   []string `json:"match"`
		NoMatch []string `json:"no_match"`
	} `json:"examples"`
}

// Pack is all the knowledge Wokko ships.
type Pack struct {
	Classes   []Class
	Allergens []Allergen
	Entries   []Entry
	Examples  map[string]Recipe

	classes map[string]*Class
	entries map[string]*Entry
	words   []classWord
}

type classWord struct {
	word  string
	class string
	// compound is true for languages that join words, where "Hühnerbrühe" ends in "brühe".
	compound bool
}

// compounding lists languages that write compound nouns as one word.
var compounding = map[string]bool{"de": true, "nl": true, "da": true, "sv": true, "nb": true, "no": true, "fi": true, "is": true}

type file struct {
	Classes   []Class    `json:"classes"`
	Allergens []Allergen `json:"allergens"`
	Entries   []Entry    `json:"entries"`
}

// Load reads the shipped pack.
func Load() (*Pack, error) {
	p := &Pack{Examples: map[string]Recipe{}, classes: map[string]*Class{}, entries: map[string]*Entry{}}
	names, err := fs.Glob(dataFS, "data/*.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := dataFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var f file
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&f); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		p.Classes = append(p.Classes, f.Classes...)
		p.Allergens = append(p.Allergens, f.Allergens...)
		p.Entries = append(p.Entries, f.Entries...)
	}
	examples, err := fs.Glob(dataFS, "data/examples/*.json")
	if err != nil {
		return nil, err
	}
	for _, name := range examples {
		raw, err := dataFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var r Recipe
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		p.Examples[strings.TrimSuffix(path.Base(name), ".json")] = r
	}
	for i := range p.Classes {
		c := &p.Classes[i]
		if p.classes[c.ID] != nil {
			return nil, fmt.Errorf("class %s appears twice", c.ID)
		}
		p.classes[c.ID] = c
		for locale, list := range c.Words {
			for _, w := range list {
				p.words = append(p.words, classWord{text.Normalize(w), c.ID, compounding[locale]})
			}
		}
	}
	for i := range p.Entries {
		e := &p.Entries[i]
		if p.entries[e.ID] != nil {
			return nil, fmt.Errorf("entry %s appears twice", e.ID)
		}
		p.entries[e.ID] = e
		normalizeWords(&e.Detect)
	}
	return p, nil
}

// MustLoad is Load for start-up, where broken shipped data is a build error.
func MustLoad() *Pack {
	p, err := Load()
	if err != nil {
		panic(err)
	}
	return p
}

// ClassesOf finds the classes of a food from all its names and aliases. When one matching word
// is part of a longer one, the longer wins, so "peanut butter" isn't butter and "Kokosmilch"
// isn't milk.
func (p *Pack) ClassesOf(names []string) []string {
	type hit struct{ word, class string }
	var hits []hit
	for _, n := range names {
		n = text.Normalize(n)
		if n == "" {
			continue
		}
		for _, cw := range p.words {
			if matchesWord(n, cw.word, cw.compound) {
				hits = append(hits, hit{cw.word, cw.class})
			}
		}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, h := range hits {
		covered := false
		for _, o := range hits {
			if len(o.word) > len(h.word) && strings.Contains(o.word, h.word) {
				covered = true
				break
			}
		}
		if !covered && !seen[h.class] {
			seen[h.class] = true
			out = append(out, h.class)
		}
	}
	sort.Strings(out)
	return out
}

// matchesWord accepts the whole name, a whole word or phrase in it, or, for languages that
// build compounds, the end of a word: "Hühnerbrühe" ends with "brühe". English "unicorn" doesn't
// end in corn.
func matchesWord(name, word string, compound bool) bool {
	if name == word {
		return true
	}
	padded := " " + name + " "
	if strings.Contains(padded, " "+word+" ") {
		return true
	}
	if compound && len([]rune(word)) >= 4 && !strings.Contains(word, " ") {
		for _, token := range strings.Fields(name) {
			if strings.HasSuffix(token, word) {
				return true
			}
		}
	}
	return false
}

// isA tells whether a class is the wanted one or one of its children.
func isA(class, want string) bool {
	return class == want || strings.HasPrefix(class, want+".")
}

// lineage is a class and all its parents, which pass on allergens and diet flags.
func (p *Pack) lineage(class string) []*Class {
	var out []*Class
	for c := class; c != ""; {
		if k := p.classes[c]; k != nil {
			out = append(out, k)
		}
		i := strings.LastIndex(c, ".")
		if i < 0 {
			break
		}
		c = c[:i]
	}
	return out
}

// normalizeWords folds condition words the way step text is folded, so data can be written
// with umlauts and capitals.
func normalizeWords(c *Cond) {
	for i, w := range c.Words {
		// Keep a leading or trailing space, which marks the start or end of a word.
		lead, trail := strings.HasPrefix(w, " "), strings.HasSuffix(w, " ")
		w = text.Normalize(w)
		if lead {
			w = " " + w
		}
		if trail {
			w += " "
		}
		c.Words[i] = w
	}
	for _, list := range [][]Cond{c.All, c.Any, c.Then} {
		for i := range list {
			normalizeWords(&list[i])
		}
	}
	if c.Not != nil {
		normalizeWords(c.Not)
	}
}

// HasClass tells whether a class exists.
func (p *Pack) HasClass(id string) bool { return p.classes[id] != nil }

// ClassList lists every class with what it implies, in the viewer's language.
func (p *Pack) ClassList(locale string) []ClassInfo {
	out := make([]ClassInfo, 0, len(p.Classes))
	for _, c := range p.Classes {
		info := ClassInfo{ID: c.ID, Name: c.Names.pick(locale), Allergens: []string{}}
		if i := strings.LastIndex(c.ID, "."); i > 0 {
			if parent := p.classes[c.ID[:i]]; parent != nil {
				info.Parent = parent.Names.pick(locale)
			}
		}
		seen := map[string]bool{}
		for _, k := range p.lineage(c.ID) {
			for _, a := range k.Allergens {
				if !seen[a] {
					seen[a] = true
					info.Allergens = append(info.Allergens, a)
				}
			}
			info.NotVegetarian = info.NotVegetarian || k.NotVegetarian
			info.NotVegan = info.NotVegan || k.NotVegan || k.NotVegetarian
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
