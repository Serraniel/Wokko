// Package catalog is the starter catalog of foods, units and tags that Wokko ships, and the
// rules for bringing a newer one into an existing database without overwriting anyone's work.
//
// Every entry has a stable key. The database remembers, per entry, the values it last took from
// the pack, so a three-way merge can tell what the pack changed from what people changed.
package catalog

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/serraniel/wokko/backend/internal/text"
)

//go:embed data/*.json data/original/*.json
var dataFS embed.FS

// Kinds of entries.
const (
	Food = "food"
	Unit = "unit"
	Tag  = "tag"
)

// Name is one language's name for an entry.
type Name struct {
	Name         string `json:"name"`
	Plural       string `json:"plural,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty"`
}

// Entry is one food, unit or tag.
type Entry struct {
	Key string `json:"key"`
	// ID is the row the original seed migration created, for entries that came with it.
	ID          int64              `json:"id,omitempty"`
	DefaultUnit string             `json:"default_unit,omitempty"`
	Grams       *float64           `json:"grams,omitempty"`
	Names       map[string]Name    `json:"names"`
	Aliases     []string           `json:"aliases,omitempty"`
	Nutrients   map[string]float64 `json:"nutrients,omitempty"`
	// Curated marks values people improved by hand, which a generator must keep.
	Curated bool `json:"curated,omitempty"`
}

// Pack is the catalog this build ships, and the original seed it grew from.
type Pack struct {
	Entries  map[string][]Entry // by kind
	Original map[string]Entry   // by kind/key
	Hash     string             // changes whenever the shipped catalog does
}

type file struct {
	Foods []Entry `json:"foods"`
	Units []Entry `json:"units"`
	Tags  []Entry `json:"tags"`
}

func read(name string) (file, error) {
	raw, err := dataFS.ReadFile(name)
	if err != nil {
		return file{}, err
	}
	var f file
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return file{}, fmt.Errorf("%s: %w", name, err)
	}
	return f, nil
}

func (f file) byKind() map[string][]Entry {
	return map[string][]Entry{Food: f.Foods, Unit: f.Units, Tag: f.Tags}
}

// Load reads the shipped pack.
func Load() (*Pack, error) {
	p := &Pack{Entries: map[string][]Entry{}, Original: map[string]Entry{}}
	h := sha256.New()
	for _, kind := range []string{"foods", "units", "tags"} {
		cur, err := read("data/" + kind + ".json")
		if err != nil {
			return nil, err
		}
		orig, err := read("data/original/" + kind + ".json")
		if err != nil {
			return nil, err
		}
		for k, list := range cur.byKind() {
			p.Entries[k] = append(p.Entries[k], list...)
		}
		for k, list := range orig.byKind() {
			for _, e := range list {
				p.Original[k+"/"+e.Key] = e
			}
		}
		raw, _ := dataFS.ReadFile("data/" + kind + ".json")
		h.Write(raw)
	}
	p.Hash = hex.EncodeToString(h.Sum(nil))[:16]
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

// Fields flattens an entry into named values, so a merge can work field by field. A missing
// field and an empty one are the same.
type Fields map[string]string

// FieldsOf flattens an entry. Units in default_unit stay keys, so they mean the same in every
// database.
func FieldsOf(e Entry) Fields {
	f := Fields{}
	for locale, n := range e.Names {
		f["name/"+locale] = EncodeName(n)
	}
	for _, a := range e.Aliases {
		if a = strings.TrimSpace(a); a != "" {
			f["alias/"+text.Normalize(a)] = a
		}
	}
	if e.DefaultUnit != "" {
		f["default_unit"] = e.DefaultUnit
	}
	if e.Grams != nil {
		f["grams"] = Number(*e.Grams)
	}
	for id, v := range e.Nutrients {
		f["nutrient/"+id] = Number(v)
	}
	return f
}

// EncodeName writes a name the same way every time, so equal names compare equal.
func EncodeName(n Name) string {
	raw, _ := json.Marshal(n)
	return string(raw)
}

// DecodeName reads EncodeName's output.
func DecodeName(s string) Name {
	var n Name
	_ = json.Unmarshal([]byte(s), &n)
	return n
}

// Number writes a value the same way every time.
func Number(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// Hash identifies a set of values, such as what a suggestion proposes.
func (f Fields) Hash() string {
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, f[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Change is a field where the pack and the database disagree and neither side is the base.
type Change struct {
	Field  string `json:"field"`
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
}

// Result of a merge: what to write, what to ask about, and the new base to remember.
type Result struct {
	Apply     Fields
	Conflicts []Change
	Base      Fields
}

// Merge compares the base (what the database last took from the pack), ours (the database now)
// and theirs (the pack now), field by field:
//
//   - nobody changed it, or both changed it the same way: nothing to do
//   - only the pack changed it: take the pack's value, if auto is true, otherwise ask
//   - only people changed it: keep theirs, which is ours
//   - both changed it differently: ask
//
// A field the pack no longer has is never removed from the database. Nothing here deletes.
func Merge(base, ours, theirs Fields, auto bool) Result {
	r := Result{Apply: Fields{}, Base: Fields{}}
	keys := map[string]bool{}
	for _, f := range []Fields{base, ours, theirs} {
		for k := range f {
			keys[k] = true
		}
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		b, o, t := base[k], ours[k], theirs[k]
		switch {
		case t == "":
			// Dropped from the pack, or never in it. Keep whatever people have.
		case o == t:
			r.Base[k] = t
		case o == b && auto:
			r.Apply[k] = t
			r.Base[k] = t
		case t == b:
			r.Base[k] = t // People changed it; the pack didn't.
		default:
			r.Conflicts = append(r.Conflicts, Change{Field: k, Ours: o, Theirs: t})
			if b != "" {
				r.Base[k] = b // Stays open until someone decides.
			}
		}
	}
	return r
}
