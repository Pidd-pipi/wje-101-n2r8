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
	recipes.GET("/:id/versions/:version", h.GetVersion)
	recipes.POST("", middleware.AuthRequired(cfg), limiter.Limit(), h.Create)
	// Only the recipe owner may publish revisions or select the active one;
	// ownership is enforced in the service layer.
	recipes.POST("/:id/versions", middleware.AuthRequired(cfg), limiter.Limit(), h.PublishVersion)
	recipes.PUT("/:id/active-version", middleware.AuthRequired(cfg), h.SetActiveVersion)
}
