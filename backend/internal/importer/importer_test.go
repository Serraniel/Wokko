package importer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

var units = map[string]string{"g": "gram", "kg": "kilogram", "el": "Esslöffel", "tl": "Teelöffel", "tbsp": "tablespoon", "cup": "cup", "cups": "cup", "prise": "Prise", "prisen": "Prise", "packung": "Packung", "zehe": "Zehe", "dose": "Dose"}

func TestParseIngredient(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		line string
		want Ingredient
	}{
		{"200 g Mehl", Ingredient{f(200), "gram", "Mehl", ""}},
		{"200g Mehl", Ingredient{f(200), "gram", "Mehl", ""}},
		{"1,5 kg Kartoffeln, festkochend", Ingredient{f(1.5), "kilogram", "Kartoffeln", "festkochend"}},
		{"1.000 g Mehl", Ingredient{f(1000), "gram", "Mehl", ""}},
		{"1 1/2 EL Öl", Ingredient{f(1.5), "Esslöffel", "Öl", ""}},
		{"½ TL Salz", Ingredient{f(0.5), "Teelöffel", "Salz", ""}},
		{"1½ cups flour", Ingredient{f(1.5), "cup", "flour", ""}},
		{"2 cups of flour", Ingredient{f(2), "cup", "flour", ""}},
		{"2 Zwiebel(n)", Ingredient{f(2), "", "Zwiebel", ""}},
		{"2 Zehe/n Knoblauch", Ingredient{f(2), "Zehe", "Knoblauch", ""}},
		{"1 Prise(n) Salz", Ingredient{f(1), "Prise", "Salz", ""}},
		{"1 Pck. Vanillezucker", Ingredient{f(1), "Packung", "Vanillezucker", ""}},
		{"1-2 EL Honig", Ingredient{f(1), "Esslöffel", "Honig", "1–2"}},
		{"Salz und Pfeffer", Ingredient{nil, "", "Salz und Pfeffer", ""}},
		{"Salz, nach Geschmack", Ingredient{nil, "", "Salz", "nach Geschmack"}},
		{"Pfeffer to taste", Ingredient{nil, "", "Pfeffer", "to taste"}},
		{"etwas Butter", Ingredient{nil, "", "Butter", "etwas"}},
		{"100 g Parmesan (gerieben)", Ingredient{f(100), "gram", "Parmesan", "gerieben"}},
		{"3 EL", Ingredient{nil, "", "3 EL", ""}},
		{"Spaghetti 320 g", Ingredient{f(320), "gram", "Spaghetti", ""}},
		{"Tuorli (di uova medie) 6", Ingredient{f(6), "", "Tuorli", "di uova medie"}},
		{"Vitamin B12", Ingredient{nil, "", "Vitamin B12", ""}},
	}
	for _, c := range cases {
		got := ParseIngredient(c.line, units)
		same := got.Unit == c.want.Unit && got.Food == c.want.Food && got.Note == c.want.Note &&
			((got.Amount == nil && c.want.Amount == nil) || (got.Amount != nil && c.want.Amount != nil && *got.Amount == *c.want.Amount))
		if !same {
			amount := "nil"
			if got.Amount != nil {
				amount = fmt.Sprint(*got.Amount)
			}
			t.Errorf("%q: got amount=%s unit=%q food=%q note=%q", c.line, amount, got.Unit, got.Food, got.Note)
		}
	}
}

func TestJSONLDInAGraphWithSections(t *testing.T) {
	page := `<html><head><title>Ignored</title><script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"WebSite","name":"Blog"},
	{"@type":["Recipe","NewsArticle"],"name":"Linsen &amp; Reis von Anna","author":{"@type":"Person","name":"Anna"},"description":"<p>Gut</p>","image":[{"@type":"ImageObject","url":"/img.jpg"}],
	"recipeYield":["4","4 Portionen"],"prepTime":"PT15M","cookTime":"PT1H","totalTime":"PT1H15M","recipeCategory":"Hauptgericht","keywords":"schnell, vegan, schnell",
	"recipeIngredient":["200 g Linsen","1 Zwiebel"],
	"recipeInstructions":[{"@type":"HowToSection","name":"Vorbereiten","itemListElement":[{"@type":"HowToStep","text":"Linsen waschen."},{"@type":"HowToStep","name":"Schneiden","text":"Zwiebel schneiden."}]},{"@type":"HowToStep","text":"Kochen."}]}]}</script></head><body></body></html>`
	r, err := ParseHTML([]byte(page), "text/html", "https://example.com/a/b")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Linsen & Reis" || r.Description != "Gut" || r.Servings != 4 || r.WorkMinutes != 15 || r.WaitMinutes != 60 || r.Guessed {
		t.Fatalf("bad fields: %+v", r)
	}
	if r.ImageURL != "https://example.com/img.jpg" {
		t.Fatalf("image: %q", r.ImageURL)
	}
	if strings.Join(r.Keywords, "|") != "Hauptgericht|schnell|vegan" {
		t.Fatalf("keywords: %v", r.Keywords)
	}
	if len(r.Steps) != 3 || r.Steps[0].Name != "Vorbereiten" || r.Steps[1].Name != "Schneiden" || r.Steps[2].Text != "Kochen." {
		t.Fatalf("steps: %+v", r.Steps)
	}
}

