package api_test

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/catalog"
)

// newerCatalog copies the shipped catalog and lets a test change it like a later release would.
func newerCatalog(p *catalog.Pack, hash string, change func(kind string, e *catalog.Entry), add ...catalog.Entry) *catalog.Pack {
	out := &catalog.Pack{Entries: map[string][]catalog.Entry{}, Original: p.Original, Hash: hash}
	for kind, list := range p.Entries {
		for _, e := range list {
			c := e
			c.Names = map[string]catalog.Name{}
			for k, v := range e.Names {
				c.Names[k] = v
			}
			c.Nutrients = map[string]float64{}
			for k, v := range e.Nutrients {
				c.Nutrients[k] = v
			}
			c.Aliases = slices.Clone(e.Aliases)
			change(kind, &c)
			out.Entries[kind] = append(out.Entries[kind], c)
		}
	}
	out.Entries[catalog.Food] = append(out.Entries[catalog.Food], add...)
	return out
}

func foodNamed(t *testing.T, c *client, q string) app.Food {
	t.Helper()
	var page app.Page[app.Food]
	c.ok("GET", v1+"/foods?q="+url.QueryEscape(q), nil, 200).decode(t, &page)
	for _, f := range page.Items {
		for _, tr := range f.Translations {
			if strings.EqualFold(tr.Name, q) {
				return f
			}
		}
	}
	t.Fatalf("no food %q in %+v", q, page.Items)
	return app.Food{}
}

func TestStarterCatalogUpdatesReachExistingDatabases(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	admin := e.admin()

	// The first run adopts the seed this database already has, without changing anything.
	first, err := e.svc.SyncCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.Added != 0 || first.Updated != 0 || first.Suggested != 0 {
		t.Fatalf("first: %+v", first)
	}

	// People work with the catalog: the admin renames onion in German and deletes tempeh, and
	// someone adds their own food that a later catalog will also bring.
	onion := foodNamed(t, admin, "onion")
	admin.ok("PUT", path("/foods/%d", onion.ID), app.FoodInput{Translations: []app.Translation{{Locale: "en", Name: "onion", Plural: "onions"}, {Locale: "de", Name: "Küchenzwiebel"}}}, 200)
	tempeh := foodNamed(t, admin, "tempeh")
	admin.ok("DELETE", path("/foods/%d", tempeh.ID), nil, 200)
	cook := e.newUser(admin, "cook")
	var berry app.Food
	cook.ok("POST", v1+"/foods", app.FoodInput{Translations: []app.Translation{{Locale: "de", Name: "Wokkobeere"}}}, 201).decode(t, &berry)

	// A newer catalog changes onion's German name and adds French, corrects garlic's energy,
	// still has tempeh, adds a new food, and one that someone already has.
	energy := 0.0
	e.svc.Catalog = newerCatalog(e.svc.Catalog, "newer", func(kind string, c *catalog.Entry) {
		switch c.Key {
		case "food.onion":
			c.Names["de"] = catalog.Name{Name: "Speisezwiebel", Plural: "Speisezwiebeln"}
			c.Names["fr"] = catalog.Name{Name: "oignon", Plural: "oignons"}
		case "food.garlic":
			energy = c.Nutrients["1"] + 1
			c.Nutrients["1"] = energy
		}
	},
		catalog.Entry{Key: "food.wokkonut", Names: map[string]catalog.Name{"en": {Name: "wokko nut"}, "de": {Name: "Wokkonuss"}}, Aliases: []string{"wokkonuts"}, Nutrients: map[string]float64{"1": 600}},
		catalog.Entry{Key: "food.wokkoberry", Names: map[string]catalog.Name{"en": {Name: "wokko berry"}, "de": {Name: "Wokkobeere"}}},
	)
	got, err := e.svc.SyncCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Added != 1 || got.Updated != 2 {
		t.Errorf("sync: %+v", got)
	}

	// Onion gains French, but keeps the German name someone chose; that becomes a question.
	onion = foodNamed(t, admin, "onion")
	langs := map[string]string{}
	for _, tr := range onion.Translations {
		langs[tr.Locale] = tr.Name
	}
	if langs["fr"] != "oignon" || langs["de"] != "Küchenzwiebel" {
		t.Errorf("onion: %v", langs)
	}
	nut := foodNamed(t, admin, "wokko nut")
	if !slices.Contains(nut.Aliases, "wokkonuts") || nut.CanEdit != true {
		t.Errorf("new food: %+v", nut)
	}
	// Tempeh stays deleted.
	var page app.Page[app.Food]
	admin.ok("GET", v1+"/foods?q=tempeh", nil, 200).decode(t, &page)
	if page.Total != 0 {
		t.Errorf("tempeh came back: %+v", page.Items)
	}
	// The person's own berry wasn't duplicated or changed; its missing English name is a question.
	admin.ok("GET", v1+"/foods?q=wokkobeere", nil, 200).decode(t, &page)
	if page.Total != 1 || len(page.Items[0].Translations) != 1 {
		t.Errorf("berry: %+v", page.Items)
	}

	var status app.CatalogStatus
	admin.ok("GET", v1+"/admin/catalog", nil, 200).decode(t, &status)
	if status.Last == nil || status.Last.Pack != "newer" {
		t.Errorf("last: %+v", status.Last)
	}
	byField := map[string]app.CatalogSuggestion{}
	for _, s := range status.Suggestions {
		byField[s.Key+" "+s.Field] = s
	}
	de, ok := byField["food.onion name/de"]
	if !ok || de.Ours != "Küchenzwiebel" || de.Theirs != "Speisezwiebel / Speisezwiebeln" || de.Name != "onion" {
		t.Fatalf("suggestions: %+v", status.Suggestions)
	}
	en, ok := byField["food.wokkoberry name/en"]
	if !ok || en.Ours != "" || en.Theirs != "wokko berry" {
		t.Fatalf("berry suggestion: %+v", status.Suggestions)
	}
	if len(status.Suggestions) != 2 {
		t.Errorf("only two questions expected: %+v", status.Suggestions)
	}

	// Only admins decide.
	cook.wantCode("GET", v1+"/admin/catalog", nil, 403, "forbidden")
	cook.wantCode("POST", path("/admin/catalog/suggestions/%d", de.ID), map[string]bool{"take": true}, 403, "forbidden")

	admin.ok("POST", path("/admin/catalog/suggestions/%d", de.ID), map[string]bool{"take": true}, 200)
	admin.ok("POST", path("/admin/catalog/suggestions/%d", en.ID), map[string]bool{"take": false}, 200)
	onion = foodNamed(t, admin, "onion")
	for _, tr := range onion.Translations {
		if tr.Locale == "de" && tr.Name != "Speisezwiebel" {
			t.Errorf("taken: %+v", onion.Translations)
		}
	}
	admin.ok("GET", v1+"/admin/catalog", nil, 200).decode(t, &status)
	if len(status.Suggestions) != 0 {
		t.Errorf("resolved: %+v", status.Suggestions)
	}

	// The same catalog isn't processed twice, and a later one doesn't ask the same questions again.
	again, err := e.svc.SyncCatalog(ctx)
	if err != nil || again.Pack != "newer" || again.Added != 1 {
		t.Errorf("again: %+v %v", again, err)
	}
	e.svc.Catalog.Hash = "newest"
	later, err := e.svc.SyncCatalog(ctx)
	if err != nil || later.Added != 0 || later.Updated != 0 || later.Suggested != 0 {
		t.Errorf("later: %+v %v", later, err)
	}
}
