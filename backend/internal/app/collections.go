package app

import (
	"context"
	"strings"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

// --- recipe books ---

func cleanBook(in *RecipeBookInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" {
		return invalid("name_required", "Please enter a name")
	}
	if len(in.Name) > 200 {
		return invalid("name_too_long", "This name is too long")
	}
	return nil
}

// bookAccess finds a book the viewer may see. Books are shared across a household, and
// anyone outside it gets the same answer as if the book didn't exist.
func (s *Service) bookAccess(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) (sqlcgen.RecipeBook, error) {
	b, err := q.GetBook(ctx, id)
	if isNoRows(err) {
		return b, ErrNotFound
	}
	if err != nil {
		return b, err
	}
	owner, err := q.GetUser(ctx, b.OwnerID)
	if err != nil {
		return b, err
	}
	if v.User == nil || owner.HouseholdID != v.User.HouseholdID {
		return b, ErrNotFound
	}
	return b, nil
}

// bookRecipes returns only the recipes this viewer may see. A book can hold someone's
// private recipe, which stays invisible to everyone else in the household.
func (s *Service) bookRecipes(ctx context.Context, q *sqlcgen.Queries, v Viewer, id int64) ([]sqlcgen.Recipe, error) {
	entries, err := q.ListBookEntries(ctx, id)
	if err != nil {
		return nil, err
	}
	out := []sqlcgen.Recipe{}
	for _, e := range entries {
		r, err := q.GetRecipe(ctx, e.RecipeID)
		if err != nil {
			return nil, err
		}
		if r.DeletedAt.Valid {
			continue
		}
		ok, err := s.canView(ctx, q, v, r)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Service) toBook(ctx context.Context, q *sqlcgen.Queries, v Viewer, b sqlcgen.RecipeBook) (RecipeBook, error) {
	rows, err := s.bookRecipes(ctx, q, v, b.ID)
	if err != nil {
		return RecipeBook{}, err
	}
	return RecipeBook{ID: b.ID, Name: b.Name, Description: b.Description, RecipeCount: int64(len(rows)), CanEdit: v.User != nil, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt}, nil
}

func (s *Service) ListBooks(ctx context.Context, v Viewer) ([]RecipeBook, error) {
	q := s.Store.Queries
	rows, err := q.ListBooksForHousehold(ctx, v.User.HouseholdID)
	if err != nil {
		return nil, err
	}
	out := make([]RecipeBook, 0, len(rows))
	for _, b := range rows {
		book, err := s.toBook(ctx, q, v, b)
		if err != nil {
			return nil, err
		}
		out = append(out, book)
	}
	return out, nil
}

func (s *Service) GetBook(ctx context.Context, v Viewer, id int64) (RecipeBookDetail, error) {
	q := s.Store.Queries
	b, err := s.bookAccess(ctx, q, v, id)
	if err != nil {
		return RecipeBookDetail{}, err
	}
	rows, err := s.bookRecipes(ctx, q, v, id)
	if err != nil {
		return RecipeBookDetail{}, err
	}
	items, err := s.summaries(ctx, q, v, rows)
	if err != nil {
		return RecipeBookDetail{}, err
	}
	book, err := s.toBook(ctx, q, v, b)
	if err != nil {
		return RecipeBookDetail{}, err
	}
	return RecipeBookDetail{RecipeBook: book, Recipes: items}, nil
}

func (s *Service) CreateBook(ctx context.Context, v Viewer, in RecipeBookInput) (RecipeBook, error) {
	if err := cleanBook(&in); err != nil {
		return RecipeBook{}, err
	}
	var out RecipeBook
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		b, err := q.CreateBook(ctx, sqlcgen.CreateBookParams{OwnerID: v.id(), Name: in.Name, Description: in.Description})
		if err != nil {
			return err
		}
		out, err = s.toBook(ctx, q, v, b)
		return err
	})
	return out, err
}

