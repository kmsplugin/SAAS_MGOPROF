package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// RefListHandler manages reference list CRUD (справочники).
type RefListHandler struct {
	repo   *repository.RefListRepository
	logger *zap.Logger
}

func NewRefListHandler(repo *repository.RefListRepository, logger *zap.Logger) *RefListHandler {
	return &RefListHandler{repo: repo, logger: logger}
}

// RegisterRoutes wires all reflist endpoints.
func (h *RefListHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	adminRole := middleware.RequireRole("admin", "super_admin")

	// Public: fetch items for a list by slug (used in registration forms)
	pub := r.Group("/reflists")
	pub.GET("/:slug/items", h.PublicItems)

	// Admin: full CRUD
	adm := r.Group("/admin/reflists", auth, adminRole)
	{
		adm.GET("", h.List)
		adm.POST("", h.Create)
		adm.GET("/:id", h.Get)
		adm.DELETE("/:id", h.Delete)
		adm.GET("/:id/items", h.ListItems)
		adm.POST("/:id/items", h.AddItem)
		adm.PUT("/:id/items/:item_id", h.UpdateItem)
		adm.DELETE("/:id/items/:item_id", h.DeleteItem)
	}
}

// List godoc
// @Summary     Список всех справочников
// @Tags        RefLists
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} model.RefList
// @Router      /api/admin/reflists [get]
func (h *RefListHandler) List(c *gin.Context) {
	lists, err := h.repo.ListAll(c.Request.Context())
	if err != nil {
		h.logger.Error("reflist list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка получения справочников."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lists": lists})
}

// Get godoc
// @Summary     Получить справочник по ID
// @Tags        RefLists
// @Produce     json
// @Security    BearerAuth
// @Param       id path int true "ID справочника"
// @Success     200 {object} model.RefList
// @Router      /api/admin/reflists/{id} [get]
func (h *RefListHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	list, err := h.repo.GetByID(c.Request.Context(), id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Справочник не найден."})
		return
	}
	if err != nil {
		h.logger.Error("reflist get", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка."})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Create godoc
// @Summary     Создать справочник
// @Tags        RefLists
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body model.CreateRefListRequest true "Данные справочника"
// @Success     201 {object} model.RefList
// @Router      /api/admin/reflists [post]
func (h *RefListHandler) Create(c *gin.Context) {
	var req model.CreateRefListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	list, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("reflist create", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка создания справочника."})
		return
	}
	c.JSON(http.StatusCreated, list)
}

// Delete godoc
// @Summary     Удалить справочник
// @Tags        RefLists
// @Security    BearerAuth
// @Param       id path int true "ID справочника"
// @Success     204
// @Router      /api/admin/reflists/{id} [delete]
func (h *RefListHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("reflist delete", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка удаления."})
		return
	}
	c.Status(http.StatusNoContent)
}

// ListItems godoc
// @Summary     Элементы справочника (admin)
// @Tags        RefLists
// @Produce     json
// @Security    BearerAuth
// @Param       id path int true "ID справочника"
// @Success     200 {array} model.RefListItem
// @Router      /api/admin/reflists/{id}/items [get]
func (h *RefListHandler) ListItems(c *gin.Context) {
	listID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	items, err := h.repo.ListItems(c.Request.Context(), listID)
	if err != nil {
		h.logger.Error("reflist items", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка получения элементов."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list_id": listID, "items": items})
}

// PublicItems returns active items for a list by slug (public, no auth).
func (h *RefListHandler) PublicItems(c *gin.Context) {
	slug := c.Param("slug")
	list, err := h.repo.GetBySlug(c.Request.Context(), slug)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Справочник не найден."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка."})
		return
	}
	items, err := h.repo.ListItems(c.Request.Context(), list.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка."})
		return
	}
	// Return only active items
	active := make([]model.RefListItem, 0, len(items))
	for _, it := range items {
		if it.IsActive {
			active = append(active, it)
		}
	}
	c.JSON(http.StatusOK, gin.H{"slug": slug, "items": active})
}

// AddItem godoc
// @Summary     Добавить элемент в справочник
// @Tags        RefLists
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path int                        true "ID справочника"
// @Param       body body model.CreateRefListItemRequest true "Элемент"
// @Success     201 {object} model.RefListItem
// @Router      /api/admin/reflists/{id}/items [post]
func (h *RefListHandler) AddItem(c *gin.Context) {
	listID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	var req model.CreateRefListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	item, err := h.repo.AddItem(c.Request.Context(), listID, req)
	if err != nil {
		h.logger.Error("reflist add item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка добавления элемента."})
		return
	}
	h.repo.UpdateListUpdatedAt(c.Request.Context(), listID)
	c.JSON(http.StatusCreated, item)
}

// UpdateItem godoc
// @Summary     Обновить элемент справочника
// @Tags        RefLists
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id      path int true "ID справочника"
// @Param       item_id path int true "ID элемента"
// @Success     200 {object} map[string]string
// @Router      /api/admin/reflists/{id}/items/{item_id} [put]
func (h *RefListHandler) UpdateItem(c *gin.Context) {
	listID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID элемента."})
		return
	}
	var body struct {
		Label     string `json:"label"      binding:"required"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	if err := h.repo.UpdateItem(c.Request.Context(), itemID, body.Label, body.SortOrder); err != nil {
		h.logger.Error("reflist update item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка обновления."})
		return
	}
	h.repo.UpdateListUpdatedAt(c.Request.Context(), listID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DeleteItem godoc
// @Summary     Удалить элемент справочника
// @Tags        RefLists
// @Security    BearerAuth
// @Param       id      path int true "ID справочника"
// @Param       item_id path int true "ID элемента"
// @Success     204
// @Router      /api/admin/reflists/{id}/items/{item_id} [delete]
func (h *RefListHandler) DeleteItem(c *gin.Context) {
	listID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID элемента."})
		return
	}
	if err := h.repo.DeleteItem(c.Request.Context(), itemID); err != nil {
		h.logger.Error("reflist delete item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка удаления."})
		return
	}
	h.repo.UpdateListUpdatedAt(c.Request.Context(), listID)
	c.Status(http.StatusNoContent)
}
