package importer

import (
	"bytes"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// Step is one instruction, with the title of its section if the source had one.
type Step struct {
	Name string
	Text string
}

// Recipe is what was read from a source, before ingredient lines are interpreted.
type Recipe struct {
	Name, Description, SourceURL, ImageURL string
	Servings, WorkMinutes, WaitMinutes     int64
	Keywords, Ingredients                  []string
	Steps                                  []Step
	// Guessed means the result came from reading the page's layout, not from recipe data
	// the site published, so it deserves a closer look.
	Guessed bool
}

// Sites stuff this field for search engines, and every tag lands in the shared list. Category
// and cuisine come first, so a low limit keeps the useful ones.
const maxKeywords = 4

// ParseHTML finds a recipe in a page: first the schema.org data most recipe sites publish
// for search engines, then microdata, then by reading the page's headings and lists.
func ParseHTML(body []byte, contentType, pageURL string) (*Recipe, error) {
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return nil, ErrNoRecipe
	}
	doc, err := nethtml.Parse(reader)
	if err != nil {
		return nil, ErrNoRecipe
	}
	r := fromJSONLD(doc)
	if r == nil {
		r = fromMicrodata(doc)
	}
	if r == nil {
		r = fromLines(pageLines(doc), false)
		if r != nil {
			r.Guessed = true
		}
	}
	if r == nil {
		return nil, ErrNoRecipe
	}
	if r.Name == "" {
		r.Name = pageTitle(doc)
	}
	if r.Description == "" {
		r.Description = metaContent(doc, "og:description", "description")
	}
	if r.ImageURL == "" {
		r.ImageURL = metaContent(doc, "og:image", "twitter:image")
	}
	r.SourceURL = pageURL
	r.ImageURL = absolute(pageURL, r.ImageURL)
	if r.Name == "" || (len(r.Ingredients) == 0 && len(r.Steps) == 0) {
		return nil, ErrNoRecipe
	}
	return r, nil
}

// ParseText reads a recipe copied from anywhere. The first line is taken as the title.
func ParseText(text string) (*Recipe, error) {
	var lines []line
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, line{text: l})
		}
	}
	if len(lines) < 2 {
		return nil, ErrNoRecipe
	}
	r := fromLines(lines, true)
	if r == nil {
		r = fromBareLines(lines)
	}
	if r == nil {
		return nil, ErrNoRecipe
	}
	r.Guessed = true
	if r.Name == "" {
		r.Name = lines[0].text
	}
	return r, nil
}

func absolute(base, ref string) string {
	if ref == "" {
		return ""
	}
	b, err1 := url.Parse(base)
	u, err2 := url.Parse(strings.TrimSpace(ref))
	if err2 != nil {
		return ""
	}
	if err1 != nil {
		return u.String()
	}
	return b.ResolveReference(u).String()
}

// ---- schema.org JSON-LD ----

func walk(n *nethtml.Node, visit func(*nethtml.Node) bool) {
	if !visit(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}

func attr(n *nethtml.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

var controlRe = regexp.MustCompile(`[\x00-\x1f]+`)

func fromJSONLD(doc *nethtml.Node) *Recipe {
	var best *Recipe
	walk(doc, func(n *nethtml.Node) bool {
		if n.Type != nethtml.ElementNode || n.Data != "script" || !strings.Contains(strings.ToLower(attr(n, "type")), "ld+json") || n.FirstChild == nil {
			return true
		}
		raw := n.FirstChild.Data
		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			// Plenty of sites put raw line breaks inside their JSON strings.
			if err := json.Unmarshal([]byte(controlRe.ReplaceAllString(raw, " ")), &data); err != nil {
				return false
			}
		}
		if m := findRecipeNode(data, 0); m != nil {
			r := readRecipe(m)
			if best == nil || (len(best.Ingredients) == 0 && len(r.Ingredients) > 0) {
				best = r
			}
		}
		return false
	})
	if best == nil || (len(best.Ingredients) == 0 && len(best.Steps) == 0) {
		return nil
	}
	return best
}

func isType(m map[string]any, want string) bool {
	for _, t := range asList(m["@type"]) {
		if s, ok := t.(string); ok {
			s = strings.ToLower(s)
			if s == want || strings.HasSuffix(s, "/"+want) {
				return true
			}
		}
	}
	return false
}

func findRecipeNode(v any, depth int) map[string]any {
	if depth > 8 {
		return nil
	}
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			if m := findRecipeNode(e, depth+1); m != nil {
				return m
			}
		}
	case map[string]any:
		if isType(x, "recipe") {
			return x
		}
		for _, k := range []string{"@graph", "mainEntity", "mainEntityOfPage", "itemListElement", "item"} {
			if m := findRecipeNode(x[k], depth+1); m != nil {
				return m
			}
		}
	}
	return nil
}

