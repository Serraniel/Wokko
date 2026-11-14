// Command catalog-refresh adds names and aliases from Wikidata to the starter catalog's foods.
//
// It only adds: a missing name in a language, or another spelling as an alias. It never
// changes or removes a value, skips entries marked curated, and refuses a name that already
// belongs to another entry. Items are found by English name and remembered in the entry, so
// each one is looked up once and a person can correct it. The result is meant for a pull
// request that a person reviews; see .github/workflows/catalog-refresh.yml.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/serraniel/wokko/backend/internal/catalog"
	"github.com/serraniel/wokko/backend/internal/knowledge"
	"github.com/serraniel/wokko/backend/internal/text"
)

const api = "https://www.wikidata.org/w/api.php"

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	dir := flag.String("dir", "internal/catalog/data", "the catalog's data directory")
	langs := flag.String("languages", "", "languages to fill, comma separated (default: the app's languages)")
	messages := flag.String("messages", "../frontend/src/lib/i18n/messages", "where the app's languages are, to find them")
	summary := flag.String("summary", "", "write a Markdown summary of the changes here")
	limit := flag.Int("aliases", 4, "aliases to add per entry and language at most")
	format := flag.Bool("format", false, "only rewrite the data files in their canonical form")
	flag.Parse()
	if *format {
		if err := formatFiles(*dir); err != nil {
			fmt.Fprintln(os.Stderr, "catalog-refresh:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(*dir, *langs, *messages, *summary, *limit); err != nil {
		fmt.Fprintln(os.Stderr, "catalog-refresh:", err)
		os.Exit(1)
	}
}

func run(dir, langList, messages, summaryPath string, limit int) error {
	languages, err := pickLanguages(langList, messages)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "foods.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	byKind, err := catalog.Decode(raw)
	if err != nil {
		return err
	}
	foods := byKind[catalog.Food]

	// Every spelling already in use, so nothing is added that would mean two foods.
	taken := map[string]string{}
	for _, f := range foods {
		for _, n := range f.Names {
			taken[text.Normalize(n.Name)] = f.Key
			taken[text.Normalize(n.Plural)] = f.Key
		}
		for _, a := range f.Aliases {
			taken[text.Normalize(a)] = f.Key
		}
	}
	delete(taken, "")

	pack, err := knowledge.Load()
	if err != nil {
		return err
	}
	var log []string
	for i := range foods {
		f := &foods[i]
		if f.Curated || f.Wikidata != "" {
			continue
		}
		candidates, err := find(f.Names["en"].Name)
		if err != nil {
			return err
		}
		found, err := fetch(candidates, []string{"en"}, true)
		if err != nil {
			return err
		}
		// The first exact match that Wikidata says is something to eat. Homonyms, such as a
		// singer called Mango or the town of Orange, aren't. Neither is a genus that merely
		// carries the food's name, as the insects called Semolina do.
		for _, id := range candidates {
			if it, ok := found[id]; ok && it.edible() && !strings.EqualFold(it.taxon(), f.Names["en"].Name) {
				f.Wikidata = id
				log = append(log, fmt.Sprintf("- `%s`: linked to [%s](https://www.wikidata.org/wiki/%s)", f.Key, id, id))
				break
			}
		}
	}
	var ids []string
	for _, f := range foods {
		if f.Wikidata != "" && !f.Curated {
			ids = append(ids, f.Wikidata)
		}
	}
	items, err := fetch(ids, languages, true)
	if err != nil {
		return err
	}
	added := 0
	for i := range foods {
		f := &foods[i]
		it, ok := items[f.Wikidata]
		if f.Curated || !ok {
			continue
		}
		// An alias must not read as another kind of food than the entry, as "pepper" would for
		// bell pepper, and must not be the species' Latin name.
		var names []string
		for _, n := range f.Names {
			names = append(names, n.Name, n.Plural)
		}
		own := pack.ClassesOf(names)
		fits := func(alias string) bool {
			if latin(alias, it.taxon()) {
				return false
			}
			kinds := pack.ClassesOf([]string{alias})
			if len(kinds) == 0 {
				return true
			}
			for _, k := range kinds {
				if slices.Contains(own, k) {
					return true
				}
			}
			return false
		}
		for _, lang := range languages {
			if label := clean(it.Labels[lang].Value); label != "" && f.Names[lang].Name == "" && free(taken, label, f.Key) {
				f.Names[lang] = catalog.Name{Name: label}
				taken[text.Normalize(label)] = f.Key
				log = append(log, fmt.Sprintf("- `%s`: %s name **%s**", f.Key, lang, label))
				added++
			}
			n := 0
			for _, a := range it.Aliases[lang] {
				alias := clean(a.Value)
				if n >= limit || alias == "" || !free(taken, alias, f.Key) || taken[text.Normalize(alias)] == f.Key || !fits(alias) {
					continue
				}
				f.Aliases = append(f.Aliases, alias)
				taken[text.Normalize(alias)] = f.Key
				log = append(log, fmt.Sprintf("- `%s`: %s alias **%s**", f.Key, lang, alias))
				added++
				n++
			}
		}
	}
	out, err := catalog.Encode(catalog.Food, foods)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	fmt.Printf("%d names and aliases added, %d lines in the summary\n", added, len(log))
	if summaryPath != "" {
		body := "Names and aliases from [Wikidata](https://www.wikidata.org) (CC0) for the starter catalog's foods. Please check that each linked item is the right food and that each alias means the same thing. Nothing was changed or removed.\n\n"
		if len(log) == 0 {
			body += "Nothing new this time.\n"
		} else {
			body += strings.Join(log, "\n") + "\n"
		}
		return os.WriteFile(summaryPath, []byte(body), 0o644)
	}
	return nil
}

// pickLanguages uses the given list, or every language the app has messages for.
func pickLanguages(list, messages string) ([]string, error) {
	if list != "" {
		return strings.Split(list, ","), nil
	}
	files, err := filepath.Glob(filepath.Join(messages, "*.json"))
	if err != nil || len(files) == 0 {
		return nil, errors.New("no languages: pass -languages or -messages")
	}
	var out []string
	for _, f := range files {
		out = append(out, strings.TrimSuffix(filepath.Base(f), ".json"))
	}
	sort.Strings(out)
	return out, nil
}

// noise marks aliases that describe the plant or a dataset rather than the food.
var noise = []string{"plant", "pflanze", "fruit", "frucht", "dataset", "species", "uk", "us", "cultivar", "sorte", "tree", "baum", "strauch", "bush", "shrub"}

// clean keeps plain names: no qualifiers in brackets, no codes or abbreviations, nothing overlong.
func clean(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 40 || len(strings.Fields(s)) > 3 || strings.ContainsAny(s, "()[]/:;,.") {
		return ""
	}
	for _, w := range strings.Fields(strings.ToLower(s)) {
		if slices.Contains(noise, w) || strings.HasSuffix(w, "baum") || strings.HasSuffix(w, "strauch") {
			return ""
		}
	}
	// Only letters, spaces, hyphens and apostrophes, and no abbreviations such as "PB".
	upper := 0
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			upper++
			if upper > 1 {
				return ""
			}
		case unicode.IsLetter(r):
			upper = 0
		case r == ' ' || r == '-' || r == '\'':
			upper = 0
		default:
			return ""
		}
	}
	for _, r := range s {
		if unicode.IsDigit(r) {
			return ""
		}
	}
	return s
}

