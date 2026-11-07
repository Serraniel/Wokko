package app

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

func foodRefs(ctx context.Context, q *sqlcgen.Queries, ids []int64, locale string) (map[int64]FoodRef, error) {
	out := map[int64]FoodRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.ListFoodNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	grouped := map[int64][]text.Translation[sqlcgen.FoodName]{}
	for _, r := range rows {
		grouped[r.FoodID] = append(grouped[r.FoodID], text.Translation[sqlcgen.FoodName]{Locale: r.Locale, Value: r})
	}
	for id, names := range grouped {
		n, _ := text.Pick(names, locale)
		out[id] = FoodRef{ID: id, Name: n.Name, Plural: n.Plural.String}
	}
	return out, nil
}

func unitRefs(ctx context.Context, q *sqlcgen.Queries, ids []int64, locale string) (map[int64]UnitRef, error) {
	out := map[int64]UnitRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.ListUnitNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	grouped := map[int64][]text.Translation[sqlcgen.UnitName]{}
	for _, r := range rows {
		grouped[r.UnitID] = append(grouped[r.UnitID], text.Translation[sqlcgen.UnitName]{Locale: r.Locale, Value: r})
	}
	for id, names := range grouped {
		n, _ := text.Pick(names, locale)
		out[id] = UnitRef{ID: id, Name: n.Name, Plural: n.Plural.String, Abbreviation: n.Abbreviation.String}
	}
	return out, nil
}

func keywordRefs(ctx context.Context, q *sqlcgen.Queries, ids []int64, locale string) (map[int64]Keyword, error) {
	out := map[int64]Keyword{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.ListKeywordNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	base, err := q.ListKeywordsRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	grouped := map[int64][]text.Translation[string]{}
	for _, r := range rows {
		grouped[r.KeywordID] = append(grouped[r.KeywordID], text.Translation[string]{Locale: r.Locale, Value: r.Name})
	}
	for _, k := range base {
		n, _ := text.Pick(grouped[k.ID], locale)
		out[k.ID] = Keyword{ID: k.ID, Name: n, CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt}
	}
	return out, nil
}

func clampPage(limit, offset int64) (int64, int64) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (s *Service) canEditCatalog(v Viewer, createdBy int64) bool {
	return v.isAdmin() || (createdBy != 0 && createdBy == v.id())
}

// ListFoods searches names in every language and aliases.
func (s *Service) ListFoods(ctx context.Context, v Viewer, query string, limit, offset int64) (Page[Food], error) {
	limit, offset = clampPage(limit, offset)
	q := s.Store.Queries
	n := text.EscapeLike(text.Normalize(query))
	ids, err := q.SearchFoodIDs(ctx, sqlcgen.SearchFoodIDsParams{Q: n, Lim: limit, Off: offset})
	if err != nil {
		return Page[Food]{}, err
	}
	total, err := q.CountFoods(ctx, n)
	if err != nil {
		return Page[Food]{}, err
	}
	items := make([]Food, 0, len(ids))
	for _, id := range ids {
		f, err := s.foodDetail(ctx, q, v, id)
		if err != nil {
			return Page[Food]{}, err
		}
		items = append(items, f)
	}
	return Page[Food]{Items: items, Total: total}, nil
}

func (s *Service) GetFood(ctx context.Context, v Viewer, id int64) (Food, error) {
	return s.foodDetail(ctx, s.Store.Queries, v, id)
}

func (s *Service) foodDetail(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (Food, error) {
	row, err := q.GetFood(ctx, id)
	if isNoRows(err) {
		return Food{}, ErrNotFound
	}
	if err != nil {
		return Food{}, err
	}
	refs, err := foodRefs(ctx, q, []int64{id}, v.locale())
	if err != nil {
		return Food{}, err
	}
	names, err := q.ListFoodNames(ctx, []int64{id})
	if err != nil {
		return Food{}, err
	}
	aliases, err := q.ListFoodAliases(ctx, []int64{id})
	if err != nil {
		return Food{}, err
	}
	uses, err := q.CountFoodUses(ctx, id)
	if err != nil {
		return Food{}, err
	}
	f := Food{FoodRef: refs[id], Uses: uses, CanEdit: s.canEditCatalog(v, row.CreatedBy.Int64), Translations: []Translation{}, Aliases: []string{}}
	for _, n := range names {
		f.Translations = append(f.Translations, Translation{Locale: n.Locale, Name: n.Name, Plural: n.Plural.String})
	}
	sort.Slice(f.Translations, func(i, j int) bool { return f.Translations[i].Locale < f.Translations[j].Locale })
	for _, a := range aliases {
		f.Aliases = append(f.Aliases, a.Alias)
	}
	// Suggest the unit this food is most often used with, then its catalog default.
	unitID := row.DefaultUnitID.Int64
	if used, err := q.MostUsedUnitForFood(ctx, id); err == nil && used.Valid {
		unitID = used.Int64
	} else if err != nil && !isNoRows(err) {
		return Food{}, err
	}
	if unitID > 0 {
		units, err := unitRefs(ctx, q, []int64{unitID}, v.locale())
		if err != nil {
			return Food{}, err
		}
		if u, ok := units[unitID]; ok {
			f.DefaultUnit = &u
		}
	}
	return f, nil
}

func cleanTranslations(in []Translation) ([]Translation, error) {
	seen := map[string]bool{}
	var out []Translation
	for _, t := range in {
		t.Locale = strings.TrimSpace(t.Locale)
		t.Name = strings.TrimSpace(t.Name)
		t.Plural = strings.TrimSpace(t.Plural)
		t.Abbreviation = strings.TrimSpace(t.Abbreviation)
		if t.Name == "" {
			continue
		}
		if !text.ValidLocale(t.Locale) {
			return nil, invalid("bad_locale", "Unknown language")
		}
		if len(t.Name) > 200 || len(t.Plural) > 200 || len(t.Abbreviation) > 40 {
			return nil, invalid("name_too_long", "This name is too long")
		}
		if seen[t.Locale] {
			return nil, invalid("duplicate_locale", "A language appears twice")
		}
		seen[t.Locale] = true
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, invalid("name_required", "Please enter a name")
	}
	return out, nil
}

func (s *Service) CreateFood(ctx context.Context, v Viewer, in FoodInput) (Food, error) {
	names, err := cleanTranslations(in.Translations)
	if err != nil {
		return Food{}, err
	}
	var id int64
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		var def int64
		if in.DefaultUnitID != nil {
			def = *in.DefaultUnitID
		}
		f, err := q.CreateFood(ctx, sqlcgen.CreateFoodParams{DefaultUnitID: nullID(def), CreatedBy: nullID(v.id())})
		if err != nil {
			return err
		}
		id = f.ID
		return writeFoodNames(ctx, q, id, names, in.Aliases)
	})
	if err != nil {
		return Food{}, err
	}
	return s.GetFood(ctx, v, id)
}

