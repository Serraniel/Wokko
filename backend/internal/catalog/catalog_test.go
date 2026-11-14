package catalog

import (
	"testing"
)

func TestMergeFieldByField(t *testing.T) {
	base := Fields{"name/en": "onion", "name/de": "Zwiebel", "nutrient/1": "40", "nutrient/2": "0.1", "alias/old": "old"}
	ours := Fields{"name/en": "onion", "name/de": "Küchenzwiebel", "nutrient/1": "40", "nutrient/2": "0.2", "alias/old": "old", "alias/mine": "mine"}
	theirs := Fields{"name/en": "yellow onion", "name/de": "Zwiebel", "name/fr": "oignon", "nutrient/1": "41", "nutrient/2": "0.3"}
	r := Merge(base, ours, theirs, true)

	// Only the pack changed these: take them.
	want := Fields{"name/en": "yellow onion", "name/fr": "oignon", "nutrient/1": "41"}
	if len(r.Apply) != len(want) {
		t.Fatalf("apply: %v", r.Apply)
	}
	for k, v := range want {
		if r.Apply[k] != v {
			t.Errorf("apply %s = %q, want %q", k, r.Apply[k], v)
		}
	}
	// Both changed nutrient 2 differently: ask. Someone's German name stays, since the pack didn't change it.
	if len(r.Conflicts) != 1 || r.Conflicts[0] != (Change{Field: "nutrient/2", Ours: "0.2", Theirs: "0.3"}) {
		t.Fatalf("conflicts: %+v", r.Conflicts)
	}
	if r.Base["nutrient/2"] != "0.1" || r.Base["name/de"] != "Zwiebel" || r.Base["name/en"] != "yellow onion" {
		t.Errorf("base: %v", r.Base)
	}
	// Dropped from the pack and added by people: both stay untouched.
	if _, ok := r.Apply["alias/old"]; ok {
		t.Error("never remove")
	}
	if _, ok := r.Apply["alias/mine"]; ok {
		t.Error("never touch what people added")
	}

	// A second run with the new base finds nothing new, only the open question.
	ours2 := Fields{}
	for k, v := range ours {
		ours2[k] = v
	}
	for k, v := range r.Apply {
		ours2[k] = v
	}
	again := Merge(r.Base, ours2, theirs, true)
	if len(again.Apply) != 0 || len(again.Conflicts) != 1 {
		t.Errorf("again: %+v", again)
	}
}

func TestPeoplesOwnEntriesOnlyGetSuggestions(t *testing.T) {
	r := Merge(Fields{}, Fields{"name/en": "my onion"}, Fields{"name/en": "my onion", "name/de": "Zwiebel"}, false)
	if len(r.Apply) != 0 || len(r.Conflicts) != 1 || r.Conflicts[0].Field != "name/de" {
		t.Fatalf("%+v", r)
	}
}

func TestPackLoadsAndMatchesTheOriginalSeed(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Entries[Food]) < 163 || len(p.Entries[Unit]) < 31 || len(p.Entries[Tag]) < 25 {
		t.Fatalf("pack: %d foods, %d units, %d tags", len(p.Entries[Food]), len(p.Entries[Unit]), len(p.Entries[Tag]))
	}
	keys := map[string]bool{}
	units := map[string]bool{}
	for _, u := range p.Entries[Unit] {
		units[u.Key] = true
	}
	for kind, list := range p.Entries {
		for _, e := range list {
			if keys[kind+"/"+e.Key] {
				t.Errorf("%s %s appears twice", kind, e.Key)
			}
			keys[kind+"/"+e.Key] = true
			if e.Names["en"].Name == "" {
				t.Errorf("%s %s has no English name", kind, e.Key)
			}
			if e.DefaultUnit != "" && !units[e.DefaultUnit] {
				t.Errorf("%s %s: unknown unit %s", kind, e.Key, e.DefaultUnit)
			}
			for id, v := range e.Nutrients {
				// Per 100 g: no part can weigh more than the whole, and nothing has more energy than fat.
				if v < 0 || (id != "1" && v > 100) || (id == "1" && v > 900) {
					t.Errorf("%s: nutrient %s = %v is implausible", e.Key, id, v)
				}
			}
		}
	}
	// Every original entry stays in the pack, so nothing can be dropped by accident.
	for k, o := range p.Original {
		if !keys[k] {
			t.Errorf("%s is in the original seed but not in the pack", k)
		}
		if o.ID == 0 {
			t.Errorf("%s: original entries need their seed id", k)
		}
	}
}
