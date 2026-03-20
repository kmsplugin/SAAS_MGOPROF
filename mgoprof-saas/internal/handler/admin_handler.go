package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// AdminHandler exposes admin panel endpoints.
type AdminHandler struct {
	adminSvc *service.AdminService
	eventSvc *service.EventService
	authSvc  *service.AuthService
	logger   *zap.Logger
}

func NewAdminHandler(
	adminSvc *service.AdminService,
	eventSvc *service.EventService,
	authSvc *service.AuthService,
	logger *zap.Logger,
) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, eventSvc: eventSvc, authSvc: authSvc, logger: logger}
}

// RegisterProtectedRoutes registers all JWT-protected admin endpoints.
// Public login routes are registered separately in main.go to allow rate limiting.
func (h *AdminHandler) RegisterProtectedRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	admin := r.Group("/admin", auth, middleware.RequireRole("admin"))
	{
		admin.GET("/stats", h.Stats)
		admin.GET("/registrations", h.Registrations)
		admin.GET("/registrations/recent", h.RecentRegistrations)
		admin.GET("/export", h.Export)
		admin.GET("/logs", h.Logs)

		admin.GET("/events", h.Events)
		admin.GET("/events/:id", h.GetEvent)
		admin.POST("/events", h.CreateEvent)
		admin.PUT("/events/:id", h.UpdateEvent)
	}
}

// Login godoc
// @Summary     Вход администратора (шаг 1 — пароль)
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Param       body body model.AdminLoginRequest true "Логин + пароль"
// @Success     200 {object} map[string]string
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/admin/login [post]
func (h *AdminHandler) Login(c *gin.Context) {
	var req model.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	if err := h.adminSvc.Login(c.Request.Context(), req, ip, ua); err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "otp_sent",
		"message": "OTP-код отправлен на email администратора.",
	})
}

// VerifyOTP godoc
// @Summary     Подтверждение OTP администратора (шаг 2)
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Param       body body model.AdminVerifyOTPRequest true "Email + OTP"
// @Success     200 {object} map[string]string
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/admin/verify-otp [post]
func (h *AdminHandler) VerifyOTP(c *gin.Context) {
	var req model.AdminVerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	token, err := h.adminSvc.VerifyOTP(c.Request.Context(), req, ip, ua)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "token": token})
}

// Stats godoc
// @Summary     Статистика панели
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} model.Stats
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/admin/stats [get]
func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.adminSvc.GetStats(c.Request.Context())
	if err != nil {
		h.logger.Error("admin stats failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Registrations godoc
// @Summary     Все регистрации
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} []model.RegistrationRow
// @Router      /api/admin/registrations [get]
func (h *AdminHandler) Registrations(c *gin.Context) {
	rows, err := h.adminSvc.GetRegistrations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"registrations": rows, "total": len(rows)})
}

// RecentRegistrations godoc
// @Summary     Последние регистрации (для дашборда)
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       limit query int false "Лимит (по умолчанию 20)"
// @Success     200 {object} []model.RegistrationRow
// @Router      /api/admin/registrations/recent [get]
func (h *AdminHandler) RecentRegistrations(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	rows, err := h.adminSvc.GetRecentRegistrations(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"registrations": rows})
}

// Export godoc
// @Summary     Экспорт регистраций в CSV
// @Tags        Admin
// @Security    BearerAuth
// @Produce     text/csv
// @Success     200
// @Router      /api/admin/export [get]
func (h *AdminHandler) Export(c *gin.Context) {
	rows, err := h.adminSvc.GetRegistrations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	filename := fmt.Sprintf("registrations_%s.csv", time.Now().Format("2006-01-02_15-04"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// UTF-8 BOM for Excel compatibility
	c.Writer.Write([]byte("\xEF\xBB\xBF"))

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"Дата и время", "Мероприятие",
		"Фамилия", "Имя", "Отчество",
		"Организация", "Округ", "Email",
		"Член профсоюза", "Номер билета", "Другое",
		"IP", "Страна", "Регион", "Город", "Статус",
	})

	unionMember := func(b bool) string {
		if b {
			return "Да"
		}
		return "Нет"
	}

	for _, r := range rows {
		_ = w.Write([]string{
			r.RegDatetime.Format("02.01.2006 15:04"),
			r.EventTitle,
			r.LastName, r.FirstName, r.Patronymic,
			r.Organization, r.District, r.Email,
			unionMember(r.IsUnionMember), r.UnionTicket, r.ExtraInfo,
			r.IPAddress, r.GeoCountry, r.GeoRegion, r.GeoCity,
			r.Status,
		})
	}
	w.Flush()
}

// Logs godoc
// @Summary     Системные логи
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       limit query int false "Лимит (по умолчанию 100)"
// @Success     200 {object} []model.Log
// @Router      /api/admin/logs [get]
func (h *AdminHandler) Logs(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	logs, err := h.adminSvc.GetLogs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}

// Events godoc
// @Summary     Все мероприятия со статистикой (admin)
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} []model.EventWithStats
// @Router      /api/admin/events [get]
func (h *AdminHandler) Events(c *gin.Context) {
	events, err := h.adminSvc.GetEventsWithStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetEvent godoc
// @Summary     Получить мероприятие для редактирования
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID"
// @Success     200 {object} model.Event
// @Router      /api/admin/events/{id} [get]
func (h *AdminHandler) GetEvent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	event, err := h.eventSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Мероприятие не найдено."})
		return
	}
	c.JSON(http.StatusOK, event)
}

// CreateEvent godoc
// @Summary     Создать мероприятие
// @Tags        Admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body model.CreateEventRequest true "Данные мероприятия"
// @Success     201 {object} model.Event
// @Router      /api/admin/events [post]
func (h *AdminHandler) CreateEvent(c *gin.Context) {
	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	event, err := h.eventSvc.Create(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("create event failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, event)
}

// UpdateEvent godoc
// @Summary     Обновить мероприятие
// @Tags        Admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id   path int                     true "ID"
// @Param       body body model.CreateEventRequest true "Данные мероприятия"
// @Success     200 {object} model.Event
// @Router      /api/admin/events/{id} [put]
func (h *AdminHandler) UpdateEvent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	event, err := h.eventSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		h.logger.Error("update event failed", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, event)
}