func writeFoodNames(ctx context.Context, q *sqlcgen.Queries, id int64, names []Translation, aliases []string) error {
	if err := q.DeleteFoodNames(ctx, id); err != nil {
		return err
	}
	for _, n := range names {
		if err := q.UpsertFoodName(ctx, sqlcgen.UpsertFoodNameParams{FoodID: id, Locale: n.Locale, Name: n.Name, Plural: nullStr(n.Plural), SearchName: text.Normalize(n.Name + " " + n.Plural)}); err != nil {
			return err
		}
	}
	if err := q.DeleteFoodAliases(ctx, id); err != nil {
		return err
	}
	for _, a := range aliases {
		if a = strings.TrimSpace(a); a != "" {
			if err := q.AddFoodAlias(ctx, sqlcgen.AddFoodAliasParams{FoodID: id, Alias: a, SearchName: text.Normalize(a)}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) UpdateFood(ctx context.Context, v Viewer, id int64, in FoodInput) (Food, error) {
	names, err := cleanTranslations(in.Translations)
	if err != nil {
		return Food{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.GetFood(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !s.canEditCatalog(v, row.CreatedBy.Int64) {
			return ErrForbidden
		}
		var def int64
		if in.DefaultUnitID != nil {
			def = *in.DefaultUnitID
		}
		if err := q.SetFoodDefaultUnit(ctx, sqlcgen.SetFoodDefaultUnitParams{DefaultUnitID: nullID(def), ID: id}); err != nil {
			return err
		}
		return writeFoodNames(ctx, q, id, names, in.Aliases)
	})
	if err != nil {
		return Food{}, err
	}
	return s.GetFood(ctx, v, id)
}

func inUse(uses int64) error {
	if uses > 0 {
		return fail(http.StatusConflict, "in_use", "Recipes still use this. Merge it into another entry instead")
	}
	return nil
}

func (s *Service) DeleteFood(ctx context.Context, v Viewer, id int64) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		uses, err := q.CountFoodUses(ctx, id)
		if err != nil {
			return err
		}
		if err := inUse(uses); err != nil {
			return err
		}
		return q.DeleteFood(ctx, id)
	})
}

// MergeFoods moves every use of source to target in one transaction. The merged name stays
// searchable as an alias of the target.
func (s *Service) MergeFoods(ctx context.Context, v Viewer, in MergeInput) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	if in.SourceID == in.TargetID {
		return invalid("merge_same", "Choose two different entries")
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := q.GetFood(ctx, in.TargetID); isNoRows(err) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		names, err := q.ListFoodNames(ctx, []int64{in.SourceID})
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return ErrNotFound
		}
		for _, n := range names {
			if err := q.AddFoodAlias(ctx, sqlcgen.AddFoodAliasParams{FoodID: in.TargetID, Alias: n.Name, SearchName: text.Normalize(n.Name)}); err != nil {
				return err
			}
		}
		if err := q.MoveIngredientsToFood(ctx, sqlcgen.MoveIngredientsToFoodParams{Target: in.TargetID, Source: in.SourceID}); err != nil {
			return err
		}
		return q.DeleteFood(ctx, in.SourceID)
	})
}

