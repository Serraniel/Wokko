package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

// What an assistant may do with names that aren't in the catalog yet, chosen by each person.
const (
	NewEntriesNever = "never" // Saving fails until every ingredient uses a known food and unit.
	NewEntriesAsk   = "ask"   // The assistant has to confirm each new name with the person first.
	NewEntriesAllow = "allow" // New names become new foods and units, as in the editor.
)

// lookupFood finds a food by a name, plural or alias in any language.
func lookupFood(ctx context.Context, q *sqlcgen.Queries, name string) (int64, bool, error) {
	if id, err := q.FindFoodIDByName(ctx, name); err == nil {
		return id, true, nil
	} else if !isNoRows(err) {
		return 0, false, err
	}
	// Another spelling of a known food, such as one kept after a merge, is that food.
	if id, err := q.FindFoodIDByAlias(ctx, text.Normalize(name)); err == nil {
		return id, true, nil
	} else if !isNoRows(err) {
		return 0, false, err
	}
	return 0, false, nil
}

// lookupUnit finds a unit by a name, plural, abbreviation or alias in any language.
func lookupUnit(ctx context.Context, q *sqlcgen.Queries, name string) (int64, bool, error) {
	if id, err := q.FindUnitIDByName(ctx, name); err == nil {
		return id, true, nil
	} else if !isNoRows(err) {
		return 0, false, err
	}
	if id, err := q.FindUnitIDByAlias(ctx, text.Normalize(name)); err == nil {
		return id, true, nil
	} else if !isNoRows(err) {
		return 0, false, err
	}
	return 0, false, nil
}

type newName struct{ kind, name string }

