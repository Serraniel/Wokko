package main

import "testing"

func TestCleanKeepsOnlyPlainNames(t *testing.T) {
	keep := []string{"spud", "Mohrrübe", "garden beet", "Kūmara", "avocado pear", "Gelbe Bete", "O'Brien"}
	drop := []string{"PB", "ACV", "MbMB", "🍇", "Allium cepa L.", "pomegranate tree", "Granatapfelbaum", "onion plant", "Leek UK", "SPINACH dataset", "spinach (leaf)", "E 160", "one two three four", ""}
	for _, s := range keep {
		if clean(s) != s {
			t.Errorf("%q should be kept", s)
		}
	}
	for _, s := range drop {
		if clean(s) != "" {
			t.Errorf("%q should be dropped", s)
		}
	}
}

func TestLatinNamesAreNotAliases(t *testing.T) {
	cases := map[string]bool{
		"Solanum tuberosum":         true,
		"Brassica oleracea italica": true,
		"Allium porrum":             true,
		"Gelbe Bete":                false,
		"garden beet":               false,
		"Peruanischer Apfel":        false,
		"spud":                      false,
	}
	for s, want := range cases {
		if got := latin(s, ""); got != want {
			t.Errorf("latin(%q) = %v", s, got)
		}
	}
	if !latin("Allium", "Allium cepa") {
		t.Error("the genus of the item's own taxon")
	}
}