func (s *Service) UpdateBook(ctx context.Context, v Viewer, id int64, in RecipeBookInput) (RecipeBook, error) {
	if err := cleanBook(&in); err != nil {
		return RecipeBook{}, err
	}
	var out RecipeBook
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.bookAccess(ctx, q, v, id); err != nil {
			return err
		}
		if err := q.UpdateBook(ctx, sqlcgen.UpdateBookParams{Name: in.Name, Description: in.Description, ID: id}); err != nil {
			return err
		}
		b, err := q.GetBook(ctx, id)
		if err != nil {
			return err
		}
		out, err = s.toBook(ctx, q, v, b)
		return err
	})
	return out, err
}

// DeleteBook removes the book only. Its recipes are not touched.
func (s *Service) DeleteBook(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.bookAccess(ctx, q, v, id); err != nil {
			return err
		}
		return q.DeleteBook(ctx, id)
	})
}

// AddToBook adds a recipe the viewer can see. It returns the entry's ID, which kitshn needs.
func (s *Service) AddToBook(ctx context.Context, v Viewer, bookID, recipeID int64) (int64, error) {
	var entry int64
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.bookAccess(ctx, q, v, bookID); err != nil {
			return err
		}
		r, err := q.GetRecipe(ctx, recipeID)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if ok, err := s.canView(ctx, q, v, r); err != nil {
			return err
		} else if !ok || r.DeletedAt.Valid {
			return ErrNotFound
		}
		e, err := q.AddBookEntry(ctx, sqlcgen.AddBookEntryParams{BookID: bookID, RecipeID: recipeID, AddedBy: nullID(v.id())})
		entry = e.ID
		return err
	})
	return entry, err
}

func (s *Service) RemoveFromBook(ctx context.Context, v Viewer, bookID, recipeID int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if _, err := s.bookAccess(ctx, q, v, bookID); err != nil {
			return err
		}
		return q.RemoveBookEntry(ctx, sqlcgen.RemoveBookEntryParams{BookID: bookID, RecipeID: recipeID})
	})
}

// BooksContaining lists the household's books that hold a recipe, for the recipe page.
func (s *Service) BooksContaining(ctx context.Context, v Viewer, recipeID int64) ([]int64, error) {
	q := s.Store.Queries
	entries, err := q.ListBookEntriesForRecipe(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	out := []int64{}
	for _, e := range entries {
		if _, err := s.bookAccess(ctx, q, v, e.BookID); err == nil {
			out = append(out, e.BookID)
		} else if err != ErrNotFound {
			return nil, err
		}
	}
	return out, nil
}

// --- a link to all of someone's recipes ---

func (s *Service) toCollectionLink(l sqlcgen.CollectionLink) CollectionLink {
	return CollectionLink{ID: l.ID, URL: s.Cfg.PublicURL + "/shared/" + l.Token, ExpiresAt: l.ExpiresAt.String, CreatedAt: l.CreatedAt}
}

func (s *Service) ListCollectionLinks(ctx context.Context, v Viewer) ([]CollectionLink, error) {
	rows, err := s.Store.Queries.ListCollectionLinks(ctx, v.id())
	if err != nil {
		return nil, err
	}
	out := make([]CollectionLink, len(rows))
	for i, l := range rows {
		out[i] = s.toCollectionLink(l)
	}
	return out, nil
}

func (s *Service) CreateCollectionLink(ctx context.Context, v Viewer, expiresInDays int64) (CollectionLink, error) {
	token, err := auth.NewToken()
	if err != nil {
		return CollectionLink{}, err
	}
	var out CollectionLink
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		p := sqlcgen.CreateCollectionLinkParams{Token: token, OwnerID: v.id()}
		if expiresInDays > 0 {
			p.ExpiresAt = nullStr(s.stamp(s.Now().AddDate(0, 0, int(expiresInDays))))
		}
		l, err := q.CreateCollectionLink(ctx, p)
		if err != nil {
			return err
		}
		out = s.toCollectionLink(l)
		return nil
	})
	return out, err
}

func (s *Service) RevokeCollectionLink(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.DeleteCollectionLink(ctx, sqlcgen.DeleteCollectionLinkParams{ID: id, OwnerID: v.id()})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Service) collectionOwner(ctx context.Context, token string) (int64, error) {
	l, err := s.Store.Queries.GetCollectionLinkByToken(ctx, token)
	if isNoRows(err) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if l.ExpiresAt.Valid && l.ExpiresAt.String <= s.stamp(s.Now()) {
		return 0, ErrNotFound
	}
	return l.OwnerID, nil
}

