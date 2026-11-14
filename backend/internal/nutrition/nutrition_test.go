package nutrition

import (
	"strings"
	"testing"
)

func TestTheDatasetIsComplete(t *testing.T) {
	all, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 7000 {
		t.Fatalf("only %d foods", len(all))
	}
	for _, it := range all {
		if it.Description == "" || it.Values[1] < 0 || it.Values[1] > 902 {
			t.Fatalf("implausible: %+v", it)
		}
		for n, v := range it.Values {
			if n != 1 && (v < 0 || v > 100) {
				t.Fatalf("%s: nutrient %d is %v per 100 g", it.Description, n, v)
			}
		}
	}
}

func TestSearchFindsThePlainFoodFirst(t *testing.T) {
	cases := map[string]string{
		"cheddar":     "Cheese, cheddar",
		"cooked ham":  "Ham",
		"onion":       "Onions, raw",
		"butter":      "Butter",
		"white bread": "Bread, white",
	}
	for q, want := range cases {
		got, err := Search([]string{q}, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) == 0 || !strings.HasPrefix(got[0].Description, want) {
			t.Errorf("%s: %v", q, descriptions(got))
		}
	}
	// A food's names in other languages simply don't match, and the English one still does.
	got, _ := Search([]string{"Kochschinken", "ham"}, 3)
	if len(got) == 0 || !strings.HasPrefix(got[0].Description, "Ham") {
		t.Errorf("mixed names: %v", descriptions(got))
	}
	if got, _ := Search([]string{"Dingsbums"}, 3); len(got) != 0 {
		t.Errorf("nothing fits: %v", descriptions(got))
	}
}

func descriptions(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Description)
	}
	return out
}
