// Package nutrition looks up nutrition values in a public-domain dataset shipped inside Wokko.
// Nothing is fetched: the data is USDA FoodData Central SR Legacy (2018-04, CC0), built into
// data/sr_legacy.json.gz by cmd/nutrition-data.
package nutrition

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/serraniel/wokko/backend/internal/text"
)

// Source names the dataset, for showing where values came from.
const Source = "USDA FoodData Central SR Legacy"

//go:embed data/sr_legacy.json.gz
var packed []byte

// Item is one food from the dataset with its values per 100 g, keyed by Wokko's nutrient ids.
type Item struct {
	ID          int64             `json:"id"`
	Description string            `json:"d"`
	Values      map[int64]float64 `json:"n"`
	words       []string
	penalty     float64
}

// variant words mark a special version of a food, which a plain search shouldn't prefer.
var variant = []string{"imitation", "nonfat", "free", "reduced", "low", "lean", "substitute", "dietetic", "fortified"}

// rate prepares an item for searching: its words without the notes in brackets, and how much
// it looks like a branded or special product rather than the plain food.
func rate(it *Item) {
	plain := it.Description
	for {
		i := strings.Index(plain, "(")
		j := strings.Index(plain, ")")
		if i < 0 || j < i {
			break
		}
		plain = plain[:i] + plain[j+1:]
	}
	it.words = words(plain)
	for _, w := range it.words {
		if slices.Contains(variant, w) {
			it.penalty += 15
		}
	}
	for _, w := range strings.Fields(it.Description) {
		w = strings.Trim(w, ",.")
		if len(w) >= 3 && w == strings.ToUpper(w) && strings.ToLower(w) != w {
			it.penalty += 40 // a brand name, such as LEAN POCKETS
			break
		}
	}
}

var (
	once  sync.Once
	items []Item
	err   error
)

func load() ([]Item, error) {
	once.Do(func() {
		zr, e := gzip.NewReader(bytes.NewReader(packed))
		if e != nil {
			err = e
			return
		}
		if e := json.NewDecoder(zr).Decode(&items); e != nil {
			err = e
			return
		}
		for i := range items {
			rate(&items[i])
		}
	})
	return items, err
}

func words(s string) []string {
	return strings.FieldsFunc(text.Normalize(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// has tells whether a description word is the search word or its plural.
func has(desc []string, w string) bool {
	for _, d := range desc {
		if d == w || d == w+"s" || d == w+"es" || strings.TrimSuffix(w, "s") == d {
			return true
		}
	}
	return false
}

// Match is a dataset food that fits a search.
type Match struct {
	Item
	score float64
}

// Search finds the dataset foods that best fit any of the names, such as a food's names in
// every language and its aliases. Foods that contain every word of a name come first, and the
// shorter, plainer description wins among them ("Cheese, cheddar" before "Cheese, cheddar, sharp,
// sliced").
func Search(names []string, limit int) ([]Item, error) {
	all, err := load()
	if err != nil {
		return nil, err
	}
	var terms [][]string
	for _, n := range names {
		var ws []string
		for _, w := range words(n) {
			if len([]rune(w)) >= 2 {
				ws = append(ws, w)
			}
		}
		if len(ws) > 0 {
			terms = append(terms, ws)
		}
	}
	if len(terms) == 0 {
		return []Item{}, nil
	}
	var found []Match
	for _, it := range all {
		best := 0.0
		for _, ws := range terms {
			matched := 0
			for _, w := range ws {
				if has(it.words, w) {
					matched++
				}
			}
			if matched == 0 {
				continue
			}
			score := 100 * float64(matched) / float64(len(ws))
			if matched == len(ws) {
				score += 100
			}
			if len(it.words) > 0 && has(it.words[:1], ws[0]) {
				score += 20 // the description starts with it: "Ham, sliced" for ham
			}
			if has(it.words, "raw") {
				score += 3
			}
			score -= float64(len(it.words)) + it.penalty
			if score > best {
				best = score
			}
		}
		if best > 0 {
			found = append(found, Match{it, best})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].ID < found[j].ID
	})
	if len(found) > limit {
		found = found[:limit]
	}
	out := make([]Item, len(found))
	for i, m := range found {
		out[i] = m.Item
		out[i].words, out[i].penalty = nil, 0
	}
	return out, nil
}