// latin tells a scientific name, such as "Solanum tuberosum", from a food's name.
func latin(s, taxon string) bool {
	words := strings.Fields(s)
	if taxon != "" && len(words) > 0 && strings.EqualFold(words[0], strings.Fields(taxon)[0]) {
		return true
	}
	if len(words) < 2 || len(words) > 3 {
		return false
	}
	for i, w := range words {
		for j, r := range w {
			if r > unicode.MaxASCII || !unicode.IsLetter(r) || (j == 0 && (i == 0) != unicode.IsUpper(r)) || (j > 0 && unicode.IsUpper(r)) {
				return false
			}
		}
	}
	// Two or three plain words, only the first capitalised: the shape of a binomial. German
	// food names capitalise every noun, so they don't look like this.
	return true
}

// free tells whether a name can go to the entry: no other food uses it, or a form of it that
// differs only by an ending, as "Cherrytomate" does from the cherry tomatoes' "Cherrytomaten".
func free(taken map[string]string, name, key string) bool {
	n := text.Normalize(name)
	if owner, ok := taken[n]; ok {
		return owner == key
	}
	for word, owner := range taken {
		if owner == key {
			continue
		}
		short, long := word, n
		if len(short) > len(long) {
			short, long = long, short
		}
		if len(short) >= 4 && strings.HasPrefix(long, short) && len(long)-len(short) <= 2 {
			return false
		}
	}
	return true
}

