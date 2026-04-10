package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

// GET /api/public/recipes
func (s *Server) ListPublicRecipes(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	limit := intQueryParam(c, "limit", 20, 100)
	offset := intQueryParam(c, "offset", 0, 0)

	recipes, err := q.ListPublicRecipes(ctx, dbq.ListPublicRecipesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
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

// GET /api/public/recipes/:id
func (s *Server) GetPublicRecipe(c echo.Context) error {
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
	if !recipe.IsPublic {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not public"})
	}

	tags, _ := q.ListRecipeTags(ctx, recipe.ID)
	return c.JSON(http.StatusOK, recipeToResponse(recipe, tags, nil))
}

// GET /api/public/share/:token
func (s *Server) GetSharedRecipes(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	token := c.Param("token")
	link, err := q.GetShareLinkByToken(ctx, token)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "share link not found"})
	}

	if link.ExpiresAt.Valid && link.ExpiresAt.Time.Before(time.Now()) {
		return c.JSON(http.StatusGone, map[string]string{"error": "share link expired"})
	}

	recipe, err := q.GetRecipeByID(ctx, link.RecipeID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "recipe not found"})
	}

	tags, _ := q.ListRecipeTags(ctx, recipe.ID)
	rawSteps, _ := q.ListRecipeSteps(ctx, recipe.ID)
	steps := make([]RecipeStepResponse, 0, len(rawSteps))
	for _, st := range rawSteps {
		stepID, _ := uuid.FromBytes(st.ID.Bytes[:])
		ings, _ := q.ListStepIngredients(ctx, st.ID)
		ingResps := buildIngredientResponses(ctx, q, ings)
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