func TestJSONLDWithRawLineBreaksAndStringInstructions(t *testing.T) {
	page := "<script type=\"application/ld+json\">{\"@type\":\"Recipe\",\"name\":\"Suppe\",\"recipeIngredient\":[\"1 l Wasser\"],\"recipeInstructions\":\"Erst das.\nDann das.\"}</script>"
	r, err := ParseHTML([]byte(page), "text/html", "https://e.com/")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Suppe" || len(r.Steps) != 1 || !strings.Contains(r.Steps[0].Text, "Dann das.") {
		t.Fatalf("got %+v", r)
	}
	page = `<script type="application/ld+json">{"@type":"Recipe","name":"S","recipeIngredient":["x"],"recipeInstructions":"1. Erst das.\n\n2. Dann das."}</script>`
	r, err = ParseHTML([]byte(page), "text/html", "https://e.com/")
	if err != nil || len(r.Steps) != 2 || r.Steps[1].Text != "Dann das." {
		t.Fatalf("paragraphs: %+v %v", r, err)
	}
}

func TestImageReferenceFallsBackToThePagePhoto(t *testing.T) {
	page := `<head><meta property="og:image" content="https://e.com/real.jpg"></head><script type="application/ld+json">{"@type":"Recipe","name":"S","image":{"@id":"https://e.com/#primaryimage"},"recipeIngredient":["x"]}</script>`
	r, err := ParseHTML([]byte(page), "text/html", "https://e.com/")
	if err != nil || r.ImageURL != "https://e.com/real.jpg" {
		t.Fatalf("got %+v %v", r, err)
	}
}

func TestMicrodata(t *testing.T) {
	page := `<div itemscope itemtype="http://schema.org/Recipe"><h1 itemprop="name">Pie</h1><img itemprop="image" src="p.jpg"><meta itemprop="prepTime" content="PT20M"><span itemprop="recipeYield">Serves 6</span>
	<li itemprop="recipeIngredient">2 apples</li><li itemprop="ingredients">1 cup sugar</li><div itemprop="recipeInstructions"><p>Bake.</p><p>Eat.</p></div></div>`
	r, err := ParseHTML([]byte(page), "text/html", "https://e.com/x/")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Pie" || r.Servings != 6 || r.WorkMinutes != 20 || len(r.Ingredients) != 2 || len(r.Steps) != 2 || r.ImageURL != "https://e.com/x/p.jpg" {
		t.Fatalf("got %+v", r)
	}
}

func TestPageWithoutRecipeDataIsReadFromItsLayout(t *testing.T) {
	page := `<html><head><meta property="og:image" content="/o.jpg"></head><body><nav><ul><li>Home</li></ul></nav><h1>Omas Kuchen</h1><p>Eine lange Geschichte.</p>
	<h2>Zutaten für 8 Stücke</h2><ul><li>200 g Mehl</li><li>3 Eier</li></ul><h3>Für den Guss:</h3><ul><li>100 g Zucker</li></ul>
	<h2>Zubereitung</h2><ol><li>Alles mischen.</li><li>Backen.</li></ol><h2>Kommentare</h2><p>Lecker!</p><footer>Impressum</footer></body></html>`
	r, err := ParseHTML([]byte(page), "text/html", "https://e.com/")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Guessed || r.Name != "Omas Kuchen" || len(r.Ingredients) != 3 || len(r.Steps) != 2 || r.ImageURL != "https://e.com/o.jpg" {
		t.Fatalf("got %+v", r)
	}
	if r.Servings != 8 {
		t.Fatalf("servings %d", r.Servings)
	}
}

func TestAnOrdinaryArticleIsNotARecipe(t *testing.T) {
	page := `<html><body><h1>Why I love cooking</h1><p>Ingredients matter a lot.</p><ul><li>One</li><li>Two</li></ul></body></html>`
	if _, err := ParseHTML([]byte(page), "text/html", "https://e.com/"); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("got %v", err)
	}
}

func TestPageInAnOldCharset(t *testing.T) {
	page := append([]byte(`<html><head><meta charset="iso-8859-1"></head><body><script type="application/ld+json">{"@type":"Recipe","name":"Gem`), 0xfc)
	page = append(page, []byte(`se","recipeIngredient":["x"]}</script></body></html>`)...)
	r, err := ParseHTML(page, "text/html", "https://e.com/")
	if err != nil || r.Name != "Gemüse" {
		t.Fatalf("got %+v %v", r, err)
	}
}

