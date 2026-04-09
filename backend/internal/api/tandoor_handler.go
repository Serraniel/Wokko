package api

import (
	"hash/crc32"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

func uuidToInt(id uuid.UUID) int {
	return int(crc32.ChecksumIEEE(id[:]))
}

func pgtypeUUIDToInt(id pgtype.UUID) int {
	if !id.Valid {
		return 0
	}
	uid, _ := uuid.FromBytes(id.Bytes[:])
	return uuidToInt(uid)
}

// Tandoor response shapes (minimal subset for kitshn)
type TandoorRecipe struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Image       *string          `json:"image"`
	WorkingTime int              `json:"working_time"`
	WaitingTime int              `json:"waiting_time"`
	Servings    int              `json:"servings"`
	Keywords    []TandoorKeyword `json:"keywords"`
	Steps       []TandoorStep    `json:"steps"`
}

type TandoorKeyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TandoorStep struct {
	ID          int                 `json:"id"`
	Name        string              `json:"name"`
	Instruction string              `json:"instruction"`
	Ingredients []TandoorIngredient `json:"ingredients"`
	Time        int                 `json:"time"`
}

type TandoorIngredient struct {
	ID     int          `json:"id"`
	Amount float64      `json:"amount"`
	Unit   *TandoorUnit `json:"unit"`
	Food   TandoorFood  `json:"food"`
	Note   string       `json:"note"`
}

type TandoorUnit struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TandoorFood struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TandoorListResponse struct {
	Count    int         `json:"count"`
	Next     *string     `json:"next"`
	Previous *string     `json:"previous"`
	Results  interface{} `json:"results"`
}

func (s *Server) buildTandoorRecipe(c echo.Context, q *dbq.Queries, r dbq.Recipe) (TandoorRecipe, error) {
	ctx := c.Request().Context()
	tags, _ := q.ListRecipeTags(ctx, r.ID)
	rawSteps, _ := q.ListRecipeSteps(ctx, r.ID)

	keywords := make([]TandoorKeyword, 0, len(tags))
	for _, t := range tags {
		tid, _ := uuid.FromBytes(t.ID.Bytes[:])
		keywords = append(keywords, TandoorKeyword{
			ID:   uuidToInt(tid),
			Name: t.Name,
		})
	}

	steps := make([]TandoorStep, 0, len(rawSteps))
	for _, st := range rawSteps {
		ings, _ := q.ListStepIngredients(ctx, st.ID)
		tandoorIngs := make([]TandoorIngredient, 0, len(ings))
		for _, ing := range ings {
			ingDB, err := q.GetIngredientByID(ctx, ing.IngredientID)
			if err != nil {
				continue
			}
			ti := TandoorIngredient{
				ID:   pgtypeUUIDToInt(ing.ID),
				Food: TandoorFood{ID: pgtypeUUIDToInt(ing.IngredientID), Name: ingDB.NameEn},
			}
			if ing.UnitID.Valid {
				unitDB, err := q.GetUnitByID(ctx, ing.UnitID)
				if err == nil {
					unitInt := pgtypeUUIDToInt(ing.UnitID)
					ti.Unit = &TandoorUnit{ID: unitInt, Name: unitDB.NameEn}
				}
			}
			if ing.Amount.Valid {
				f, _ := ing.Amount.Float64Value()
				if f.Valid {
					ti.Amount = f.Float64
				}
			}
			if ing.Notes.Valid {
				ti.Note = ing.Notes.String
			}
			tandoorIngs = append(tandoorIngs, ti)
		}

		dur := 0
		if st.DurationMin.Valid {
			dur = int(st.DurationMin.Int32)
		}
		steps = append(steps, TandoorStep{
			ID:          pgtypeUUIDToInt(st.ID),
			Name:        "",
			Instruction: st.Description,
			Ingredients: tandoorIngs,
			Time:        dur,
		})
	}

	workTime := 0
	if r.WorkTimeMin.Valid {
		workTime = int(r.WorkTimeMin.Int32)
	}
	servings := 1
	if r.Servings.Valid {
		servings = int(r.Servings.Int32)
	}
	desc := ""
	if r.Description.Valid {
		desc = r.Description.String
	}
	var image *string
	if r.PhotoPath.Valid && r.PhotoPath.String != "" {
		url := s.cfg.PublicURL + "/api/media/" + r.PhotoPath.String
		image = &url
	}

	return TandoorRecipe{
		ID:          pgtypeUUIDToInt(r.ID),
		Name:        r.Name,
		Description: desc,
		Image:       image,
		WorkingTime: workTime,
		WaitingTime: 0,
		Servings:    servings,
		Keywords:    keywords,
		Steps:       steps,
	}, nil
}

