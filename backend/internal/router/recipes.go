package router

import (
	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/config"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/handler"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
)

func registerRecipeRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.RecipeHandler, limiter *middleware.RateLimiter) {
	recipes := v1.Group("/recipes")
	recipes.GET("", h.List)
	recipes.GET("/:id", h.Get)
	recipes.POST("", middleware.AuthRequired(cfg), limiter.Limit(), h.Create)

	// Recipe versions: only the recipe author may publish a new version,
	// choose the current version, or deprecate/restore a version (ownership
	// is enforced in the service).
	recipes.POST("/:id/versions", middleware.AuthRequired(cfg), limiter.Limit(), h.AddVersion)
	recipes.PUT("/:id/current-version", middleware.AuthRequired(cfg), h.SetCurrentVersion)
	recipes.PUT("/:id/versions/:versionNumber/status", middleware.AuthRequired(cfg), h.SetVersionStatus)
}