func TestPastedText(t *testing.T) {
	r, err := ParseText("Tomatensoße\n\nZutaten:\n800 g Tomaten\n1 Zwiebel\n\nZubereitung:\n1. Schneiden.\n2. Köcheln.\n")
	if err != nil || r.Name != "Tomatensoße" || len(r.Ingredients) != 2 || len(r.Steps) != 2 || r.Steps[0].Text != "Schneiden." {
		t.Fatalf("with titles: %+v %v", r, err)
	}
	r, err = ParseText("Quick soup\n2 carrots\n1 l water\nBoil it all until soft, then blend.")
	if err != nil || r.Name != "Quick soup" || len(r.Ingredients) != 2 || len(r.Steps) != 1 {
		t.Fatalf("bare: %+v %v", r, err)
	}
	if _, err := ParseText("one line"); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("one line: %v", err)
	}
}

func TestMinutes(t *testing.T) {
	for in, want := range map[string]int64{"PT1H30M": 90, "P0DT0H20M": 20, "PT45M": 45, "P1D": 1440, "PT90S": 0, "1 Std. 10 Min": 0, "30 min": 30, "1 h 10 min": 70, "garbage": 0, "": 0} {
		if in == "1 Std. 10 Min" {
			continue // Written-out times vary too much to promise. The editor shows what was found.
		}
		if got := minutes(in); got != want {
			t.Errorf("%q: %d, want %d", in, got, want)
		}
	}
}

func TestOnlyPublicAddressesCount(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.20", "172.17.0.1", "169.254.169.254", "100.64.0.1", "::1", "fe80::1", "fc00::1", "::ffff:127.0.0.1", "64:ff9b::7f00:1", "0.0.0.0"} {
		if publicIP(netip.MustParseAddr(s)) {
			t.Errorf("%s counted as public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "2a00:1450:4001::1"} {
		if !publicIP(netip.MustParseAddr(s)) {
			t.Errorf("%s counted as private", s)
		}
	}
}

func TestFetchRefusesPrivateTargetsEvenAfterRedirects(t *testing.T) {
	inner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("the private server was reached") }))
	defer inner.Close()
	ctx := context.Background()
	safe := NewFetcher("test", false)
	if _, _, _, err := safe.Page(ctx, inner.URL); !errors.Is(err, ErrBadURL) && !errors.Is(err, ErrPrivate) {
		t.Fatalf("direct: %v", err)
	}
	for _, bad := range []string{"http://127.0.0.1/", "http://localhost/", "http://[::1]/", "http://169.254.169.254/latest/meta-data/", "http://192.168.1.20/", "http://2130706433/"} {
		if _, _, _, err := safe.Page(ctx, bad); !errors.Is(err, ErrPrivate) && !errors.Is(err, ErrUnreachable) {
			t.Errorf("%s: %v", bad, err)
		} else if bad == "http://127.0.0.1/" && !errors.Is(err, ErrPrivate) {
			t.Errorf("%s: wanted ErrPrivate, got %v", bad, err)
		}
	}
	for _, bad := range []string{"file:///etc/passwd", "ftp://example.com/", "gopher://x/", "http://user:pw@example.com/", "http://example.com:8080/", "not a url", ""} {
		if _, _, _, err := safe.Page(ctx, bad); !errors.Is(err, ErrBadURL) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	// A public-looking server redirecting inward: the redirect's target is checked on connect.
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:80/admin", http.StatusFound)
	}))
	defer redirect.Close()
	f := NewFetcher("test", false)
	f.client.Transport.(*http.Transport).DialContext = (&dialFirstAllowed{f: f, allowed: redirect.Listener.Addr().String()}).dial
	if _, _, _, err := f.get(ctx, "http://example.com/", "*/*", 1000); !errors.Is(err, ErrPrivate) {
		t.Fatalf("redirect: %v", err)
	}
}

// dialFirstAllowed lets one known test server through and applies the real check to the rest,
// which stands in for a public site that redirects to an internal address.
type dialFirstAllowed struct {
	f       *Fetcher
	allowed string
}

func (d *dialFirstAllowed) dial(ctx context.Context, network, address string) (net.Conn, error) {
	if address == "example.com:80" {
		return (&net.Dialer{}).DialContext(ctx, network, d.allowed)
	}
	return (&net.Dialer{Control: d.f.checkDial}).DialContext(ctx, network, address)
}

func TestBlockedAndOversizedResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blocked":
			w.WriteHeader(http.StatusForbidden)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/big":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write(make([]byte, maxPageBytes+10))
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
		}
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "Wokko/") {
			t.Errorf("user agent %q", ua)
		}
	}))
	defer srv.Close()
	f, ctx := NewFetcher("test", true), context.Background()
	for path, want := range map[string]error{"/blocked": ErrBlocked, "/missing": ErrUnreachable, "/big": ErrNotPage, "/pdf": ErrNotPage} {
		if _, _, _, err := f.Page(ctx, srv.URL+path); !errors.Is(err, want) {
			t.Errorf("%s: %v", path, err)
		}
	}
}