// --- units ---

func (s *Service) ListUnits(ctx context.Context, v Viewer, query string, limit, offset int64) (Page[Unit], error) {
	limit, offset = clampPage(limit, offset)
	q := s.Store.Queries
	n := text.EscapeLike(text.Normalize(query))
	ids, err := q.SearchUnitIDs(ctx, sqlcgen.SearchUnitIDsParams{Q: n, Lim: limit, Off: offset})
	if err != nil {
		return Page[Unit]{}, err
	}
	total, err := q.CountUnits(ctx, n)
	if err != nil {
		return Page[Unit]{}, err
	}
	items := make([]Unit, 0, len(ids))
	for _, id := range ids {
		u, err := s.unitDetail(ctx, q, v, id)
		if err != nil {
			return Page[Unit]{}, err
		}
		items = append(items, u)
	}
	return Page[Unit]{Items: items, Total: total}, nil
}

func (s *Service) GetUnit(ctx context.Context, v Viewer, id int64) (Unit, error) {
	return s.unitDetail(ctx, s.Store.Queries, v, id)
}

func (s *Service) unitDetail(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (Unit, error) {
	row, err := q.GetUnit(ctx, id)
	if isNoRows(err) {
		return Unit{}, ErrNotFound
	}
	if err != nil {
		return Unit{}, err
	}
	refs, err := unitRefs(ctx, q, []int64{id}, v.locale())
	if err != nil {
		return Unit{}, err
	}
	names, err := q.ListUnitNames(ctx, []int64{id})
	if err != nil {
		return Unit{}, err
	}
	aliases, err := q.ListUnitAliases(ctx, []int64{id})
	if err != nil {
		return Unit{}, err
	}
	uses, err := q.CountUnitUses(ctx, sql64(id))
	if err != nil {
		return Unit{}, err
	}
	u := Unit{UnitRef: refs[id], Uses: uses, CanEdit: s.canEditCatalog(v, row.CreatedBy.Int64), Translations: []Translation{}, Aliases: []string{}}
	for _, n := range names {
		u.Translations = append(u.Translations, Translation{Locale: n.Locale, Name: n.Name, Plural: n.Plural.String, Abbreviation: n.Abbreviation.String})
	}
	sort.Slice(u.Translations, func(i, j int) bool { return u.Translations[i].Locale < u.Translations[j].Locale })
	for _, a := range aliases {
		u.Aliases = append(u.Aliases, a.Alias)
	}
	return u, nil
}

func writeUnitNames(ctx context.Context, q *sqlcgen.Queries, id int64, names []Translation, aliases []string) error {
	if err := q.DeleteUnitNames(ctx, id); err != nil {
		return err
	}
	for _, n := range names {
		if err := q.UpsertUnitName(ctx, sqlcgen.UpsertUnitNameParams{UnitID: id, Locale: n.Locale, Name: n.Name, Plural: nullStr(n.Plural), Abbreviation: nullStr(n.Abbreviation), SearchName: text.Normalize(n.Name + " " + n.Abbreviation)}); err != nil {
			return err
		}
	}
	if err := q.DeleteUnitAliases(ctx, id); err != nil {
		return err
	}
	for _, a := range aliases {
		if a = strings.TrimSpace(a); a != "" {
			if err := q.AddUnitAlias(ctx, sqlcgen.AddUnitAliasParams{UnitID: id, Alias: a, SearchName: text.Normalize(a)}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) CreateUnit(ctx context.Context, v Viewer, in UnitInput) (Unit, error) {
	names, err := cleanTranslations(in.Translations)
	if err != nil {
		return Unit{}, err
	}
	var id int64
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		u, err := q.CreateUnit(ctx, nullID(v.id()))
		if err != nil {
			return err
		}
		id = u.ID
		return writeUnitNames(ctx, q, id, names, in.Aliases)
	})
	if err != nil {
		return Unit{}, err
	}
	return s.GetUnit(ctx, v, id)
}

func (s *Service) UpdateUnit(ctx context.Context, v Viewer, id int64, in UnitInput) (Unit, error) {
	names, err := cleanTranslations(in.Translations)
	if err != nil {
		return Unit{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.GetUnit(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !s.canEditCatalog(v, row.CreatedBy.Int64) {
			return ErrForbidden
		}
		return writeUnitNames(ctx, q, id, names, in.Aliases)
	})
	if err != nil {
		return Unit{}, err
	}
	return s.GetUnit(ctx, v, id)
}

func (s *Service) DeleteUnit(ctx context.Context, v Viewer, id int64) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		uses, err := q.CountUnitUses(ctx, sql64(id))
		if err != nil {
			return err
		}
		if err := inUse(uses); err != nil {
			return err
		}
		return q.DeleteUnit(ctx, id)
	})
}

