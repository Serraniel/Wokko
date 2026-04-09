package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

type UnitResponse struct {
	ID             string  `json:"id"`
	NameEn         string  `json:"name_en"`
	NameDe         *string `json:"name_de"`
	AbbreviationEn *string `json:"abbreviation_en"`
	AbbreviationDe *string `json:"abbreviation_de"`
	PluralEn       *string `json:"plural_en"`
	PluralDe       *string `json:"plural_de"`
}

func unitToResponse(u dbq.Unit) UnitResponse {
	id, _ := uuid.FromBytes(u.ID.Bytes[:])
	r := UnitResponse{
		ID:     id.String(),
		NameEn: u.NameEn,
	}
	if u.NameDe.Valid {
		r.NameDe = &u.NameDe.String
	}
	if u.AbbreviationEn.Valid {
		r.AbbreviationEn = &u.AbbreviationEn.String
	}
	if u.AbbreviationDe.Valid {
		r.AbbreviationDe = &u.AbbreviationDe.String
	}
	if u.PluralEn.Valid {
		r.PluralEn = &u.PluralEn.String
	}
	if u.PluralDe.Valid {
		r.PluralDe = &u.PluralDe.String
	}
	return r
}

// GET /api/units
func (s *Server) ListUnits(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	search := c.QueryParam("search")
	limit := intQueryParam(c, "limit", 20, 200)
	offset := intQueryParam(c, "offset", 0, 0)

	var items []dbq.Unit
	if search != "" {
		var err error
		items, err = q.SearchUnits(ctx, dbq.SearchUnitsParams{
			Column1: pgtype.Text{String: search, Valid: true},
			Limit:   int32(limit),
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else {
		var err error
		items, err = q.ListUnits(ctx, dbq.ListUnitsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	out := make([]UnitResponse, 0, len(items))
	for _, u := range items {
		out = append(out, unitToResponse(u))
	}
	return c.JSON(http.StatusOK, out)
}

// GET /api/units/:id
func (s *Server) GetUnit(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	id, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	unit, err := q.GetUnitByID(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return c.JSON(http.StatusOK, unitToResponse(unit))
}

// POST /api/units
func (s *Server) CreateUnit(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	var req struct {
		NameEn         string  `json:"name_en"`
		NameDe         *string `json:"name_de"`
		AbbreviationEn *string `json:"abbreviation_en"`
		AbbreviationDe *string `json:"abbreviation_de"`
		PluralEn       *string `json:"plural_en"`
		PluralDe       *string `json:"plural_de"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.NameEn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name_en is required"})
	}

	params := dbq.CreateUnitParams{NameEn: req.NameEn}
	if req.NameDe != nil {
		params.NameDe = pgtype.Text{String: *req.NameDe, Valid: true}
	}
	if req.AbbreviationEn != nil {
		params.AbbreviationEn = pgtype.Text{String: *req.AbbreviationEn, Valid: true}
	}
	if req.AbbreviationDe != nil {
		params.AbbreviationDe = pgtype.Text{String: *req.AbbreviationDe, Valid: true}
	}
	if req.PluralEn != nil {
		params.PluralEn = pgtype.Text{String: *req.PluralEn, Valid: true}
	}
	if req.PluralDe != nil {
		params.PluralDe = pgtype.Text{String: *req.PluralDe, Valid: true}
	}

	unit, err := q.CreateUnit(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, unitToResponse(unit))
}

// PUT /api/units/:id
func (s *Server) UpdateUnit(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	id, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req struct {
		NameEn         *string `json:"name_en"`
		NameDe         *string `json:"name_de"`
		AbbreviationEn *string `json:"abbreviation_en"`
		AbbreviationDe *string `json:"abbreviation_de"`
		PluralEn       *string `json:"plural_en"`
		PluralDe       *string `json:"plural_de"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	params := dbq.UpdateUnitParams{ID: id}
	if req.NameEn != nil {
		params.NameEn = pgtype.Text{String: *req.NameEn, Valid: true}
	}
	if req.NameDe != nil {
		params.NameDe = pgtype.Text{String: *req.NameDe, Valid: true}
	}
	if req.AbbreviationEn != nil {
		params.AbbreviationEn = pgtype.Text{String: *req.AbbreviationEn, Valid: true}
	}
	if req.AbbreviationDe != nil {
		params.AbbreviationDe = pgtype.Text{String: *req.AbbreviationDe, Valid: true}
	}
	if req.PluralEn != nil {
		params.PluralEn = pgtype.Text{String: *req.PluralEn, Valid: true}
	}
	if req.PluralDe != nil {
		params.PluralDe = pgtype.Text{String: *req.PluralDe, Valid: true}
	}

	unit, err := q.UpdateUnit(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, unitToResponse(unit))
}

// POST /api/units/merge
func (s *Server) MergeUnits(c echo.Context) error {
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

	if err := q.MergeUnitReferences(ctx, dbq.MergeUnitReferencesParams{
		UnitID:   sourceUUID,
		UnitID_2: targetUUID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := q.DeleteUnit(ctx, sourceUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "merged"})
}