// GET /api/recipe/
func (s *Server) TandoorListRecipes(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	limit := intQueryParam(c, "page_size", 20, 100)
	offset := intQueryParam(c, "page", 0, 0)
	if offset > 0 {
		offset = (offset - 1) * limit
	}

	recipes, err := q.ListRecipesByAuthor(ctx, dbq.ListRecipesByAuthorParams{
		AuthorID: uuidToPgtype(userID),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	count, _ := q.CountRecipesByAuthor(ctx, uuidToPgtype(userID))

	results := make([]TandoorRecipe, 0, len(recipes))
	for _, r := range recipes {
		tr, err := s.buildTandoorRecipe(c, q, r)
		if err == nil {
			results = append(results, tr)
		}
	}

	return c.JSON(http.StatusOK, TandoorListResponse{
		Count:   int(count),
		Results: results,
	})
}

// GET /api/recipe/:id/
func (s *Server) TandoorGetRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if !recipe.IsPublic && authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	tr, err := s.buildTandoorRecipe(c, q, recipe)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, tr)
}

// POST /api/recipe/
func (s *Server) TandoorCreateRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		WorkingTime int    `json:"working_time"`
		Servings    int    `json:"servings"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	params := dbq.CreateRecipeParams{
		AuthorID: uuidToPgtype(userID),
		Name:     req.Name,
	}
	if req.Description != "" {
		params.Description = pgtype.Text{String: req.Description, Valid: true}
	}
	if req.WorkingTime > 0 {
		params.WorkTimeMin = pgtype.Int4{Int32: int32(req.WorkingTime), Valid: true}
	}
	if req.Servings > 0 {
		params.Servings = pgtype.Int4{Int32: int32(req.Servings), Valid: true}
	}

	recipe, err := q.CreateRecipe(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	tr, _ := s.buildTandoorRecipe(c, q, recipe)
	return c.JSON(http.StatusCreated, tr)
}

// GET /api/food/
func (s *Server) TandoorListFood(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	limit := intQueryParam(c, "page_size", 20, 200)
	offset := intQueryParam(c, "page", 0, 0)
	if offset > 0 {
		offset = (offset - 1) * limit
	}

	items, err := q.ListIngredients(ctx, dbq.ListIngredientsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	count, _ := q.CountIngredients(ctx)
	results := make([]TandoorFood, 0, len(items))
	for _, i := range items {
		results = append(results, TandoorFood{
			ID:   pgtypeUUIDToInt(i.ID),
			Name: i.NameEn,
		})
	}
	return c.JSON(http.StatusOK, TandoorListResponse{Count: int(count), Results: results})
}

// GET /api/unit/
func (s *Server) TandoorListUnits(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	limit := intQueryParam(c, "page_size", 20, 200)
	offset := intQueryParam(c, "page", 0, 0)
	if offset > 0 {
		offset = (offset - 1) * limit
	}

	items, err := q.ListUnits(ctx, dbq.ListUnitsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	count, _ := q.CountUnits(ctx)
	results := make([]TandoorUnit, 0, len(items))
	for _, u := range items {
		results = append(results, TandoorUnit{
			ID:   pgtypeUUIDToInt(u.ID),
			Name: u.NameEn,
		})
	}
	return c.JSON(http.StatusOK, TandoorListResponse{Count: int(count), Results: results})
}

// GET /api/keyword/
func (s *Server) TandoorListKeywords(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	tags, err := q.ListTags(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	results := make([]TandoorKeyword, 0, len(tags))
	for _, t := range tags {
		tid, _ := uuid.FromBytes(t.ID.Bytes[:])
		results = append(results, TandoorKeyword{
			ID:   uuidToInt(tid),
			Name: t.Name,
		})
	}
	return c.JSON(http.StatusOK, TandoorListResponse{Count: len(tags), Results: results})
}
