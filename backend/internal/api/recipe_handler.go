package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

// ---- response types ----

type TagResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StepIngredientResponse struct {
	ID             string   `json:"id"`
	IngredientID   string   `json:"ingredient_id"`
	UnitID         *string  `json:"unit_id"`
	Amount         *float64 `json:"amount"`
	Notes          *string  `json:"notes"`
	IngredientName string   `json:"ingredient_name"`
	UnitName       string   `json:"unit_name"`
	UnitAbbr       string   `json:"unit_abbr"`
}

type RecipeStepResponse struct {
	ID          string                   `json:"id"`
	StepOrder   int32                    `json:"step_order"`
	Description string                   `json:"description"`
	DurationMin *int32                   `json:"duration_min"`
	Ingredients []StepIngredientResponse `json:"ingredients"`
}

type RecipeResponse struct {
	ID          string               `json:"id"`
	AuthorID    string               `json:"author_id"`
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	PhotoPath   *string              `json:"photo_path"`
	WorkTimeMin *int32               `json:"work_time_min"`
	Servings    *int32               `json:"servings"`
	IsPublic    bool                 `json:"is_public"`
	SourceURL   *string              `json:"source_url"`
	Tags        []TagResponse        `json:"tags"`
	Steps       []RecipeStepResponse `json:"steps,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

func recipeToResponse(r dbq.Recipe, tags []dbq.Tag, steps []RecipeStepResponse) RecipeResponse {
	authorID, _ := uuid.FromBytes(r.AuthorID.Bytes[:])
	recipeID, _ := uuid.FromBytes(r.ID.Bytes[:])

	resp := RecipeResponse{
		ID:       recipeID.String(),
		AuthorID: authorID.String(),
		Name:     r.Name,
		IsPublic: r.IsPublic,
		Tags:     []TagResponse{},
		Steps:    steps,
	}
	if r.Description.Valid {
		resp.Description = &r.Description.String
	}
	if r.PhotoPath.Valid {
		resp.PhotoPath = &r.PhotoPath.String
	}
	if r.WorkTimeMin.Valid {
		resp.WorkTimeMin = &r.WorkTimeMin.Int32
	}
	if r.Servings.Valid {
		resp.Servings = &r.Servings.Int32
	}
	if r.SourceUrl.Valid {
		resp.SourceURL = &r.SourceUrl.String
	}
	if r.CreatedAt.Valid {
		resp.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		resp.UpdatedAt = r.UpdatedAt.Time
	}
	for _, t := range tags {
		tid, _ := uuid.FromBytes(t.ID.Bytes[:])
		resp.Tags = append(resp.Tags, TagResponse{ID: tid.String(), Name: t.Name})
	}
	return resp
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func parsePgtypeUUID(s string) (pgtype.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return uuidToPgtype(id), nil
}

// resolveLang returns "de" if the Accept-Language header starts with "de", else "en".
func resolveLang(c echo.Context) string {
	al := c.Request().Header.Get("Accept-Language")
	if strings.HasPrefix(al, "de") {
		return "de"
	}
	return "en"
}

// pickLang returns deVal if lang=="de" and deVal is non-empty, else enVal.
func pickLang(lang, enVal, deVal string) string {
	if lang == "de" && deVal != "" {
		return deVal
	}
	return enVal
}

// ---- helpers ----

func intQueryParam(c echo.Context, name string, def, max int) int {
	v, err := strconv.Atoi(c.QueryParam(name))
	if err != nil || v < 0 {
		return def
	}
	if max > 0 && v > max {
		return max
	}
	return v
}

// ---- handlers ----

// GET /api/recipes
func (s *Server) ListRecipes(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	limit := intQueryParam(c, "limit", 20, 100)
	offset := intQueryParam(c, "offset", 0, 0)

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipes, err := q.ListRecipesByAuthor(ctx, dbq.ListRecipesByAuthorParams{
		AuthorID: uuidToPgtype(userID),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	out := make([]RecipeResponse, 0, len(recipes))
	for _, r := range recipes {
		tags, _ := q.ListRecipeTags(ctx, r.ID)
		out = append(out, recipeToResponse(r, tags, nil))
	}
	return c.JSON(http.StatusOK, out)
}

// POST /api/recipes
func (s *Server) CreateRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	var req struct {
		Name        string   `json:"name"`
		Description *string  `json:"description"`
		WorkTimeMin *int32   `json:"work_time_min"`
		Servings    *int32   `json:"servings"`
		IsPublic    bool     `json:"is_public"`
		SourceURL   *string  `json:"source_url"`
		Tags        []string `json:"tags"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	params := dbq.CreateRecipeParams{
		AuthorID: uuidToPgtype(userID),
		Name:     req.Name,
		IsPublic: req.IsPublic,
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.WorkTimeMin != nil {
		params.WorkTimeMin = pgtype.Int4{Int32: *req.WorkTimeMin, Valid: true}
	}
	if req.Servings != nil {
		params.Servings = pgtype.Int4{Int32: *req.Servings, Valid: true}
	}
	if req.SourceURL != nil {
		params.SourceUrl = pgtype.Text{String: *req.SourceURL, Valid: true}
	}

	recipe, err := q.CreateRecipe(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Handle tags
	for _, tagName := range req.Tags {
		tag, err := q.CreateTag(ctx, tagName)
		if err != nil {
			continue
		}
		_ = q.AddRecipeTag(ctx, dbq.AddRecipeTagParams{RecipeID: recipe.ID, TagID: tag.ID})
	}

	tags, _ := q.ListRecipeTags(ctx, recipe.ID)
	return c.JSON(http.StatusCreated, recipeToResponse(recipe, tags, nil))
}

// GET /api/recipes/:id
func (s *Server) GetRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if !recipe.IsPublic && authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	tags, _ := q.ListRecipeTags(ctx, recipe.ID)
	rawSteps, _ := q.ListRecipeSteps(ctx, recipe.ID)
	steps := make([]RecipeStepResponse, 0, len(rawSteps))
	for _, st := range rawSteps {
		stepID, _ := uuid.FromBytes(st.ID.Bytes[:])
		ings, _ := q.ListStepIngredients(ctx, st.ID)
		ingResps := buildIngredientResponses(resolveLang(c), ings)
		sr := RecipeStepResponse{
			ID:          stepID.String(),
			StepOrder:   st.StepOrder,
			Description: st.Description,
			Ingredients: ingResps,
		}
		if st.DurationMin.Valid {
			sr.DurationMin = &st.DurationMin.Int32
		}
		steps = append(steps, sr)
	}

	return c.JSON(http.StatusOK, recipeToResponse(recipe, tags, steps))
}

// PUT /api/recipes/:id
func (s *Server) UpdateRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		WorkTimeMin *int32   `json:"work_time_min"`
		Servings    *int32   `json:"servings"`
		IsPublic    *bool    `json:"is_public"`
		SourceURL   *string  `json:"source_url"`
		Tags        []string `json:"tags"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	params := dbq.UpdateRecipeParams{ID: recipeUUID}
	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.WorkTimeMin != nil {
		params.WorkTimeMin = pgtype.Int4{Int32: *req.WorkTimeMin, Valid: true}
	}
	if req.Servings != nil {
		params.Servings = pgtype.Int4{Int32: *req.Servings, Valid: true}
	}
	if req.IsPublic != nil {
		params.IsPublic = pgtype.Bool{Bool: *req.IsPublic, Valid: true}
	}
	if req.SourceURL != nil {
		params.SourceUrl = pgtype.Text{String: *req.SourceURL, Valid: true}
	}

	updated, err := q.UpdateRecipe(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if req.Tags != nil {
		_ = q.RemoveAllRecipeTags(ctx, recipeUUID)
		for _, tagName := range req.Tags {
			tag, err := q.CreateTag(ctx, tagName)
			if err != nil {
				continue
			}
			_ = q.AddRecipeTag(ctx, dbq.AddRecipeTagParams{RecipeID: recipeUUID, TagID: tag.ID})
		}
	}

	tags, _ := q.ListRecipeTags(ctx, updated.ID)
	return c.JSON(http.StatusOK, recipeToResponse(updated, tags, nil))
}

// DELETE /api/recipes/:id
func (s *Server) DeleteRecipe(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	if recipe.PhotoPath.Valid && recipe.PhotoPath.String != "" {
		s.storage.DeletePhoto(recipe.PhotoPath.String)
	}

	if err := q.DeleteRecipe(ctx, recipeUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /api/recipes/:id/photo
func (s *Server) UploadRecipePhoto(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	file, err := c.FormFile("photo")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "photo field required"})
	}

	if recipe.PhotoPath.Valid && recipe.PhotoPath.String != "" {
		s.storage.DeletePhoto(recipe.PhotoPath.String)
	}

	filename, err := s.storage.SavePhoto(file)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	updated, err := q.UpdateRecipe(ctx, dbq.UpdateRecipeParams{
		ID:        recipeUUID,
		PhotoPath: pgtype.Text{String: filename, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	tags, _ := q.ListRecipeTags(ctx, updated.ID)
	return c.JSON(http.StatusOK, recipeToResponse(updated, tags, nil))
}

// GET /api/recipes/:id/steps
func (s *Server) ListRecipeSteps(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if !recipe.IsPublic && authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	rawSteps, _ := q.ListRecipeSteps(ctx, recipeUUID)
	steps := make([]RecipeStepResponse, 0, len(rawSteps))
	for _, st := range rawSteps {
		stepID, _ := uuid.FromBytes(st.ID.Bytes[:])
		ings, _ := q.ListStepIngredients(ctx, st.ID)
		ingResps := buildIngredientResponses(resolveLang(c), ings)
		sr := RecipeStepResponse{
			ID:          stepID.String(),
			StepOrder:   st.StepOrder,
			Description: st.Description,
			Ingredients: ingResps,
		}
		if st.DurationMin.Valid {
			sr.DurationMin = &st.DurationMin.Int32
		}
		steps = append(steps, sr)
	}
	return c.JSON(http.StatusOK, steps)
}

func buildIngredientResponses(lang string, ings []dbq.StepIngredient) []StepIngredientResponse {
	out := make([]StepIngredientResponse, 0, len(ings))
	for _, ing := range ings {
		iid, _ := uuid.FromBytes(ing.ID.Bytes[:])
		ingID, _ := uuid.FromBytes(ing.IngredientID.Bytes[:])
		ir := StepIngredientResponse{
			ID:             iid.String(),
			IngredientID:   ingID.String(),
			IngredientName: pickLang(lang, ing.IngredientName, ing.IngredientNameDe),
			UnitName:       pickLang(lang, ing.UnitName, ing.UnitNameDe),
			UnitAbbr:       pickLang(lang, ing.UnitAbbr, ing.UnitAbbrDe),
		}
		if ing.UnitID.Valid {
			uid, _ := uuid.FromBytes(ing.UnitID.Bytes[:])
			s := uid.String()
			ir.UnitID = &s
		}
		if ing.Amount.Valid {
			f, _ := ing.Amount.Float64Value()
			if f.Valid {
				ir.Amount = &f.Float64
			}
		}
		if ing.Notes.Valid {
			ir.Notes = &ing.Notes.String
		}
		out = append(out, ir)
	}
	return out
}

// POST /api/recipes/:id/steps
func (s *Server) CreateRecipeStep(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	var req struct {
		Description string  `json:"description"`
		DurationMin *int32  `json:"duration_min"`
		StepOrder   *int32  `json:"step_order"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var stepOrder int32
	if req.StepOrder != nil {
		stepOrder = *req.StepOrder
	} else {
		existing, _ := q.ListRecipeSteps(ctx, recipeUUID)
		stepOrder = int32(len(existing) + 1)
	}

	params := dbq.CreateRecipeStepParams{
		RecipeID:    recipeUUID,
		StepOrder:   stepOrder,
		Description: req.Description,
	}
	if req.DurationMin != nil {
		params.DurationMin = pgtype.Int4{Int32: *req.DurationMin, Valid: true}
	}

	step, err := q.CreateRecipeStep(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	stepID, _ := uuid.FromBytes(step.ID.Bytes[:])
	resp := RecipeStepResponse{
		ID:          stepID.String(),
		StepOrder:   step.StepOrder,
		Description: step.Description,
		Ingredients: []StepIngredientResponse{},
	}
	if step.DurationMin.Valid {
		resp.DurationMin = &step.DurationMin.Int32
	}
	return c.JSON(http.StatusCreated, resp)
}

// PUT /api/recipes/:id/steps/:stepId
func (s *Server) UpdateRecipeStep(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	stepUUID, err := parsePgtypeUUID(c.Param("stepId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid stepId"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	step, err := q.GetRecipeStepByID(ctx, stepUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "step not found"})
	}

	var req struct {
		Description string  `json:"description"`
		DurationMin *int32  `json:"duration_min"`
		StepOrder   *int32  `json:"step_order"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	params := dbq.UpdateRecipeStepParams{
		ID:          stepUUID,
		Description: req.Description,
		StepOrder:   step.StepOrder,
	}
	if req.StepOrder != nil {
		params.StepOrder = *req.StepOrder
	}
	if req.DurationMin != nil {
		params.DurationMin = pgtype.Int4{Int32: *req.DurationMin, Valid: true}
	}

	updated, err := q.UpdateRecipeStep(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	stepID, _ := uuid.FromBytes(updated.ID.Bytes[:])
	resp := RecipeStepResponse{
		ID:          stepID.String(),
		StepOrder:   updated.StepOrder,
		Description: updated.Description,
		Ingredients: []StepIngredientResponse{},
	}
	if updated.DurationMin.Valid {
		resp.DurationMin = &updated.DurationMin.Int32
	}
	return c.JSON(http.StatusOK, resp)
}

// DELETE /api/recipes/:id/steps/:stepId
func (s *Server) DeleteRecipeStep(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	stepUUID, err := parsePgtypeUUID(c.Param("stepId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid stepId"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	if err := q.DeleteRecipeStep(ctx, stepUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /api/recipes/:id/steps/:stepId/ingredients
func (s *Server) AddStepIngredient(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	stepUUID, err := parsePgtypeUUID(c.Param("stepId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid stepId"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	var req struct {
		IngredientID string   `json:"ingredient_id"`
		UnitID       *string  `json:"unit_id"`
		Amount       *float64 `json:"amount"`
		Notes        *string  `json:"notes"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	ingUUID, err := parsePgtypeUUID(req.IngredientID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ingredient_id"})
	}

	params := dbq.CreateStepIngredientParams{
		StepID:       stepUUID,
		IngredientID: ingUUID,
	}
	if req.UnitID != nil {
		uid, err := parsePgtypeUUID(*req.UnitID)
		if err == nil {
			params.UnitID = uid
		}
	}
	if req.Amount != nil {
		params.Amount = pgtype.Numeric{}
		if err := params.Amount.Scan(*req.Amount); err != nil {
			_ = err // ignore, will be null
		}
	}
	if req.Notes != nil {
		params.Notes = pgtype.Text{String: *req.Notes, Valid: true}
	}

	si, err := q.CreateStepIngredient(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if params.UnitID.Valid {
		_ = q.IncrementIngredientUnitStat(ctx, dbq.IncrementIngredientUnitStatParams{
			IngredientID: ingUUID,
			UnitID:       params.UnitID,
		})
	}

	iid, _ := uuid.FromBytes(si.ID.Bytes[:])
	ingID2, _ := uuid.FromBytes(si.IngredientID.Bytes[:])
	ir := StepIngredientResponse{
		ID:           iid.String(),
		IngredientID: ingID2.String(),
	}
	if si.UnitID.Valid {
		uid, _ := uuid.FromBytes(si.UnitID.Bytes[:])
		s := uid.String()
		ir.UnitID = &s
	}
	if si.Amount.Valid {
		f, _ := si.Amount.Float64Value()
		if f.Valid {
			ir.Amount = &f.Float64
		}
	}
	if si.Notes.Valid {
		ir.Notes = &si.Notes.String
	}
	return c.JSON(http.StatusCreated, ir)
}

// DELETE /api/recipes/:id/steps/:stepId/ingredients/:ingId
func (s *Server) RemoveStepIngredient(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	ingUUID, err := parsePgtypeUUID(c.Param("ingId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ingId"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	if err := q.DeleteStepIngredient(ctx, ingUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /api/recipes/:id/share
func (s *Server) CreateShareLink(c echo.Context) error {
	userID, _ := c.Get("user_id").(uuid.UUID)
	recipeUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	recipe, err := q.GetRecipeByID(ctx, recipeUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	authorID, _ := uuid.FromBytes(recipe.AuthorID.Bytes[:])
	if authorID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	var req struct {
		ExpiresInDays *int `json:"expires_in_days"`
	}
	_ = c.Bind(&req)

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	params := dbq.CreateShareLinkParams{
		Token:    token,
		UserID:   uuidToPgtype(userID),
		RecipeID: recipeUUID,
	}
	if req.ExpiresInDays != nil {
		t := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		params.ExpiresAt = pgtype.Timestamptz{Time: t, Valid: true}
	}

	link, err := q.CreateShareLink(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"token": link.Token,
		"url":   s.cfg.PublicURL + "/share/" + link.Token,
	})
}
