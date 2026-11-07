// Package text holds locale-independent text helpers.
package text

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var folder = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Normalize folds text for searching: lowercase, without diacritics, so "kase" finds "Käse".
// It works for any language; scripts without diacritics are only lowercased.
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ß", "ss")
	out, _, err := transform.String(folder, s)
	if err != nil {
		return s
	}
	return strings.Join(strings.Fields(out), " ")
}

// EscapeLike escapes nothing today because search text is matched by LIKE with user input
// bound as a parameter; % and _ are simply removed to avoid surprising wildcard matches.
func EscapeLike(s string) string {
	return strings.NewReplacer("%", "", "_", " ").Replace(s)
}

var localeRe = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// ValidLocale reports whether s looks like a BCP 47 tag. Any language can be stored.
func ValidLocale(s string) bool { return localeRe.MatchString(s) }

// PickLocale chooses the best language from an Accept-Language header value.
func PickLocale(header, fallback string) string {
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if tag != "" && tag != "*" && ValidLocale(tag) {
			return strings.ToLower(strings.SplitN(tag, "-", 2)[0])
		}
	}
	return fallback
}

// Translation is one localized variant of a name.
type Translation[T any] struct {
	Locale string
	Value  T
}

// Pick returns the value for the wanted locale, then English, then the first one available,
// so content in a language the viewer doesn't use is shown rather than hidden.
func Pick[T any](items []Translation[T], locale string) (T, bool) {
	var english *T
	for i := range items {
		if items[i].Locale == locale {
			return items[i].Value, true
		}
		if items[i].Locale == "en" {
			english = &items[i].Value
		}
	}
	if english != nil {
		return *english, true
	}
	if len(items) > 0 {
		return items[0].Value, true
	}
	var zero T
	return zero, false
}
