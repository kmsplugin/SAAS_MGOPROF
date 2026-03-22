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

// AdminPanelHandler serves full-page HTML admin UI (server-side rendered).
// All routes live under /api/panel and require admin JWT + role middleware.
type AdminPanelHandler struct {
	eventRepo   *repository.EventRepository
	regRepo     *repository.RegistrationRepository
	fieldSvc    *service.FieldService
	refListRepo *repository.RefListRepository
	scanSvc     *service.ScanService
	trackingSvc *service.TrackingService
	adminSvc    *service.AdminService
	logger      *zap.Logger
}

func NewAdminPanelHandler(
	eventRepo *repository.EventRepository,
	regRepo *repository.RegistrationRepository,
	fieldSvc *service.FieldService,
	refListRepo *repository.RefListRepository,
	scanSvc *service.ScanService,
	trackingSvc *service.TrackingService,
	adminSvc *service.AdminService,
	logger *zap.Logger,
) *AdminPanelHandler {
	return &AdminPanelHandler{
		eventRepo:   eventRepo,
		regRepo:     regRepo,
		fieldSvc:    fieldSvc,
		refListRepo: refListRepo,
		scanSvc:     scanSvc,
		trackingSvc: trackingSvc,
		adminSvc:    adminSvc,
		logger:      logger,
	}
}

// RegisterRoutes wires all admin panel HTML page routes.
func (h *AdminPanelHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	adminRole := middleware.RequireRole("admin", "super_admin")
	g := r.Group("/panel", auth, adminRole)
	{
		g.GET("",  h.Dashboard)

		// Events
		g.GET("/events",         h.EventsPage)
		g.GET("/events/new",     h.EventFormPage)
		g.GET("/events/:id/edit",          h.EventFormPage)
		g.GET("/events/:id/fields",        h.FormBuilder)
		g.GET("/events/:id/registrations", h.EventRegistrationsPage)
		g.GET("/events/:id/attendance",    h.AttendancePage)
		g.GET("/events/:id/tracking",      h.TrackingPage)

		// Global views
		g.GET("/registrations", h.RegistrationsPage)
		g.GET("/reflists",      h.RefListsPage)
		g.GET("/logs",          h.LogsPage)
	}
}

// ── Page data structs ─────────────────────────────────────────────────────────

type basePage struct {
	Page  string // nav active item
	Title string
}

type dashboardData struct {
	basePage
	Events []model.EventWithStats
	Stats  model.Stats
	Recent []model.RegistrationRow
	Now    time.Time
}

type eventsPageData struct {
	basePage
	Events []model.EventWithStats
}

type eventFormData struct {
	basePage
	Event    *model.Event
	IsCreate bool
}

type registrationsPageData struct {
	basePage
	Registrations []model.RegistrationRow
	Total         int
	EventID       int
	EventTitle    string
}

type trackingPageData struct {
	basePage
	Event    *model.Event
	Tracking []model.TrackingEvent
	Total    int
	Counts   map[string]int
}

type logsPageData struct {
	basePage
	Logs  []model.Log
	Total int
}

type refListsPageData struct {
	basePage
	Lists []model.RefList
}

type formBuilderData struct {
	basePage
	Event  *model.Event
	Fields []model.EventField
}

type attendancePageData struct {
	basePage
	Event             *model.Event
	// Offline (check_in / check_out based)
	Entries           int
	Exits             int
	Present           int
	// Online (stream_connect / stream_disconnect based)
	StreamConnects    int
	StreamDisconnects int
	OnlineActive      int
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *AdminPanelHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	events, err := h.adminSvc.GetEventsWithStats(ctx)
	if err != nil {
		h.logger.Error("dashboard events", zap.Error(err))
		events = []model.EventWithStats{}
	}
	stats, err := h.adminSvc.GetStats(ctx)
	if err != nil {
		h.logger.Error("dashboard stats", zap.Error(err))
		stats = &model.Stats{}
	}
	recent, err := h.adminSvc.GetRecentRegistrations(ctx, 15)
	if err != nil {
		recent = []model.RegistrationRow{}
	}

	renderAdminPage(c.Writer, "dashboard", dashboardData{
		basePage: basePage{Page: "dashboard", Title: "Дашборд"},
		Events:   events,
		Stats:    *stats,
		Recent:   recent,
		Now:      time.Now(),
	})
}

func (h *AdminPanelHandler) EventsPage(c *gin.Context) {
	events, err := h.adminSvc.GetEventsWithStats(c.Request.Context())
	if err != nil {
		h.logger.Error("events page", zap.Error(err))
		events = []model.EventWithStats{}
	}
	renderAdminPage(c.Writer, "events", eventsPageData{
		basePage: basePage{Page: "events", Title: "Мероприятия"},
		Events:   events,
	})
}