func (s *Service) MergeUnits(ctx context.Context, v Viewer, in MergeInput) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	if in.SourceID == in.TargetID {
		return invalid("merge_same", "Choose two different entries")
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := q.GetUnit(ctx, in.TargetID); isNoRows(err) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		names, err := q.ListUnitNames(ctx, []int64{in.SourceID})
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return ErrNotFound
		}
		for _, n := range names {
			if err := q.AddUnitAlias(ctx, sqlcgen.AddUnitAliasParams{UnitID: in.TargetID, Alias: n.Name, SearchName: text.Normalize(n.Name)}); err != nil {
				return err
			}
		}
		if err := q.MoveIngredientsToUnit(ctx, sqlcgen.MoveIngredientsToUnitParams{Target: nullID(in.TargetID), Source: nullID(in.SourceID)}); err != nil {
			return err
		}
		if err := q.MoveFoodDefaultsToUnit(ctx, sqlcgen.MoveFoodDefaultsToUnitParams{Target: nullID(in.TargetID), Source: nullID(in.SourceID)}); err != nil {
			return err
		}
		return q.DeleteUnit(ctx, in.SourceID)
	})
}

// --- keywords ---

func (s *Service) ListKeywords(ctx context.Context, v Viewer, query string, limit, offset int64) (Page[Keyword], error) {
	limit, offset = clampPage(limit, offset)
	q := s.Store.Queries
	n := text.EscapeLike(text.Normalize(query))
	ids, err := q.SearchKeywordIDs(ctx, sqlcgen.SearchKeywordIDsParams{Q: n, Lim: limit, Off: offset})
	if err != nil {
		return Page[Keyword]{}, err
	}
	total, err := q.CountKeywords(ctx, n)
	if err != nil {
		return Page[Keyword]{}, err
	}
	refs, err := keywordRefs(ctx, q, ids, v.locale())
	if err != nil {
		return Page[Keyword]{}, err
	}
	items := make([]Keyword, 0, len(ids))
	for _, id := range ids {
		items = append(items, refs[id])
	}
	return Page[Keyword]{Items: items, Total: total}, nil
}

func (s *Service) GetKeyword(ctx context.Context, v Viewer, id int64) (Keyword, error) {
	refs, err := keywordRefs(ctx, s.Store.Queries, []int64{id}, v.locale())
	if err != nil {
		return Keyword{}, err
	}
	k, ok := refs[id]
	if !ok {
		return Keyword{}, ErrNotFound
	}
	return k, nil
}