// ListCollection shows a collection link's recipes without any account or author details.
func (s *Service) ListCollection(ctx context.Context, locale, token, query string, limit, offset int64) (Page[RecipeSummary], error) {
	owner, err := s.collectionOwner(ctx, token)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	limit, offset = clampPage(limit, offset)
	q := s.Store.Queries
	n := text.EscapeLike(text.Normalize(query))
	rows, err := q.ListCollectionRecipes(ctx, sqlcgen.ListCollectionRecipesParams{OwnerID: owner, Q: n, Lim: limit, Off: offset})
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	total, err := q.CountCollectionRecipes(ctx, sqlcgen.CountCollectionRecipesParams{OwnerID: owner, Q: n})
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	items, err := s.summaries(ctx, q, Viewer{Locale: locale}, rows)
	if err != nil {
		return Page[RecipeSummary]{}, err
	}
	for i := range items {
		items[i].PhotoURL, items[i].ThumbURL = photoURLs(rows[i].Photo.String, token)
	}
	return Page[RecipeSummary]{Items: items, Total: total}, nil
}

func (s *Service) GetCollectionRecipe(ctx context.Context, locale, token string, id int64) (Recipe, error) {
	owner, err := s.collectionOwner(ctx, token)
	if err != nil {
		return Recipe{}, err
	}
	q := s.Store.Queries
	r, err := q.GetRecipe(ctx, id)
	if isNoRows(err) || (err == nil && !collectionShows(r, owner)) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}
	doc, err := s.buildRecipe(ctx, q, Viewer{Locale: locale}, r)
	if err != nil {
		return Recipe{}, err
	}
	doc.PhotoURL, doc.ThumbURL = photoURLs(r.Photo.String, token)
	return anonymous(doc), nil
}

func collectionShows(r sqlcgen.Recipe, owner int64) bool {
	return r.OwnerID == owner && !r.DeletedAt.Valid && r.Visibility != "private"
}

func (s *Service) collectionCovers(ctx context.Context, token string, r sqlcgen.Recipe) (bool, error) {
	owner, err := s.collectionOwner(ctx, token)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return collectionShows(r, owner), nil
}

// reindexRecipe rebuilds a recipe's search text, for when a catalog name changes under it.
func (s *Service) reindexRecipe(ctx context.Context, q *sqlcgen.Queries, id int64) error {
	r, err := q.GetRecipe(ctx, id)
	if err != nil {
		return err
	}
	all := []string{r.Name, r.Description}
	steps, err := q.ListStepTexts(ctx, id)
	if err != nil {
		return err
	}
	for _, st := range steps {
		all = append(all, st.Name, st.Instruction)
	}
	facts, err := q.ListIngredientFacts(ctx, id)
	if err != nil {
		return err
	}
	var foods []int64
	for _, f := range facts {
		all = append(all, f.Note)
		foods = append(foods, f.FoodID)
	}
	if len(foods) > 0 {
		names, err := q.ListAllFoodSearchNames(ctx, foods)
		if err != nil {
			return err
		}
		for _, n := range names {
			all = append(all, n.SearchName)
		}
		aliases, err := q.ListFoodAliasSearchNames(ctx, foods)
		if err != nil {
			return err
		}
		all = append(all, aliases...)
	}
	keywords, err := q.ListKeywordSearchNamesForRecipe(ctx, id)
	if err != nil {
		return err
	}
	all = append(all, keywords...)
	return q.SetRecipeSearchText(ctx, sqlcgen.SetRecipeSearchTextParams{SearchText: text.Normalize(strings.Join(all, " ")), ID: id})
}

func (s *Service) reindexFood(ctx context.Context, q *sqlcgen.Queries, foodID int64) error {
	ids, err := q.ListRecipeIDsUsingFood(ctx, foodID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.reindexRecipe(ctx, q, id); err != nil {
			return err
		}
	}
	return nil
}