func asList(v any) []any {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		return x
	}
	return []any{v}
}

var (
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	breakRe = regexp.MustCompile(`(?i)<\s*(br|/p|/li|/div)\s*/?>`)
)

// clean turns a value that may hold HTML and entities into plain text.
func clean(s string) string {
	s = html.UnescapeString(html.UnescapeString(s))
	s = breakRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spacesRe.ReplaceAllString(l, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func text(v any) string {
	switch x := v.(type) {
	case string:
		return clean(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		if len(x) > 0 {
			return text(x[0])
		}
	case map[string]any:
		for _, k := range []string{"text", "name", "@value"} {
			if s := text(x[k]); s != "" {
				return s
			}
		}
	}
	return ""
}

func imageURL(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		for _, e := range x {
			if s := imageURL(e); s != "" {
				return s
			}
		}
	case map[string]any:
		for _, k := range []string{"url", "contentUrl", "@id"} {
			// An @id like "page#primaryimage" names a node, not a file. The page's og:image
			// is used instead, which is the same photo on sites that do this.
			if s, ok := x[k].(string); ok && s != "" && !strings.Contains(s, "#") {
				return s
			}
		}
	}
	return ""
}

func readRecipe(m map[string]any) *Recipe {
	r := &Recipe{Name: text(m["name"]), Description: text(m["description"]), ImageURL: imageURL(m["image"])}
	if r.Name == "" {
		r.Name = text(m["headline"])
	}
	// Chefkoch and others add the author to the title for search engines.
	if a := text(m["author"]); a != "" {
		for _, joiner := range []string{" von ", " by "} {
			r.Name = strings.TrimSpace(strings.TrimSuffix(r.Name, joiner+a))
		}
	}
	r.Servings = firstNumber(text(m["recipeYield"]))
	if r.Servings == 0 {
		r.Servings = firstNumber(text(m["yield"]))
	}
	setTimes(r, text(m["prepTime"]), text(m["cookTime"]), text(m["totalTime"]))
	source := m["recipeIngredient"]
	if source == nil {
		source = m["ingredients"]
	}
	for _, e := range asList(source) {
		if s := text(e); s != "" {
			r.Ingredients = append(r.Ingredients, s)
		}
	}
	r.Steps = readSteps(m["recipeInstructions"], "")
	r.Keywords = readKeywords(m["recipeCategory"], m["recipeCuisine"], m["keywords"])
	return r
}

var numberPrefixRe = regexp.MustCompile(`^(?:\d+[.)]|(?i:step|schritt)\s+\d+[.:)]?)\s*`)

func splitParagraphs(s string) []string {
	sep := "\n"
	if strings.Contains(s, "\n\n") {
		sep = "\n\n"
	}
	var out []string
	for _, p := range strings.Split(s, sep) {
		if p = strings.TrimSpace(numberPrefixRe.ReplaceAllString(strings.TrimSpace(p), "")); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readSteps(v any, section string) []Step {
	var out []Step
	add := func(s Step) {
		if s.Name == "" {
			s.Name = section
			section = "" // Only the section's first step carries its title.
		}
		out = append(out, s)
	}
	switch x := v.(type) {
	case string:
		for _, p := range splitParagraphs(clean(x)) {
			add(Step{Text: p})
		}
	case []any:
		for _, e := range x {
			for _, s := range readSteps(e, section) {
				section = ""
				out = append(out, s)
			}
		}
	case map[string]any:
		if isType(x, "howtosection") || (x["itemListElement"] != nil && text(x["text"]) == "") {
			name := text(x["name"])
			inner := readSteps(x["itemListElement"], name)
			if len(inner) == 0 {
				inner = readSteps(x["steps"], name)
			}
			out = append(out, inner...)
			break
		}
		body := text(x["text"])
		if body == "" {
			body = text(x["description"])
		}
		name := text(x["name"])
		if body == "" {
			body, name = name, ""
		}
		// Many sites repeat the start of the text as the step's name.
		if name != "" && (strings.HasPrefix(body, strings.TrimRight(name, ".… ")) || len(name) > 60) {
			name = ""
		}
		if body != "" {
			add(Step{Name: name, Text: body})
		}
	}
	return out
}

func readKeywords(values ...any) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range values {
		for _, e := range asList(v) {
			for _, k := range strings.Split(text(e), ",") {
				k = strings.TrimSpace(k)
				key := strings.ToLower(k)
				if k == "" || len(k) > 30 || seen[key] || len(out) >= maxKeywords {
					continue
				}
				seen[key] = true
				out = append(out, k)
			}
		}
	}
	return out
}

var (
	digitsRe = regexp.MustCompile(`\d+`)
	isoRe    = regexp.MustCompile(`(?i)^P(?:(\d+)D)?(?:T(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:\d+(?:\.\d+)?S)?)?$`)
	looseRe  = regexp.MustCompile(`(?i)(?:(\d+)\s*(?:h|std|stunde|stunden|hour|hours|hr|hrs)\b)?\s*(?:(\d+)\s*(?:m|min|mins|minute|minuten|minutes)\b)?`)
)

func firstNumber(s string) int64 {
	n, err := strconv.ParseInt(digitsRe.FindString(s), 10, 64)
	if err != nil || n < 1 || n > 1000 {
		return 0
	}
	return n
}

func minutes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f := func(v string) float64 { x, _ := strconv.ParseFloat(v, 64); return x }
	var total float64
	if m := isoRe.FindStringSubmatch(s); m != nil {
		total = f(m[1])*1440 + f(m[2])*60 + f(m[3])
	} else if m := looseRe.FindStringSubmatch(s); m != nil {
		total = f(m[1])*60 + f(m[2])
	}
	if total < 0 || total > 60*24*60 {
		return 0
	}
	return int64(total + 0.5)
}

func setTimes(r *Recipe, prep, cook, total string) {
	p, c, t := minutes(prep), minutes(cook), minutes(total)
	r.WorkMinutes, r.WaitMinutes = p, c
	switch {
	case p == 0 && c == 0:
		r.WorkMinutes = t
	case c == 0 && t > p:
		r.WaitMinutes = t - p
	}
}

// ---- microdata ----

func hasProp(n *nethtml.Node, name string) bool {
	for _, p := range strings.Fields(attr(n, "itemprop")) {
		if strings.EqualFold(p, name) {
			return true
		}
	}
	return false
}

func fromMicrodata(doc *nethtml.Node) *Recipe {
	var root *nethtml.Node
	walk(doc, func(n *nethtml.Node) bool {
		if root == nil && n.Type == nethtml.ElementNode && strings.Contains(strings.ToLower(attr(n, "itemtype")), "schema.org/recipe") {
			root = n
		}
		return root == nil
	})
	if root == nil {
		return nil
	}
	r := &Recipe{}
	var prep, cook, total string
	var keywords []any
	value := func(n *nethtml.Node) string {
		for _, a := range []string{"content", "datetime"} {
			if v := attr(n, a); v != "" {
				return clean(v)
			}
		}
		return nodeText(n)
	}
	walk(root, func(n *nethtml.Node) bool {
		if n == root || n.Type != nethtml.ElementNode {
			return true
		}
		switch {
		case hasProp(n, "name") && r.Name == "":
			r.Name = value(n)
		case hasProp(n, "description") && r.Description == "":
			r.Description = value(n)
		case hasProp(n, "image") && r.ImageURL == "":
			r.ImageURL = attr(n, "src")
			if r.ImageURL == "" {
				r.ImageURL = attr(n, "content")
			}
		case hasProp(n, "recipeYield"):
			r.Servings = firstNumber(value(n))
		case hasProp(n, "prepTime"):
			prep = value(n)
		case hasProp(n, "cookTime"):
			cook = value(n)
		case hasProp(n, "totalTime"):
			total = value(n)
		case hasProp(n, "recipeIngredient") || hasProp(n, "ingredients"):
			if s := value(n); s != "" {
				r.Ingredients = append(r.Ingredients, strings.ReplaceAll(s, "\n", " "))
			}
		case hasProp(n, "recipeInstructions"):
			for _, p := range splitParagraphs(value(n)) {
				r.Steps = append(r.Steps, Step{Text: p})
			}
			return false
		case hasProp(n, "keywords") || hasProp(n, "recipeCategory") || hasProp(n, "recipeCuisine"):
			keywords = append(keywords, value(n))
		}
		return true
	})
	setTimes(r, prep, cook, total)
	r.Keywords = readKeywords(keywords...)
	if len(r.Ingredients) == 0 {
		return nil
	}
	return r
}

// ---- reading a page's layout ----

var (
	skipTags  = map[string]bool{"script": true, "style": true, "noscript": true, "nav": true, "footer": true, "aside": true, "form": true, "svg": true, "template": true, "iframe": true, "button": true}
	blockTags = map[string]bool{"p": true, "div": true, "li": true, "ul": true, "ol": true, "section": true, "article": true, "br": true, "tr": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "table": true, "main": true}
)

func nodeText(n *nethtml.Node) string {
	var b strings.Builder
	var rec func(*nethtml.Node)
	rec = func(c *nethtml.Node) {
		if c.Type == nethtml.TextNode {
			b.WriteString(c.Data)
			return
		}
		if c.Type == nethtml.ElementNode && skipTags[c.Data] {
			return
		}
		isBlock := c.Type == nethtml.ElementNode && blockTags[c.Data]
		if isBlock {
			b.WriteString("\n")
		}
		for k := c.FirstChild; k != nil; k = k.NextSibling {
			rec(k)
		}
		if isBlock {
			b.WriteString("\n")
		}
	}
	rec(n)
	lines := strings.Split(b.String(), "\n")
	var out []string
	for _, l := range lines {
		if l = strings.TrimSpace(spacesRe.ReplaceAllString(l, " ")); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

type line struct {
	text    string
	heading bool
	item    bool
}

func hasBlockChild(n *nethtml.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && (blockTags[c.Data] || hasBlockChild(c)) {
			return true
		}
	}
	return false
}

func pageLines(doc *nethtml.Node) []line {
	var out []line
	walk(doc, func(n *nethtml.Node) bool {
		if n.Type != nethtml.ElementNode {
			return true
		}
		if skipTags[n.Data] {
			return false
		}
		switch n.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			if t := nodeText(n); t != "" {
				out = append(out, line{text: strings.ReplaceAll(t, "\n", " "), heading: true})
			}
			return false
		case "li":
			if t := nodeText(n); t != "" {
				out = append(out, line{text: strings.ReplaceAll(t, "\n", " "), item: true})
			}
			return false
		case "p", "div", "td":
			if !hasBlockChild(n) {
				if t := nodeText(n); t != "" {
					out = append(out, line{text: strings.ReplaceAll(t, "\n", " ")})
				}
				return false
			}
		}
		return true
	})
	return out
}

var (
	ingredientsRe = regexp.MustCompile(`(?i)^(?:die\s+|the\s+|les\s+|gli\s+|los\s+)?(ingredients?|zutaten|ingrédients|ingredienti|ingredientes|ingrediënten|ingredienser|składniki)\b`)
	stepsRe       = regexp.MustCompile(`(?i)^(?:die\s+|the\s+|la\s+|le\s+)?(instructions?|directions?|method|preparation|steps|how to make|zubereitung|anleitung|so geht'?s|schritte|arbeitsschritte|préparation|preparazione|procedimento|preparación|elaboración|instrucciones|modo de preparo|bereiding|bereidingswijze|tillagning|przygotowanie)\b`)
	yieldRe       = regexp.MustCompile(`(?i)\b(?:für|for|serves|makes|yield:?|portionen:?|servings:?)\s+(\d+)|(\d+)\s*(?:portionen|servings|personen|people)\b`)
)

const maxHeadingLen = 48

// fromLines reads "title, ingredients, steps" from a flat list of blocks. It returns nil
// unless it finds both sections, so a normal article isn't mistaken for a recipe.
func fromLines(lines []line, plainText bool) *Recipe {
	r := &Recipe{}
	const (
		none = iota
		ingredients
		steps
	)
	state := none
	for _, l := range lines {
		isIng := len(l.text) <= maxHeadingLen && ingredientsRe.MatchString(l.text)
		isStep := len(l.text) <= maxHeadingLen && stepsRe.MatchString(l.text)
		if (l.heading || !l.item) && isIng && len(r.Ingredients) == 0 {
			state = ingredients
			if r.Servings == 0 {
				if m := yieldRe.FindStringSubmatch(l.text); m != nil {
					r.Servings = firstNumber(m[1] + m[2])
				}
			}
			continue
		}
		if (l.heading || !l.item) && isStep && len(r.Steps) == 0 {
			state = steps
			continue
		}
		switch state {
		case none:
			if l.heading && r.Name == "" && !isStep && !isIng {
				r.Name = l.text
			}
			if r.Servings == 0 && len(l.text) < 60 {
				if m := yieldRe.FindStringSubmatch(l.text); m != nil {
					r.Servings = firstNumber(m[1] + m[2])
				}
			}
		case ingredients:
			if l.heading || strings.HasSuffix(l.text, ":") {
				if !strings.HasSuffix(l.text, ":") && len(r.Ingredients) > 0 && !plainText && !subheadingRe.MatchString(l.text) {
					state = none
				}
				continue
			}
			if !plainText && !l.item && len(l.text) > 100 {
				continue
			}
			r.Ingredients = append(r.Ingredients, l.text)
		case steps:
			if l.heading {
				if stepHeadingRe.MatchString(l.text) {
					continue
				}
				if len(r.Steps) > 0 && !plainText {
					state = none
				}
				continue
			}
			r.Steps = append(r.Steps, Step{Text: strings.TrimSpace(numberPrefixRe.ReplaceAllString(l.text, ""))})
		}
	}
	if len(r.Ingredients) < 2 || len(r.Steps) < 1 {
		return nil
	}
	return r
}

var (
	subheadingRe  = regexp.MustCompile(`(?i)^(für|for|pour|per|para)\b`)
	stepHeadingRe = regexp.MustCompile(`(?i)^(schritt|step|étape|passo|paso)\s*\d+`)
	startsNumRe   = regexp.MustCompile(`^[\d½¼¾⅓⅔⅛]`)
)

// fromBareLines handles pasted text with no section titles: a title, then lines that start
// with an amount, then everything else as steps.
func fromBareLines(lines []line) *Recipe {
	r := &Recipe{Name: lines[0].text}
	for _, l := range lines[1:] {
		if len(r.Steps) == 0 && len(l.text) <= 100 && startsNumRe.MatchString(l.text) && !numberPrefixRe.MatchString(l.text) {
			r.Ingredients = append(r.Ingredients, l.text)
			continue
		}
		r.Steps = append(r.Steps, Step{Text: strings.TrimSpace(numberPrefixRe.ReplaceAllString(l.text, ""))})
	}
	if len(r.Ingredients) == 0 && len(r.Steps) == 0 {
		return nil
	}
	return r
}

func pageTitle(doc *nethtml.Node) string {
	if t := metaContent(doc, "og:title"); t != "" {
		return t
	}
	var out string
	walk(doc, func(n *nethtml.Node) bool {
		if out == "" && n.Type == nethtml.ElementNode && (n.Data == "h1" || n.Data == "title") {
			out = strings.ReplaceAll(nodeText(n), "\n", " ")
		}
		return out == ""
	})
	return out
}

func metaContent(doc *nethtml.Node, names ...string) string {
	for _, name := range names {
		var out string
		walk(doc, func(n *nethtml.Node) bool {
			if out == "" && n.Type == nethtml.ElementNode && n.Data == "meta" && (strings.EqualFold(attr(n, "property"), name) || strings.EqualFold(attr(n, "name"), name)) {
				out = clean(attr(n, "content"))
			}
			return out == ""
		})
		if out != "" {
			return out
		}
	}
	return ""
}
