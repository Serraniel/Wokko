package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

type IngredientResponse struct {
	ID              string  `json:"id"`
	NameEn          string  `json:"name_en"`
	NameDe          *string `json:"name_de"`
	PluralEn        *string `json:"plural_en"`
	PluralDe        *string `json:"plural_de"`
	DefaultUnitID   *string `json:"default_unit_id"`
	SuggestedUnitID *string `json:"suggested_unit_id"`
}

func ingredientToResponse(i dbq.Ingredient, suggestedUnitID *string) IngredientResponse {
	id, _ := uuid.FromBytes(i.ID.Bytes[:])
	r := IngredientResponse{
		ID:              id.String(),
		NameEn:          i.NameEn,
		SuggestedUnitID: suggestedUnitID,
	}
	if i.NameDe.Valid {
		r.NameDe = &i.NameDe.String
	}
	if i.PluralEn.Valid {
		r.PluralEn = &i.PluralEn.String
	}
	if i.PluralDe.Valid {
		r.PluralDe = &i.PluralDe.String
	}
	if i.DefaultUnitID.Valid {
		uid, _ := uuid.FromBytes(i.DefaultUnitID.Bytes[:])
		s := uid.String()
		r.DefaultUnitID = &s
	}
	return r
}

// GET /api/ingredients
func (s *Server) ListIngredients(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	search := c.QueryParam("search")
	limit := intQueryParam(c, "limit", 20, 200)
	offset := intQueryParam(c, "offset", 0, 0)

	var items []dbq.Ingredient
	if search != "" {
		var err error
		items, err = q.SearchIngredients(ctx, dbq.SearchIngredientsParams{
			Column1: pgtype.Text{String: search, Valid: true},
			Limit:   int32(limit),
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else {
		var err error
		items, err = q.ListIngredients(ctx, dbq.ListIngredientsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	out := make([]IngredientResponse, 0, len(items))
	for _, i := range items {
		var suggested *string
		suid, err := q.GetSuggestedUnitForIngredient(ctx, i.ID)
		if err == nil && suid.Valid {
			uid, _ := uuid.FromBytes(suid.Bytes[:])
			s := uid.String()
			suggested = &s
		}
		out = append(out, ingredientToResponse(i, suggested))
	}
	return c.JSON(http.StatusOK, out)
}

// GET /api/ingredients/:id
func (s *Server) GetIngredient(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	id, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ing, err := q.GetIngredientByID(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	var suggested *string
	suid, err := q.GetSuggestedUnitForIngredient(ctx, id)
	if err == nil && suid.Valid {
		uid, _ := uuid.FromBytes(suid.Bytes[:])
		s := uid.String()
		suggested = &s
	}

	return c.JSON(http.StatusOK, ingredientToResponse(ing, suggested))
}

// POST /api/ingredients
func (s *Server) CreateIngredient(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	var req struct {
		NameEn        string  `json:"name_en"`
		NameDe        *string `json:"name_de"`
		PluralEn      *string `json:"plural_en"`
		PluralDe      *string `json:"plural_de"`
		DefaultUnitID *string `json:"default_unit_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.NameEn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name_en is required"})
	}

	params := dbq.CreateIngredientParams{NameEn: req.NameEn}
	if req.NameDe != nil {
		params.NameDe = pgtype.Text{String: *req.NameDe, Valid: true}
	}
	if req.PluralEn != nil {
		params.PluralEn = pgtype.Text{String: *req.PluralEn, Valid: true}
	}
	if req.PluralDe != nil {
		params.PluralDe = pgtype.Text{String: *req.PluralDe, Valid: true}
	}
	if req.DefaultUnitID != nil {
		uid, err := parsePgtypeUUID(*req.DefaultUnitID)
		if err == nil {
			params.DefaultUnitID = uid
		}
	}

	ing, err := q.CreateIngredient(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, ingredientToResponse(ing, nil))
}

// PUT /api/ingredients/:id
func (s *Server) UpdateIngredient(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	id, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		NameEn        *string `json:"name_en"`
		NameDe        *string `json:"name_de"`
		PluralEn      *string `json:"plural_en"`
		PluralDe      *string `json:"plural_de"`
		DefaultUnitID *string `json:"default_unit_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	params := dbq.UpdateIngredientParams{ID: id}
	if req.NameEn != nil {
		params.NameEn = pgtype.Text{String: *req.NameEn, Valid: true}
	}
	if req.NameDe != nil {
		params.NameDe = pgtype.Text{String: *req.NameDe, Valid: true}
	}
	if req.PluralEn != nil {
		params.PluralEn = pgtype.Text{String: *req.PluralEn, Valid: true}
	}
	if req.PluralDe != nil {
		params.PluralDe = pgtype.Text{String: *req.PluralDe, Valid: true}
	}
	if req.DefaultUnitID != nil {
		uid, err := parsePgtypeUUID(*req.DefaultUnitID)
		if err == nil {
			params.DefaultUnitID = uid
		}
	}

	ing, err := q.UpdateIngredient(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, ingredientToResponse(ing, nil))
}

// POST /api/ingredients/merge
func (s *Server) MergeIngredients(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	var req struct {
		SourceID string `json:"source_id"`
		TargetID string `json:"target_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	sourceUUID, err := parsePgtypeUUID(req.SourceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid source_id"})
	}
	targetUUID, err := parsePgtypeUUID(req.TargetID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid target_id"})
	}

	if err := q.MergeIngredientReferences(ctx, dbq.MergeIngredientReferencesParams{
		IngredientID:   sourceUUID,
		IngredientID_2: targetUUID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := q.DeleteIngredient(ctx, sourceUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "merged"})
}
