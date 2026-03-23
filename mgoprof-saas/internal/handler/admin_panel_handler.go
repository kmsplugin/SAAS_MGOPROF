package handler

import (
	"context"
	"fmt"
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
	eventRepo        *repository.EventRepository
	regRepo          *repository.RegistrationRepository
	fieldSvc         *service.FieldService
	refListRepo      *repository.RefListRepository
	scanSvc          *service.ScanService
	trackingSvc      *service.TrackingService
	adminSvc         *service.AdminService
	onlineSessionSvc *service.OnlineSessionService // may be nil for tests without sessions
	logger           *zap.Logger
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

// WithOnlineSessionService attaches the online session service so the attendance
// page can show structured session totals. This is optional — if not set, the
// page falls back to tracking-log based counters.
func (h *AdminPanelHandler) WithOnlineSessionService(svc *service.OnlineSessionService) *AdminPanelHandler {
	h.onlineSessionSvc = svc
	return h
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
	Event   *model.Event

	// Offline channel — QR check_in / check_out
	Entries int
	Exits   int
	Present int

	// Online channel — structured online_sessions (aggregate)
	OnlineActiveNow     int    // sessions with ended_at IS NULL
	OnlineTotalSessions int    // all sessions ever
	OnlineTotalSeconds  int    // sum of duration_seconds
	OnlineTotalDuration string // formatted "Xч Yм"

	// Per-participant online breakdown with name + email (populated when OnlineSessionService available)
	OnlineParticipants []model.OnlineParticipantDetail

	// Legacy fallback (used only when OnlineSessionService is unavailable)
	StreamConnects    int
	StreamDisconnects int
}

// fmtDuration converts total seconds into a human-readable "Xч Yм" string.
func fmtDuration(secs int) string {
	if secs <= 0 {
		return "0м"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	return fmt.Sprintf("%dм", m)
}

// countStreamActions tallies stream_connect / stream_disconnect tracking events.
// Returns connects, disconnects, and active count (floored at 0).
// Extracted as a pure function so it can be unit-tested without a DB or HTTP stack.
func countStreamActions(events []model.TrackingEvent) (connects, disconnects, active int) {
	for _, e := range events {
		switch e.Action {
		case "stream_connect":
			connects++
		case "stream_disconnect":
			disconnects++
		}
	}
	active = connects - disconnects
	if active < 0 {
		active = 0
	}
	return
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

	// Online and hybrid events: prefer structured session data.
	if event.EventType == "online" || event.EventType == "hybrid" {
		if h.onlineSessionSvc != nil {
			stats, err := h.onlineSessionSvc.AdminStats(c.Request.Context(), id)
			if err != nil {
				h.logger.Error("attendance page: online stats", zap.Int("event", id), zap.Error(err))
			} else {
				data.OnlineActiveNow = stats.ActiveNow
				// Aggregate totals from registration summaries (view-based, fast).
				for _, row := range stats.Registrations {
					data.OnlineTotalSessions += row.SessionCount
					data.OnlineTotalSeconds += row.TotalSeconds
				}
				data.OnlineTotalDuration = fmtDuration(data.OnlineTotalSeconds)
				// Per-participant detail (includes name + email).
				data.OnlineParticipants = stats.Participants
			}
		} else {
			// Fallback to legacy tracking counters when service not wired.
			tracking, _ := h.trackingSvc.ListByEvent(c.Request.Context(), id)
			data.StreamConnects, data.StreamDisconnects, _ = countStreamActions(tracking)
		}
	}

	// Offline and hybrid events: count QR scan check_in / check_out.
	if event.EventType != "online" {
		data.Entries, data.Exits, data.Present, _ = h.scanSvc.GetAttendanceSummary(context.Background(), id)
	}

	renderAdminPage(c.Writer, "attendance", data)
}
