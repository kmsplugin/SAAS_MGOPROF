package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
	"mgoprof-saas/internal/service"
)

// AdminPanelHandler serves full-page HTML admin UI.
type AdminPanelHandler struct {
	eventRepo   *repository.EventRepository
	fieldSvc    *service.FieldService
	refListRepo *repository.RefListRepository
	scanSvc     *service.ScanService
	adminSvc    *service.AdminService
	logger      *zap.Logger
}

func NewAdminPanelHandler(
	eventRepo *repository.EventRepository,
	fieldSvc *service.FieldService,
	refListRepo *repository.RefListRepository,
	scanSvc *service.ScanService,
	adminSvc *service.AdminService,
	logger *zap.Logger,
) *AdminPanelHandler {
	return &AdminPanelHandler{
		eventRepo:   eventRepo,
		fieldSvc:    fieldSvc,
		refListRepo: refListRepo,
		scanSvc:     scanSvc,
		adminSvc:    adminSvc,
		logger:      logger,
	}
}

// RegisterRoutes wires all admin panel HTML pages.
func (h *AdminPanelHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	adminRole := middleware.RequireRole("admin", "super_admin")
	g := r.Group("/panel", auth, adminRole)
	{
		g.GET("", h.Dashboard)
		g.GET("/events/:id/fields", h.FormBuilder)
		g.GET("/reflists", h.RefListsPage)
		g.GET("/events/:id/attendance", h.AttendancePage)
	}
}

// ── Dashboard ─────────────────────────────────────────────────────────────────

type dashboardData struct {
	Events []model.EventWithStats
	Stats  model.Stats
	Now    time.Time
}

func (h *AdminPanelHandler) Dashboard(c *gin.Context) {
	events, err := h.adminSvc.GetEventsWithStats(c.Request.Context())
	if err != nil {
		h.logger.Error("dashboard events", zap.Error(err))
		events = []model.EventWithStats{}
	}
	stats, err := h.adminSvc.GetStats(c.Request.Context())
	if err != nil {
		h.logger.Error("dashboard stats", zap.Error(err))
		stats = &model.Stats{}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	renderAdminPage(c.Writer, "dashboard", dashboardData{Events: events, Stats: *stats, Now: time.Now()})
}

// ── Form Builder ──────────────────────────────────────────────────────────────

type formBuilderData struct {
	Event  *model.Event
	Fields []model.EventField
}

func (h *AdminPanelHandler) FormBuilder(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), eventID)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	fields, err := h.fieldSvc.ListByEvent(c.Request.Context(), eventID)
	if err != nil {
		fields = []model.EventField{}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	renderAdminPage(c.Writer, "formbuilder", formBuilderData{Event: event, Fields: fields})
}

// ── Reference Lists ───────────────────────────────────────────────────────────

type refListsPageData struct {
	Lists []model.RefList
}

func (h *AdminPanelHandler) RefListsPage(c *gin.Context) {
	lists, err := h.refListRepo.ListAll(c.Request.Context())
	if err != nil {
		lists = []model.RefList{}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	renderAdminPage(c.Writer, "reflists", refListsPageData{Lists: lists})
}

// ── Attendance Dashboard ──────────────────────────────────────────────────────

type attendancePageData struct {
	Event   *model.Event
	Entries int
	Exits   int
	Present int
}

func (h *AdminPanelHandler) AttendancePage(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), eventID)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	entries, exits, present, _ := h.scanSvc.GetAttendanceSummary(
		context.Background(), eventID)
	c.Header("Content-Type", "text/html; charset=utf-8")
	renderAdminPage(c.Writer, "attendance", attendancePageData{
		Event:   event,
		Entries: entries,
		Exits:   exits,
		Present: present,
	})
}
