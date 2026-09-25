package handler

import (
	"net/http"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/dto"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/middleware"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/service"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/util"
	"github.com/gin-gonic/gin"
)

type RopeInspectionHandler struct{ service service.RopeInspectionService }

func NewRopeInspectionHandler(s service.RopeInspectionService) *RopeInspectionHandler {
	return &RopeInspectionHandler{service: s}
}

func (h *RopeInspectionHandler) Register(group *gin.RouterGroup) {
	resource := group.Group("/rope-inspections")
	resource.GET("", h.list)
	// Gin forbids a static segment next to "/:id", so the current-conclusion
	// view lives here instead of a get-by-id route the workbench never needs.
	resource.GET("/current", h.current)
	resource.POST("", middleware.RequireMinimumRole("operator"), h.create)
}

func (h *RopeInspectionHandler) list(c *gin.Context) {
	query := bindPage(c)
	result, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}
	util.Page(c, result.Items, result.Page, result.PageSize, result.Total)
}

func (h *RopeInspectionHandler) current(c *gin.Context) {
	items, err := h.service.Current(c.Request.Context(), c.Query("planCode"))
	if err != nil {
		handleError(c, err)
		return
	}
	util.OK(c, items)
}

func (h *RopeInspectionHandler) create(c *gin.Context) {
	var input dto.CreateRopeInspection
	if err := c.ShouldBindJSON(&input); err != nil {
		util.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := h.service.Create(c.Request.Context(), input, actorFromContext(c), requestIDFromContext(c))
	if err != nil {
		handleError(c, err)
		return
	}
	util.Created(c, item)
}
