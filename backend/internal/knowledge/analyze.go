package knowledge

import (
	"fmt"
	"sort"
	"strings"

	"github.com/serraniel/wokko/backend/internal/text"
)

// Ingredient is a food as a recipe uses it. Names holds every name the food has, in the
// viewer's language first, so the first one is what people see.
type Ingredient struct {
	Names   []string `json:"names"`
	Classes []string `json:"classes,omitempty"` // filled in by Analyze when empty
	Amount  *float64 `json:"amount,omitempty"`
	Unit    string   `json:"unit,omitempty"`
}

type Step struct {
	Text        string       `json:"text"`
	Minutes     int64        `json:"minutes,omitempty"`
	Ingredients []Ingredient `json:"ingredients"`
}

type Recipe struct {
	Name        string `json:"name,omitempty"`
	WorkMinutes int64  `json:"work_minutes,omitempty"`
	WaitMinutes int64  `json:"wait_minutes,omitempty"`
	Steps       []Step `json:"steps"`
}

type span struct{ from, to int }

type run struct {
	p    *Pack
	r    Recipe
	text []string
	memo map[string]bool
	busy map[string]bool
}

// Analyze reads a recipe with the pack. It never changes the recipe.
func (p *Pack) Analyze(r Recipe, locale string) Result {
	for i := range r.Steps {
		for j := range r.Steps[i].Ingredients {
			ing := &r.Steps[i].Ingredients[j]
			if len(ing.Classes) == 0 {
				ing.Classes = p.ClassesOf(ing.Names)
			}
		}
	}
	x := &run{p: p, r: r, memo: map[string]bool{}, busy: map[string]bool{}}
	for _, st := range r.Steps {
		x.text = append(x.text, " "+text.Normalize(st.Text)+" ")
	}
	res := Result{Sauces: []Finding{}, Techniques: []Finding{}, Problems: []Finding{}, Safety: []Finding{}, Substitutes: []Finding{}, Allergens: []FoundAllergen{}, Unclassified: []string{}, Diet: Diet{Because: []string{}}}
	if len(r.Steps) > 0 {
		all := span{0, len(r.Steps) - 1}
		for i := range p.Entries {
			e := &p.Entries[i]
			if e.Check != "" {
				continue
			}
			var foods []string
			if e.For != "" {
				foods = x.foodsOf(e.For)
				if len(foods) == 0 || (!e.Detect.empty() && !x.entry(e.ID, all)) {
					continue
				}
			} else if !x.entry(e.ID, all) {
				continue
			}
			f := Finding{ID: e.ID, Kind: e.Kind, Parent: e.Parent, Severity: e.Severity, Title: e.Names.pick(locale), Text: e.Explain.pick(locale), Tips: []string{}, Steps: []int{}, Foods: []string{}, Sources: e.Sources}
			if foods != nil {
				f.Foods = foods
			}
			for _, t := range e.Tips {
				f.Tips = append(f.Tips, t.pick(locale))
			}
			for k := range r.Steps {
				if e.For == "" && x.entry(e.ID, span{k, k}) {
					f.Steps = append(f.Steps, k+1)
				}
			}
			switch e.Kind {
			case "sauce":
				res.Sauces = append(res.Sauces, f)
			case "technique":
				res.Techniques = append(res.Techniques, f)
			case "safety":
				res.Safety = append(res.Safety, f)
			case "substitution":
				res.Substitutes = append(res.Substitutes, f)
			default:
				res.Problems = append(res.Problems, f)
			}
		}
	}
	res.Problems = append(res.Problems, p.timeCheck(r, locale)...)
	p.foods(r, locale, &res)
	return res
}

func (x *run) entry(id string, s span) bool {
	key := fmt.Sprintf("%s@%d-%d", id, s.from, s.to)
	if v, ok := x.memo[key]; ok {
		return v
	}
	e := x.p.entries[id]
	if e == nil || x.busy[key] {
		return false // A loop in the data. The tests catch it, and here it just doesn't match.
	}
	x.busy[key] = true
	v := x.cond(e.Detect, s)
	delete(x.busy, key)
	x.memo[key] = v
	return v
}

