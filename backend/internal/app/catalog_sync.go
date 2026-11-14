package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/serraniel/wokko/backend/internal/catalog"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

// SyncCatalog brings the starter catalog this build ships into the database. It adds new
// entries, updates entries nobody edited, and turns everything else into suggestions for the
// admin. It never deletes, never brings back what was deleted or merged here, and never changes
// a person's own entries on its own. It does nothing when this catalog was already brought in.
func (s *Service) SyncCatalog(ctx context.Context) (CatalogSync, error) {
	pack := s.Catalog
	if last, err := s.Store.Queries.LastCatalogSync(ctx); err == nil && last.Pack == pack.Hash {
		return toCatalogSync(last), nil
	} else if err != nil && !isNoRows(err) {
		return CatalogSync{}, err
	}
	var out CatalogSync
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		out = CatalogSync{Pack: pack.Hash, Version: s.Cfg.Version}
		// Units first, so foods can refer to them by key.
		for _, kind := range []string{catalog.Unit, catalog.Food, catalog.Tag} {
			for _, e := range pack.Entries[kind] {
				if err := s.syncEntry(ctx, q, kind, e, &out); err != nil {
					return err
				}
			}
		}
		return q.AddCatalogSync(ctx, sqlcgen.AddCatalogSyncParams{Pack: out.Pack, Version: out.Version, Added: out.Added, Updated: out.Updated, Suggested: out.Suggested})
	})
	if err != nil {
		return CatalogSync{}, err
	}
	s.Log.Info("starter catalog brought in", "added", out.Added, "updated", out.Updated, "suggested", out.Suggested)
	return out, nil
}

func (s *Service) syncEntry(ctx context.Context, q *sqlcgen.Queries, kind string, e catalog.Entry, out *CatalogSync) error {
	row, err := q.GetCatalogEntry(ctx, sqlcgen.GetCatalogEntryParams{Kind: kind, Key: e.Key})
	base := catalog.Fields{}
	switch {
	case isNoRows(err):
		id, created, err := s.adoptEntry(ctx, q, kind, e)
		if err != nil {
			return err
		}
		if id == 0 {
			// It came with the original seed, but was deleted or merged here. It stays gone.
			return q.SaveCatalogEntry(ctx, sqlcgen.SaveCatalogEntryParams{Kind: kind, Key: e.Key, Base: "{}"})
		}
		if orig, ok := s.Catalog.Original[kind+"/"+e.Key]; ok && orig.ID == id {
			base = catalog.FieldsOf(orig)
		}
		if created {
			out.Added++
		}
		row = sqlcgen.CatalogEntry{Kind: kind, Key: e.Key, EntityID: sql64(id)}
	case err != nil:
		return err
	default:
		if !row.EntityID.Valid {
			return nil
		}
		if err := json.Unmarshal([]byte(row.Base), &base); err != nil {
			return err
		}
	}
	id := row.EntityID.Int64
	ours, owned, exists, err := s.catalogFields(ctx, q, kind, id)
	if err != nil {
		return err
	}
	if !exists {
		return q.ForgetCatalogEntity(ctx, sqlcgen.ForgetCatalogEntityParams{Kind: kind, EntityID: sql64(id)})
	}
	res := catalog.Merge(base, ours, catalog.FieldsOf(e), !owned)
	if len(res.Apply) > 0 {
		if err := s.writeCatalogFields(ctx, q, kind, id, ours, res.Apply); err != nil {
			return err
		}
		if base := len(base); base > 0 {
			out.Updated++
		}
	}
	if err := q.ClearCatalogSuggestions(ctx, sqlcgen.ClearCatalogSuggestionsParams{Kind: kind, Key: e.Key}); err != nil {
		return err
	}
	for _, c := range res.Conflicts {
		if err := q.AddCatalogSuggestion(ctx, sqlcgen.AddCatalogSuggestionParams{Kind: kind, Key: e.Key, EntityID: id, Field: c.Field, Ours: c.Ours, Theirs: c.Theirs}); err != nil {
			return err
		}
		out.Suggested++
	}
	raw, err := json.Marshal(res.Base)
	if err != nil {
		return err
	}
	return q.SaveCatalogEntry(ctx, sqlcgen.SaveCatalogEntryParams{Kind: kind, Key: e.Key, EntityID: sql64(id), Base: string(raw)})
}