func get(params url.Values, out any) error {
	params.Set("format", "json")
	params.Set("maxlag", "5")
	req, err := http.NewRequest("GET", api+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Wokko-catalog-refresh/1 (https://github.com/Serraniel/Wokko)")
	time.Sleep(200 * time.Millisecond) // Be polite to a free service.
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("wikidata: %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// find returns the items whose English label is exactly the name. An item that only has the
// name as an alias, or a different label, is too uncertain to take on its own.
func find(name string) ([]string, error) {
	if name == "" {
		return nil, nil
	}
	var res struct {
		Search []struct {
			ID    string `json:"id"`
			Match struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"match"`
		} `json:"search"`
	}
	err := get(url.Values{"action": {"wbsearchentities"}, "search": {name}, "language": {"en"}, "type": {"item"}, "limit": {"7"}}, &res)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, s := range res.Search {
		if s.Match.Type == "label" && strings.EqualFold(s.Match.Text, name) {
			out = append(out, s.ID)
		}
	}
	return out, nil
}

type value struct {
	Value string `json:"value"`
}

type item struct {
	Labels  map[string]value   `json:"labels"`
	Aliases map[string][]value `json:"aliases"`
	Claims  map[string][]struct {
		Mainsnak struct {
			Datavalue struct {
				Value json.RawMessage `json:"value"`
			} `json:"datavalue"`
		} `json:"mainsnak"`
	} `json:"claims"`
}

// Kinds of item that are something to eat, or a species grown for it.
var edibleKinds = []string{
	"Q2095",     // food
	"Q10675206", // ingredient
	"Q11004",    // vegetable
	"Q1364",     // fruit
	"Q3314483",  // fruit, as food
	"Q42527",    // spice
	"Q207123",   // herb
	"Q16521",    // taxon, such as Allium cepa
	"Q4886",     // cultivar
	"Q12117",    // cereal
	"Q11396",    // nut
	"Q145909",   // legume
	"Q10943",    // cheese
	"Q8495",     // milk
	"Q2477654",  // dairy product
	"Q10990",    // meat
	"Q152",      // fish
	"Q1047113",  // condiment
	"Q178359",   // sauce
	"Q1124086",  // mushroom
	"Q1150958",  // food additive
	"Q28803",    // sugar
	"Q8486",     // coffee
}

func (it item) ids(prop string) []string {
	var out []string
	for _, c := range it.Claims[prop] {
		var v struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(c.Mainsnak.Datavalue.Value, &v) == nil && v.ID != "" {
			out = append(out, v.ID)
		}
	}
	return out
}

func (it item) edible() bool {
	for _, id := range append(it.ids("P31"), it.ids("P279")...) {
		if slices.Contains(edibleKinds, id) {
			return true
		}
	}
	return false
}

// taxon is the item's scientific name, if it is a species.
func (it item) taxon() string {
	for _, c := range it.Claims["P225"] {
		var name string
		if json.Unmarshal(c.Mainsnak.Datavalue.Value, &name) == nil {
			return name
		}
	}
	return ""
}

func fetch(ids []string, languages []string, claims bool) (map[string]item, error) {
	out := map[string]item{}
	for len(ids) > 0 {
		batch := ids[:min(50, len(ids))]
		ids = ids[len(batch):]
		var res struct {
			Entities map[string]item `json:"entities"`
		}
		props := "labels|aliases"
		if claims {
			props += "|claims"
		}
		params := url.Values{"action": {"wbgetentities"}, "ids": {strings.Join(batch, "|")}, "props": {props}, "languages": {strings.Join(languages, "|")}}
		if err := get(params, &res); err != nil {
			return nil, err
		}
		for id, it := range res.Entities {
			if slices.Contains(batch, id) {
				out[id] = it
			}
		}
	}
	return out, nil
}

// formatFiles rewrites every data file as Encode writes it.
func formatFiles(dir string) error {
	for name, kind := range catalog.Files {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries, err := catalog.Decode(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out, err := catalog.Encode(kind, entries[kind])
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
	}
	return nil
}
