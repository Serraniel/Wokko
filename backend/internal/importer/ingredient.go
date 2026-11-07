package importer

import (
	"regexp"
	"strconv"
	"strings"
)

// Ingredient is one parsed line of a recipe.
type Ingredient struct {
	Amount *float64
	Unit   string
	Food   string
	Note   string
}

var fractions = strings.NewReplacer("½", " 1/2 ", "¼", " 1/4 ", "¾", " 3/4 ", "⅓", " 1/3 ", "⅔", " 2/3 ", "⅛", " 1/8 ", "⅜", " 3/8 ", "⅝", " 5/8 ", "⅞", " 7/8 ", "⅕", " 1/5 ", "⅙", " 1/6 ", " ", " ", " ", " ", " ", " ", "⁄", "/")

const num = `\d+\s+\d+/\d+|\d+/\d+|\d+[.,]\d+|\d+`

var (
	amountRe   = regexp.MustCompile(`(?i)^(?:ca\.?\s*|circa\s+|etwa\s+|approx\.?\s*|about\s+|~\s*)?(` + num + `)(?:\s*(?:-|–|—|bis|to)\s*(` + num + `))?\s*(.*)$`)
	trailingRe = regexp.MustCompile(`^(.*\p{L}.*?)\s+(` + num + `)\s*(\p{L}*\.?)$`)
	genderRe   = regexp.MustCompile(`(?i)\((?:n|s|e|en|es|er|r)\)|/(?:n|en|e|s)\b`)
	parensRe   = regexp.MustCompile(`\s*\(([^)]*)\)`)
	bulletRe   = regexp.MustCompile(`^[\s\-–—•*·▢☐□]+`)
	spacesRe   = regexp.MustCompile(`\s+`)
	leadingRe  = regexp.MustCompile(`(?i)^(etwas|evtl\.?|eventuell|optional|some|a little|a bit of)\s+`)
	tasteRe    = regexp.MustCompile(`(?i)[\s,]+(nach belieben|nach geschmack|n\.\s?b\.|to taste|optional|as needed|for serving|zum servieren)\.?$`)
)

// unitAliases are spellings recipe sites use that the catalog doesn't list. A unit mapped to
// itself isn't in the starter catalog and is created when the recipe is saved.
var unitAliases = map[string]string{
	"pck": "Packung", "pkg": "Packung", "pack": "Packung", "pkt": "Packung", "päckchen": "Packung", "packet": "Packung", "tüte": "Beutel",
	"gr": "g", "kilo": "kg", "lbs": "lb", "tbs": "tbsp", "tbl": "tbsp", "stück": "Stück", "st": "Stück", "stk": "Stück",
	"msp": "Msp", "messerspitze": "Msp", "becher": "Becher", "dl": "dl", "knolle": "Knolle",
}

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if parts := strings.Fields(s); len(parts) == 2 {
		w, ok1 := parseNumber(parts[0])
		f, ok2 := parseNumber(parts[1])
		return w + f, ok1 && ok2
	}
	if a, b, ok := strings.Cut(s, "/"); ok {
		x, err1 := strconv.ParseFloat(a, 64)
		y, err2 := strconv.ParseFloat(b, 64)
		if err1 != nil || err2 != nil || y == 0 {
			return 0, false
		}
		return x / y, true
	}
	// "1.000" and "1,000" are thousands in a recipe. "1,5" is a decimal comma.
	if i := strings.IndexAny(s, ".,"); i > 0 && i <= 3 && len(s)-i-1 == 3 {
		s = s[:i] + s[i+1:]
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return v, err == nil
}

func unitKey(token string) string {
	return strings.ToLower(strings.TrimRight(genderRe.ReplaceAllString(token, ""), ".,:"))
}

// ParseIngredient reads a line such as "1 1/2 EL Öl, gehackt". units maps lower-case
// spellings to the name to save. When unsure, more of the line stays in the food name, which
// is easy to fix in the editor. Nothing is ever dropped.
func ParseIngredient(line string, units map[string]string) Ingredient {
	s := strings.TrimSpace(spacesRe.ReplaceAllString(fractions.Replace(line), " "))
	s = bulletRe.ReplaceAllString(s, "")
	var in Ingredient
	var notes []string
	if m := amountRe.FindStringSubmatch(s); m != nil {
		if v, ok := parseNumber(m[1]); ok {
			in.Amount = &v
			if m[2] != "" {
				notes = append(notes, strings.TrimSpace(m[1])+"–"+strings.TrimSpace(m[2]))
			}
			s = m[3]
			if _, bare := units[unitKey(s)]; bare {
				s = "" // Only an amount and a unit, with no food named.
			} else if token, rest, ok := strings.Cut(s, " "); ok {
				key := unitKey(token)
				if name, known := units[key]; known {
					in.Unit, s = name, rest
				} else if name, known := unitAliases[key]; known {
					if canon, ok := units[strings.ToLower(name)]; ok {
						name = canon
					}
					in.Unit, s = name, rest
				}
			}
		}
	}
	// Some sites, many Italian ones among them, write the amount last: "Spaghetti 320 g".
	if in.Amount == nil {
		if m := trailingRe.FindStringSubmatch(s); m != nil {
			if v, ok := parseNumber(m[2]); ok {
				name, known := units[unitKey(m[3])]
				if !known {
					name, known = unitAliases[unitKey(m[3])]
				}
				if m[3] == "" || known {
					in.Amount, in.Unit, s = &v, name, m[1]
				}
			}
		}
	}
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "of "))
	if m := leadingRe.FindStringSubmatch(s); m != nil {
		notes = append(notes, strings.ToLower(m[1]))
		s = s[len(m[0]):]
	}
	s = genderRe.ReplaceAllString(s, "")
	if m := tasteRe.FindStringSubmatch(s); m != nil {
		notes = append(notes, m[1])
		s = s[:len(s)-len(m[0])]
	}
	for _, m := range parensRe.FindAllStringSubmatch(s, -1) {
		if t := strings.TrimSpace(m[1]); t != "" {
			notes = append(notes, t)
		}
	}
	s = parensRe.ReplaceAllString(s, "")
	if food, note, ok := strings.Cut(s, ","); ok && strings.TrimSpace(food) != "" {
		s = food
		notes = append(notes, strings.TrimSpace(note))
	}
	in.Food = strings.TrimSpace(s)
	if r := []rune(in.Food); len(r) > 120 {
		cut := 120
		for cut > 60 && r[cut] != ' ' {
			cut--
		}
		notes = append(notes, strings.TrimSpace(string(r[cut:])))
		in.Food = strings.TrimSpace(string(r[:cut]))
	}
	if in.Food == "" { // "2 EL" alone, or a line of only parentheses: keep what was written.
		in.Food = strings.TrimSpace(line)
		in.Amount, in.Unit, notes = nil, "", nil
	}
	in.Note = strings.Join(notes, ", ")
	return in
}