// newCatalogNames lists the ingredient names a save would add to the catalog.
func (s *Service) newCatalogNames(ctx context.Context, in RecipeInput) ([]newName, error) {
	q := s.Store.Queries
	var out []newName
	seen := map[string]bool{}
	add := func(kind, name string, find func(context.Context, *sqlcgen.Queries, string) (int64, bool, error)) error {
		key := kind + ":" + text.Normalize(name)
		if name == "" || seen[key] {
			return nil
		}
		seen[key] = true
		_, ok, err := find(ctx, q, name)
		if err == nil && !ok {
			out = append(out, newName{kind, name})
		}
		return err
	}
	for _, st := range in.Steps {
		for _, ing := range st.Ingredients {
			if ing.FoodID <= 0 {
				if err := add("food", strings.TrimSpace(ing.FoodName), lookupFood); err != nil {
					return nil, err
				}
			}
			if ing.UnitID <= 0 {
				if err := add("unit", strings.TrimSpace(ing.UnitName), lookupUnit); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

// CheckAssistantNewEntries applies the person's choice about new foods and units to a recipe an
// assistant wants to save. The refusal names each new entry with close matches, so the
// assistant can use those instead.
func (s *Service) CheckAssistantNewEntries(ctx context.Context, v Viewer, in RecipeInput, confirmed []string) error {
	mode := v.User.AssistantNewEntries
	if mode == NewEntriesAllow {
		return nil
	}
	names, err := s.newCatalogNames(ctx, in)
	if err != nil {
		return err
	}
	ok := map[string]bool{}
	for _, c := range confirmed {
		ok[text.Normalize(c)] = true
	}
	var missing []string
	for _, n := range names {
		if mode == NewEntriesAsk && ok[text.Normalize(n.name)] {
			continue
		}
		line, err := s.describeNewName(ctx, v, n)
		if err != nil {
			return err
		}
		missing = append(missing, line)
	}
	if len(missing) == 0 {
		return nil
	}
	list := strings.Join(missing, "; ")
	if mode == NewEntriesNever {
		return invalid("new_entries_not_allowed", "Nothing was saved. These aren't in the catalog, and this person doesn't let assistants add foods or units: "+list+". Use the food_id or unit_id of a match, search again under other names, or leave the unit out. If a match is the same thing under another name, add_food_names or add_unit_names can record that name.")
	}
	return invalid("new_entries_need_confirmation", "Nothing was saved. These would be new catalog entries: "+list+". Use the food_id or unit_id of a match where one fits. For the rest, ask the person whether to add them, and only after they agree, send the same call again with confirmed_new listing those names.")
}

// describeNewName names a new entry with up to three close matches from the catalog.
func (s *Service) describeNewName(ctx context.Context, v Viewer, n newName) (string, error) {
	var matches []string
	for _, word := range searchWords(n.name) {
		if len(matches) >= 3 || len([]rune(word)) < 3 {
			continue
		}
		if n.kind == "food" {
			page, err := s.ListFoods(ctx, v, word, 3, 0)
			if err != nil {
				return "", err
			}
			for _, f := range page.Items {
				matches = appendMatch(matches, fmt.Sprintf("food_id %d %q", f.ID, f.Name))
			}
		} else {
			page, err := s.ListUnits(ctx, v, word, 3, 0)
			if err != nil {
				return "", err
			}
			for _, u := range page.Items {
				matches = appendMatch(matches, fmt.Sprintf("unit_id %d %q", u.ID, u.Name))
			}
		}
	}
	if len(matches) > 3 {
		matches = matches[:3]
	}
	if len(matches) == 0 {
		return fmt.Sprintf("%s %q (no close match)", n.kind, n.name), nil
	}
	return fmt.Sprintf("%s %q (close: %s)", n.kind, n.name, strings.Join(matches, ", ")), nil
}

// searchWords is the whole name, then each word, then each word without its last two letters,
// so a plural or inflected form still finds the catalog's name.
func searchWords(name string) []string {
	words := append([]string{name}, strings.Fields(name)...)
	for _, w := range strings.Fields(name) {
		if r := []rune(w); len(r) > 5 {
			words = append(words, string(r[:len(r)-2]))
		}
	}
	return words
}

func appendMatch(list []string, m string) []string {
	for _, have := range list {
		if have == m {
			return list
		}
	}
	return append(list, m)
}

// mergeNames adds translations and aliases to existing ones without removing or renaming any.
// A language that already has a name only gains a missing plural or abbreviation.
func mergeNames(have []Translation, haveAliases []string, add []Translation, addAliases []string) ([]Translation, []string) {
	names := append([]Translation{}, have...)
	for _, t := range add {
		t.Locale, t.Name = strings.TrimSpace(t.Locale), strings.TrimSpace(t.Name)
		found := false
		for i := range names {
			if names[i].Locale == t.Locale {
				found = true
				if names[i].Plural == "" {
					names[i].Plural = strings.TrimSpace(t.Plural)
				}
				if names[i].Abbreviation == "" {
					names[i].Abbreviation = strings.TrimSpace(t.Abbreviation)
				}
			}
		}
		if !found {
			names = append(names, t)
		}
	}
	aliases := append([]string{}, haveAliases...)
	known := map[string]bool{}
	for _, n := range names {
		known[text.Normalize(n.Name)] = true
	}
	for _, a := range aliases {
		known[text.Normalize(a)] = true
	}
	for _, a := range addAliases {
		a = strings.TrimSpace(a)
		if a != "" && !known[text.Normalize(a)] {
			known[text.Normalize(a)] = true
			aliases = append(aliases, a)
		}
	}
	return names, aliases
}

// AddFoodNames gives a food more names: translations, plurals and aliases. Nothing is removed,
// and the same rule as editing decides who may do it.
func (s *Service) AddFoodNames(ctx context.Context, v Viewer, id int64, add []Translation, aliases []string) (Food, error) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		cur, err := s.foodDetail(ctx, q, v, id)
		if err != nil {
			return err
		}
		if !cur.CanEdit {
			return ErrForbidden
		}
		merged, all := mergeNames(cur.Translations, cur.Aliases, add, aliases)
		names, err := cleanTranslations(merged)
		if err != nil {
			return err
		}
		if err := writeFoodNames(ctx, q, id, names, all); err != nil {
			return err
		}
		return s.reindexFood(ctx, q, id) // Recipes become findable under the new names.
	})
	if err != nil {
		return Food{}, err
	}
	return s.GetFood(ctx, v, id)
}

// AddUnitNames gives a unit more names, the same way as AddFoodNames.
func (s *Service) AddUnitNames(ctx context.Context, v Viewer, id int64, add []Translation, aliases []string) (Unit, error) {
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		cur, err := s.unitDetail(ctx, q, v, id)
		if err != nil {
			return err
		}
		if !cur.CanEdit {
			return ErrForbidden
		}
		merged, all := mergeNames(cur.Translations, cur.Aliases, add, aliases)
		names, err := cleanTranslations(merged)
		if err != nil {
			return err
		}
		return writeUnitNames(ctx, q, id, names, all)
	})
	if err != nil {
		return Unit{}, err
	}
	return s.GetUnit(ctx, v, id)
}
