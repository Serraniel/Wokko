package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/config"
)

func RegisterRoutes(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	s := NewServer(pool, cfg)

	// Static uploads
	e.Static("/api/media", cfg.UploadPath)

	// Health
	e.GET("/api/health", s.HealthCheck)

	// Auth (public)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/setup", s.SetupAdmin)
	authGroup.POST("/register", s.Register)
	authGroup.POST("/login", s.Login)
	authGroup.POST("/refresh", s.RefreshToken)
	authGroup.POST("/logout", s.Logout)
	authGroup.GET("/me", s.GetCurrentUser, s.AuthMiddleware)
	authGroup.GET("/passkey/register/begin", s.PasskeyRegisterBegin, s.AuthMiddleware)
	authGroup.POST("/passkey/register/finish", s.PasskeyRegisterFinish, s.AuthMiddleware)
	authGroup.GET("/passkey/login/begin", s.PasskeyLoginBegin)
	authGroup.POST("/passkey/login/finish", s.PasskeyLoginFinish)

	// Protected API
	apiGroup := e.Group("/api", s.AuthMiddleware)

	// Recipes
	apiGroup.GET("/recipes", s.ListRecipes)
	apiGroup.POST("/recipes", s.CreateRecipe)
	apiGroup.GET("/recipes/:id", s.GetRecipe)
	apiGroup.PUT("/recipes/:id", s.UpdateRecipe)
	apiGroup.DELETE("/recipes/:id", s.DeleteRecipe)
	apiGroup.POST("/recipes/:id/photo", s.UploadRecipePhoto)
	apiGroup.GET("/recipes/:id/steps", s.ListRecipeSteps)
	apiGroup.POST("/recipes/:id/steps", s.CreateRecipeStep)
	apiGroup.PUT("/recipes/:id/steps/:stepId", s.UpdateRecipeStep)
	apiGroup.DELETE("/recipes/:id/steps/:stepId", s.DeleteRecipeStep)
	apiGroup.POST("/recipes/:id/steps/:stepId/ingredients", s.AddStepIngredient)
	apiGroup.DELETE("/recipes/:id/steps/:stepId/ingredients/:ingId", s.RemoveStepIngredient)
	apiGroup.POST("/recipes/:id/share", s.CreateShareLink)

	// Ingredients
	apiGroup.GET("/ingredients", s.ListIngredients)
	apiGroup.POST("/ingredients", s.CreateIngredient)
	apiGroup.POST("/ingredients/merge", s.MergeIngredients)
	apiGroup.GET("/ingredients/:id", s.GetIngredient)
	apiGroup.PUT("/ingredients/:id", s.UpdateIngredient)

	// Units
	apiGroup.GET("/units", s.ListUnits)
	apiGroup.POST("/units", s.CreateUnit)
	apiGroup.POST("/units/merge", s.MergeUnits)
	apiGroup.GET("/units/:id", s.GetUnit)
	apiGroup.PUT("/units/:id", s.UpdateUnit)

	// Admin
	admin := e.Group("/api/admin", s.AuthMiddleware, s.AdminMiddleware)
	admin.GET("/users", s.ListUsers)
	admin.DELETE("/users/:id", s.DeleteUserAdmin)
	admin.POST("/invites", s.CreateInviteLinkAdmin)
	admin.GET("/invites", s.ListInviteLinksAdmin)
	admin.DELETE("/invites/:id", s.DeleteInviteLinkAdmin)
	admin.GET("/settings", s.GetSettings)
	admin.PUT("/settings", s.UpdateSettings)

	// Public (no auth)
	pub := e.Group("/api/public")
	pub.GET("/recipes", s.ListPublicRecipes)
	pub.GET("/recipes/:id", s.GetPublicRecipe)
	pub.GET("/share/:token", s.GetSharedRecipes)

	// Tandoor compat (for kitshn)
	tandoor := e.Group("/api", s.TandoorAuthMiddleware)
	tandoor.GET("/recipe/", s.TandoorListRecipes)
	tandoor.GET("/recipe/:id/", s.TandoorGetRecipe)
	tandoor.POST("/recipe/", s.TandoorCreateRecipe)
	tandoor.GET("/food/", s.TandoorListFood)
	tandoor.GET("/unit/", s.TandoorListUnits)
	tandoor.GET("/keyword/", s.TandoorListKeywords)
}