// adoptEntry finds the row an entry belongs to: the one the original seed created, or one that
// already has one of its names, or a new one. id is 0 when a seeded row no longer exists.
func (s *Service) adoptEntry(ctx context.Context, q *sqlcgen.Queries, kind string, e catalog.Entry) (id int64, created bool, err error) {
	if e.ID > 0 {
		_, _, exists, err := s.catalogFields(ctx, q, kind, e.ID)
		if err != nil || !exists {
			return 0, false, err
		}
		// Unless another entry already took it, which only a merge here can cause.
		n, err := q.CatalogEntityIsEntry(ctx, sqlcgen.CatalogEntityIsEntryParams{Kind: kind, EntityID: sql64(e.ID)})
		if err != nil || n > 0 {
			return 0, false, err
		}
		return e.ID, false, nil
	}
	// A new entry someone may already have under the same name. Then that row gets the entry's
	// missing values as suggestions rather than a duplicate.
	for _, n := range e.Names {
		for _, name := range []string{n.Name, n.Plural, n.Abbreviation} {
			if name = strings.TrimSpace(name); name == "" {
				continue
			}
			var found int64
			var ok bool
			switch kind {
			case catalog.Food:
				found, ok, err = lookupFood(ctx, q, name)
			case catalog.Unit:
				found, ok, err = lookupUnit(ctx, q, name)
			default:
				found, err = q.FindKeywordBySearchName(ctx, text.Normalize(name))
				ok = err == nil
				if isNoRows(err) {
					err = nil
				}
			}
			if err != nil {
				return 0, false, err
			}
			if ok {
				return found, false, nil
			}
		}
	}
	switch kind {
	case catalog.Food:
		id, err = q.CreateCatalogFood(ctx)
	case catalog.Unit:
		id, err = q.CreateCatalogUnit(ctx)
	default:
		var k sqlcgen.Keyword
		k, err = q.CreateKeyword(ctx)
		id = k.ID
	}
	return id, err == nil, err
}