func (h *AdminPanelHandler) EventFormPage(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		// Create mode
		renderAdminPage(c.Writer, "event_form", eventFormData{
			basePage: basePage{Page: "events", Title: "Создать мероприятие"},
			IsCreate: true,
		})
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil || id < 1 {
		c.String(http.StatusBadRequest, "Неверный ID мероприятия")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), id)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	renderAdminPage(c.Writer, "event_form", eventFormData{
		basePage: basePage{Page: "events", Title: "Редактировать мероприятие"},
		Event:    event,
		IsCreate: false,
	})
}

func (h *AdminPanelHandler) EventRegistrationsPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), id)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	regs, err := h.regRepo.ListByEvent(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("event registrations", zap.Int("event", id), zap.Error(err))
		regs = []model.RegistrationRow{}
	}
	renderAdminPage(c.Writer, "registrations", registrationsPageData{
		basePage:      basePage{Page: "registrations", Title: "Участники: " + event.Title},
		Registrations: regs,
		Total:         len(regs),
		EventID:       id,
		EventTitle:    event.Title,
	})
}

func (h *AdminPanelHandler) RegistrationsPage(c *gin.Context) {
	regs, err := h.adminSvc.GetRegistrations(c.Request.Context())
	if err != nil {
		h.logger.Error("registrations page", zap.Error(err))
		regs = []model.RegistrationRow{}
	}
	renderAdminPage(c.Writer, "registrations", registrationsPageData{
		basePage:      basePage{Page: "registrations", Title: "Регистрации"},
		Registrations: regs,
		Total:         len(regs),
	})
}

func (h *AdminPanelHandler) TrackingPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), id)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	tracking, err := h.trackingSvc.ListByEvent(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("tracking page", zap.Int("event", id), zap.Error(err))
		tracking = []model.TrackingEvent{}
	}

	// Aggregate counts per action for the summary cards
	counts := make(map[string]int)
	for _, t := range tracking {
		counts[t.Action]++
	}

	renderAdminPage(c.Writer, "tracking", trackingPageData{
		basePage: basePage{Page: "events", Title: "Трекинг: " + event.Title},
		Event:    event,
		Tracking: tracking,
		Total:    len(tracking),
		Counts:   counts,
	})
}

func (h *AdminPanelHandler) LogsPage(c *gin.Context) {
	limit := 200
	logs, err := h.adminSvc.GetLogs(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("logs page", zap.Error(err))
		logs = []model.Log{}
	}
	renderAdminPage(c.Writer, "logs", logsPageData{
		basePage: basePage{Page: "logs", Title: "Логи"},
		Logs:     logs,
		Total:    len(logs),
	})
}

func (h *AdminPanelHandler) FormBuilder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), id)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}
	fields, err := h.fieldSvc.ListByEvent(c.Request.Context(), id)
	if err != nil {
		fields = []model.EventField{}
	}
	renderAdminPage(c.Writer, "formbuilder", formBuilderData{
		basePage: basePage{Page: "events", Title: "Поля формы: " + event.Title},
		Event:    event,
		Fields:   fields,
	})
}

func (h *AdminPanelHandler) RefListsPage(c *gin.Context) {
	lists, err := h.refListRepo.ListAll(c.Request.Context())
	if err != nil {
		lists = []model.RefList{}
	}
	renderAdminPage(c.Writer, "reflists", refListsPageData{
		basePage: basePage{Page: "reflists", Title: "Справочники"},
		Lists:    lists,
	})
}

func (h *AdminPanelHandler) AttendancePage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.String(http.StatusBadRequest, "Неверный ID")
		return
	}
	event, err := h.eventRepo.FindByID(c.Request.Context(), id)
	if err != nil || event == nil {
		c.String(http.StatusNotFound, "Мероприятие не найдено")
		return
	}

	data := attendancePageData{
		basePage: basePage{Page: "events", Title: "Присутствие: " + event.Title},
		Event:    event,
	}

	if event.EventType == "online" {
		// For online events, count stream_connect / stream_disconnect tracking events.
		// Present = connects − disconnects (floor 0).
		tracking, _ := h.trackingSvc.ListByEvent(c.Request.Context(), id)
		for _, t := range tracking {
			switch t.Action {
			case "stream_connect":
				data.StreamConnects++
			case "stream_disconnect":
				data.StreamDisconnects++
			}
		}
		data.OnlineActive = data.StreamConnects - data.StreamDisconnects
		if data.OnlineActive < 0 {
			data.OnlineActive = 0
		}
	} else {
		// For offline/hybrid events, use QR scan check_in / check_out counts.
		data.Entries, data.Exits, data.Present, _ = h.scanSvc.GetAttendanceSummary(context.Background(), id)
	}

	renderAdminPage(c.Writer, "attendance", data)
}