func (x *run) cond(c Cond, s span) bool {
	if c.Class != "" && !x.hasClass(c.Class, s) {
		return false
	}
	if len(c.Words) > 0 && !x.hasWord(c.Words, s) {
		return false
	}
	if c.Entry != "" && !x.entry(c.Entry, s) {
		return false
	}
	if c.MinMinutes > 0 && !x.longStep(c.MinMinutes, s) {
		return false
	}
	if c.Not != nil && x.cond(*c.Not, s) {
		return false
	}
	if len(c.Any) > 0 {
		found := false
		for _, a := range c.Any {
			if x.cond(a, s) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.All) > 0 && !x.all(c.All, c.Window, s) {
		return false
	}
	if len(c.Then) > 0 && !x.then(c.Then, s) {
		return false
	}
	return true
}

func (x *run) all(list []Cond, window int, s span) bool {
	ok := func(in span) bool {
		for _, c := range list {
			if !x.cond(c, in) {
				return false
			}
		}
		return true
	}
	if window <= 0 || window > s.to-s.from+1 {
		return ok(s)
	}
	for k := s.from; k+window-1 <= s.to; k++ {
		if ok(span{k, k + window - 1}) {
			return true
		}
	}
	return false
}

// then finds each condition in a later step than the previous one, within the span.
func (x *run) then(list []Cond, s span) bool {
	next := s.from
	for i, c := range list {
		found := -1
		for k := next; k <= s.to; k++ {
			if x.cond(c, span{k, k}) {
				found = k
				break
			}
		}
		if found < 0 {
			return false
		}
		if i < len(list)-1 {
			next = found + 1
		}
	}
	return true
}

func (x *run) hasClass(want string, s span) bool {
	for k := s.from; k <= s.to; k++ {
		for _, ing := range x.r.Steps[k].Ingredients {
			for _, c := range ing.Classes {
				if isA(c, want) {
					return true
				}
			}
		}
	}
	return false
}

func (x *run) hasWord(words []string, s span) bool {
	for k := s.from; k <= s.to; k++ {
		for _, w := range words {
			if strings.Contains(x.text[k], w) {
				return true
			}
		}
	}
	return false
}

func (x *run) longStep(min int64, s span) bool {
	for k := s.from; k <= s.to; k++ {
		if x.r.Steps[k].Minutes >= min {
			return true
		}
	}
	return false
}

// timeCheck compares the steps' minutes with the recipe's stated time.
func (p *Pack) timeCheck(r Recipe, locale string) []Finding {
	var sum int64
	for _, st := range r.Steps {
		sum += st.Minutes
	}
	total := r.WorkMinutes + r.WaitMinutes
	if total == 0 || sum <= total+15 || float64(sum) <= float64(total)*1.25 {
		return nil
	}
	e := p.entries["problem.times_disagree"]
	if e == nil {
		return nil
	}
	return []Finding{{ID: e.ID, Kind: e.Kind, Severity: e.Severity, Title: e.Names.pick(locale),
		Text: strings.NewReplacer("{steps}", fmt.Sprint(sum), "{total}", fmt.Sprint(total)).Replace(e.Explain.pick(locale)),
		Tips: []string{}, Steps: []int{}, Foods: []string{}, Sources: e.Sources}}
}

// foods works out allergens and diets from the classes of every ingredient.
func (p *Pack) foods(r Recipe, locale string, res *Result) {
	found := map[string]*FoundAllergen{}
	maybeFoods := map[string][]string{}
	add := func(id, food string, maybe bool) {
		if maybe {
			if !contains(maybeFoods[id], food) {
				maybeFoods[id] = append(maybeFoods[id], food)
			}
			return
		}
		if found[id] == nil {
			found[id] = &FoundAllergen{ID: id, Foods: []string{}}
		}
		if !contains(found[id].Foods, food) {
			found[id].Foods = append(found[id].Foods, food)
		}
	}
	vegetarian, vegan, known := true, true, true
	seen := map[string]bool{}
	for _, st := range r.Steps {
		for _, ing := range st.Ingredients {
			name := ""
			if len(ing.Names) > 0 {
				name = ing.Names[0]
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			if len(ing.Classes) == 0 {
				known = false
				res.Unclassified = append(res.Unclassified, name)
				continue
			}
			notVeg, notVegan := false, false
			for _, c := range ing.Classes {
				for _, k := range p.lineage(c) {
					if k.DietUnknown && !hasDietClass(p, ing.Classes) {
						known = false
					}
					notVeg = notVeg || k.NotVegetarian
					notVegan = notVegan || k.NotVegan || k.NotVegetarian
					for _, a := range k.Allergens {
						add(a, name, false)
					}
					for _, a := range k.MayContain {
						add(a, name, true)
					}
				}
			}
			if notVeg {
				vegetarian = false
			}
			if notVegan {
				vegan = false
				res.Diet.Because = append(res.Diet.Because, name)
			}
		}
	}
	// A food that surely contains an allergen settles it; otherwise products that may contain it
	// are listed so people know to check the label.
	for id, foods := range maybeFoods {
		if found[id] == nil {
			found[id] = &FoundAllergen{ID: id, Foods: foods, Maybe: true}
		}
	}
	for _, a := range p.Allergens {
		if f := found[a.ID]; f != nil {
			f.Name = a.Names.pick(locale)
			res.Allergens = append(res.Allergens, *f)
		}
	}
	sort.SliceStable(res.Allergens, func(i, j int) bool {
		if res.Allergens[i].Maybe != res.Allergens[j].Maybe {
			return !res.Allergens[i].Maybe
		}
		return res.Allergens[i].Name < res.Allergens[j].Name
	})
	res.Diet.Vegetarian = verdict(vegetarian, known)
	res.Diet.Vegan = verdict(vegan, known)
}

// verdict says no as soon as one food rules it out, but yes only when every food is known.
func verdict(ok, known bool) *bool {
	if !ok {
		f := false
		return &f
	}
	if !known {
		return nil
	}
	t := true
	return &t
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// hasDietClass tells whether a more specific class settles the diet, as "chicken stock" does
// for "stock".
func hasDietClass(p *Pack, classes []string) bool {
	for _, c := range classes {
		k := p.classes[c]
		if k != nil && !k.DietUnknown && strings.HasPrefix(c, "liquid.stock.") {
			return true
		}
	}
	return false
}

// foodsOf names the recipe's ingredients of a class, each once.
func (x *run) foodsOf(class string) []string {
	var out []string
	for _, st := range x.r.Steps {
		for _, ing := range st.Ingredients {
			for _, c := range ing.Classes {
				if isA(c, class) && len(ing.Names) > 0 && !contains(out, ing.Names[0]) {
					out = append(out, ing.Names[0])
				}
			}
		}
	}
	return out
}

// empty is true for a condition that says nothing.
func (c Cond) empty() bool {
	return c.Class == "" && len(c.Words) == 0 && c.Entry == "" && len(c.All) == 0 && len(c.Any) == 0 && c.Not == nil && len(c.Then) == 0 && c.MinMinutes == 0
}
