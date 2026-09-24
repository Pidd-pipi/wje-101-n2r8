package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/middleware"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/service"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// RecipeHandler exposes brew recipe endpoints.
type RecipeHandler struct {
	svc    *service.RecipeService
	logger *slog.Logger
}

// NewRecipeHandler creates a RecipeHandler.
func NewRecipeHandler(svc *service.RecipeService, logger *slog.Logger) *RecipeHandler {
	return &RecipeHandler{svc: svc, logger: logger}
}

// List handles GET /recipes.
func (h *RecipeHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	device := c.Query("device")
	keyword := c.Query("keyword")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	items, total, err := h.svc.List(device, keyword, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.PageData{List: items, Total: total, Page: page, Size: pageSize}))
}

// Get handles GET /recipes/:id — recipe with its active version and history.
func (h *RecipeHandler) Get(c *gin.Context) {
	id, ok := parseRecipeID(c)
	if !ok {
		return
	}
	rec, active, versions, err := h.svc.GetDetail(id)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"recipe": rec, "active_version": active, "versions": versions}))
}

// GetVersion handles GET /recipes/:id/versions/:version.
func (h *RecipeHandler) GetVersion(c *gin.Context) {
	id, ok := parseRecipeID(c)
	if !ok {
		return
	}
	versionNumber, err := strconv.Atoi(c.Param("version"))
	if err != nil || versionNumber < 1 {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid version number"))
		return
	}
	ver, err := h.svc.GetVersion(id, versionNumber)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(ver))
}

// Create handles POST /recipes — creates a recipe together with version 1.
func (h *RecipeHandler) Create(c *gin.Context) {
	var req dto.RecipeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	rec := &model.BrewRecipe{Name: req.Name, Device: req.Device}
	ver := &model.BrewRecipeVersion{
		WaterTemp: req.WaterTemp, GrindSize: req.GrindSize, Ratio: req.Ratio, Steps: req.Steps,
	}
	created, createdVer, err := h.svc.Create(middleware.GetUserID(c), rec, ver)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(gin.H{"recipe": created, "version": createdVer}))
}

// PublishVersion handles POST /recipes/:id/versions (owner only).
func (h *RecipeHandler) PublishVersion(c *gin.Context) {
	id, ok := parseRecipeID(c)
	if !ok {
		return
	}
	var req dto.RecipeVersionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	patch := &model.BrewRecipe{Name: req.Name, Device: req.Device}
	ver := &model.BrewRecipeVersion{
		WaterTemp: req.WaterTemp, GrindSize: req.GrindSize, Ratio: req.Ratio, Steps: req.Steps,
	}
	rec, created, err := h.svc.PublishVersion(middleware.GetUserID(c), id, patch, ver)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(gin.H{"recipe": rec, "version": created}))
}

// SetActiveVersion handles PUT /recipes/:id/active-version (owner only).
func (h *RecipeHandler) SetActiveVersion(c *gin.Context) {
	id, ok := parseRecipeID(c)
	if !ok {
		return
	}
	var req dto.RecipeActiveVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	ver, err := h.svc.SetActiveVersion(middleware.GetUserID(c), id, req.VersionID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"version": ver}))
}

func parseRecipeID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid recipe id"))
		return 0, false
	}
	return uint(id), true
}