// catalogFields reads a row the way the catalog describes entries. owned is true for a
// person's own food or unit, which only changes when the admin agrees.
func (s *Service) catalogFields(ctx context.Context, q *sqlcgen.Queries, kind string, id int64) (f catalog.Fields, owned, exists bool, err error) {
	f = catalog.Fields{}
	ids := []int64{id}
	switch kind {
	case catalog.Food:
		row, err := q.GetFood(ctx, id)
		if isNoRows(err) {
			return nil, false, false, nil
		} else if err != nil {
			return nil, false, false, err
		}
		owned = row.CreatedBy.Valid
		names, err := q.ListFoodNames(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, n := range names {
			f["name/"+n.Locale] = catalog.EncodeName(catalog.Name{Name: n.Name, Plural: n.Plural.String})
		}
		aliases, err := q.ListFoodAliases(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, a := range aliases {
			f["alias/"+text.Normalize(a.Alias)] = a.Alias
		}
		if row.DefaultUnitID.Valid {
			f["default_unit"], err = s.unitKey(ctx, q, row.DefaultUnitID.Int64)
			if err != nil {
				return nil, false, false, err
			}
		}
		values, err := q.ListFoodNutrients(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, v := range values {
			f["nutrient/"+strconv.FormatInt(v.NutrientID, 10)] = catalog.Number(v.Per100g)
		}
	case catalog.Unit:
		row, err := q.GetUnit(ctx, id)
		if isNoRows(err) {
			return nil, false, false, nil
		} else if err != nil {
			return nil, false, false, err
		}
		owned = row.CreatedBy.Valid
		names, err := q.ListUnitNames(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, n := range names {
			f["name/"+n.Locale] = catalog.EncodeName(catalog.Name{Name: n.Name, Plural: n.Plural.String, Abbreviation: n.Abbreviation.String})
		}
		aliases, err := q.ListUnitAliases(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, a := range aliases {
			f["alias/"+text.Normalize(a.Alias)] = a.Alias
		}
		if row.Grams.Valid {
			f["grams"] = catalog.Number(row.Grams.Float64)
		}
	default:
		if _, err := q.GetKeyword(ctx, id); isNoRows(err) {
			return nil, false, false, nil
		} else if err != nil {
			return nil, false, false, err
		}
		names, err := q.ListKeywordNames(ctx, ids)
		if err != nil {
			return nil, false, false, err
		}
		for _, n := range names {
			f["name/"+n.Locale] = catalog.EncodeName(catalog.Name{Name: n.Name})
		}
	}
	return f, owned, true, nil
}

// unitKey names a unit by its catalog key, or by its id when it isn't from the catalog, so it
// never looks equal to a catalog value.
func (s *Service) unitKey(ctx context.Context, q *sqlcgen.Queries, id int64) (string, error) {
	rows, err := q.ListCatalogEntries(ctx, catalog.Unit)
	if err != nil {
		return "", err
	}
	for _, r := range rows {
		if r.EntityID.Valid && r.EntityID.Int64 == id {
			return r.Key, nil
		}
	}
	return "id:" + strconv.FormatInt(id, 10), nil
}

func (s *Service) unitByKey(ctx context.Context, q *sqlcgen.Queries, key string) (int64, bool, error) {
	row, err := q.GetCatalogEntry(ctx, sqlcgen.GetCatalogEntryParams{Kind: catalog.Unit, Key: key})
	if isNoRows(err) || (err == nil && !row.EntityID.Valid) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return row.EntityID.Int64, true, nil
}

// writeCatalogFields writes changed values to a row. ours is the row as it is, so names and
// aliases are written as a whole.
func (s *Service) writeCatalogFields(ctx context.Context, q *sqlcgen.Queries, kind string, id int64, ours, changes catalog.Fields) error {
	all := catalog.Fields{}
	for k, v := range ours {
		all[k] = v
	}
	namesChanged := false
	for k, v := range changes {
		all[k] = v
		if strings.HasPrefix(k, "name/") || strings.HasPrefix(k, "alias/") {
			namesChanged = true
		}
	}
	var names []Translation
	var aliases []string
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		switch {
		case strings.HasPrefix(k, "name/"):
			n := catalog.DecodeName(all[k])
			names = append(names, Translation{Locale: strings.TrimPrefix(k, "name/"), Name: n.Name, Plural: n.Plural, Abbreviation: n.Abbreviation})
		case strings.HasPrefix(k, "alias/"):
			aliases = append(aliases, all[k])
		}
	}
	switch kind {
	case catalog.Food:
		if namesChanged {
			if err := writeFoodNames(ctx, q, id, names, aliases); err != nil {
				return err
			}
			if err := s.reindexFood(ctx, q, id); err != nil {
				return err
			}
		}
		for k, v := range changes {
			switch {
			case k == "default_unit":
				unit, ok, err := s.unitByKey(ctx, q, v)
				if err != nil {
					return err
				}
				if ok {
					if err := q.SetFoodDefaultUnit(ctx, sqlcgen.SetFoodDefaultUnitParams{DefaultUnitID: sql64(unit), ID: id}); err != nil {
						return err
					}
				}
			case strings.HasPrefix(k, "nutrient/"):
				nid, err := strconv.ParseInt(strings.TrimPrefix(k, "nutrient/"), 10, 64)
				if err != nil {
					return err
				}
				value, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				if err := q.SetFoodNutrient(ctx, sqlcgen.SetFoodNutrientParams{FoodID: id, NutrientID: nid, Per100g: value}); err != nil {
					return err
				}
			}
		}
	case catalog.Unit:
		if namesChanged {
			if err := writeUnitNames(ctx, q, id, names, aliases); err != nil {
				return err
			}
		}
		if v, ok := changes["grams"]; ok {
			g, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			if err := q.SetUnitGrams(ctx, sqlcgen.SetUnitGramsParams{Grams: sql.NullFloat64{Float64: g, Valid: true}, ID: id}); err != nil {
				return err
			}
		}
	default:
		for _, n := range names {
			if err := q.UpsertKeywordName(ctx, sqlcgen.UpsertKeywordNameParams{KeywordID: id, Locale: n.Locale, Name: n.Name, SearchName: text.Normalize(n.Name)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// CatalogStatus is what the admin sees: the last update and the open suggestions.
func (s *Service) CatalogStatus(ctx context.Context, v Viewer) (CatalogStatus, error) {
	if !v.isAdmin() {
		return CatalogStatus{}, ErrForbidden
	}
	q := s.Store.Queries
	out := CatalogStatus{Suggestions: []CatalogSuggestion{}}
	if last, err := q.LastCatalogSync(ctx); err == nil {
		l := toCatalogSync(last)
		out.Last = &l
	} else if !isNoRows(err) {
		return CatalogStatus{}, err
	}
	rows, err := q.ListCatalogSuggestions(ctx)
	if err != nil {
		return CatalogStatus{}, err
	}
	nutrients, err := q.ListNutrients(ctx, v.locale())
	if err != nil {
		return CatalogStatus{}, err
	}
	for _, r := range rows {
		sg := CatalogSuggestion{ID: r.ID, Kind: r.Kind, Key: r.Key, EntityID: r.EntityID, Field: r.Field, Ours: displayField(r.Field, r.Ours), Theirs: displayField(r.Field, r.Theirs)}
		switch r.Kind {
		case catalog.Food:
			refs, err := foodRefs(ctx, q, []int64{r.EntityID}, v.locale())
			if err != nil {
				return CatalogStatus{}, err
			}
			sg.Name = refs[r.EntityID].Name
		case catalog.Unit:
			refs, err := unitRefs(ctx, q, []int64{r.EntityID}, v.locale())
			if err != nil {
				return CatalogStatus{}, err
			}
			sg.Name = refs[r.EntityID].Name
		default:
			k, err := s.GetKeyword(ctx, v, r.EntityID)
			if err != nil && err != ErrNotFound {
				return CatalogStatus{}, err
			}
			sg.Name = k.Name
		}
		if strings.HasPrefix(r.Field, "nutrient/") {
			for _, n := range nutrients {
				if "nutrient/"+strconv.FormatInt(n.ID, 10) == r.Field {
					sg.Nutrient = n.Name
					sg.NutrientUnit = n.Unit
				}
			}
		}
		out.Suggestions = append(out.Suggestions, sg)
	}
	return out, nil
}

// displayField turns a stored value into text for people.
func displayField(field, value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(field, "name/") {
		n := catalog.DecodeName(value)
		parts := []string{n.Name}
		if n.Plural != "" && n.Plural != n.Name {
			parts = append(parts, n.Plural)
		}
		s := strings.Join(parts, " / ")
		if n.Abbreviation != "" {
			s += " (" + n.Abbreviation + ")"
		}
		return s
	}
	return value
}

// ResolveCatalogSuggestion takes the catalog's value, or keeps the database's. Either way the
// question doesn't come back until a newer catalog changes that value again.
func (s *Service) ResolveCatalogSuggestion(ctx context.Context, v Viewer, id int64, take bool) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		sg, err := q.GetCatalogSuggestion(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		row, err := q.GetCatalogEntry(ctx, sqlcgen.GetCatalogEntryParams{Kind: sg.Kind, Key: sg.Key})
		if err != nil {
			return err
		}
		base := catalog.Fields{}
		if err := json.Unmarshal([]byte(row.Base), &base); err != nil {
			return err
		}
		if take {
			ours, _, exists, err := s.catalogFields(ctx, q, sg.Kind, sg.EntityID)
			if err != nil {
				return err
			}
			if exists {
				if err := s.writeCatalogFields(ctx, q, sg.Kind, sg.EntityID, ours, catalog.Fields{sg.Field: sg.Theirs}); err != nil {
					return err
				}
			}
		}
		base[sg.Field] = sg.Theirs
		raw, err := json.Marshal(base)
		if err != nil {
			return err
		}
		if err := q.SaveCatalogEntry(ctx, sqlcgen.SaveCatalogEntryParams{Kind: row.Kind, Key: row.Key, EntityID: row.EntityID, Base: string(raw)}); err != nil {
			return err
		}
		return q.DeleteCatalogSuggestion(ctx, id)
	})
}

func toCatalogSync(r sqlcgen.CatalogSync) CatalogSync {
	return CatalogSync{Pack: r.Pack, Version: r.Version, Added: r.Added, Updated: r.Updated, Suggested: r.Suggested, At: r.CreatedAt}
}
