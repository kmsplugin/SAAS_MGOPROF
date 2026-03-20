package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/mailer"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// QuestionHandler handles user cabinet Q&A endpoints.
type QuestionHandler struct {
	questionRepo *repository.QuestionRepository
	mailer       *mailer.Mailer
	logger       *zap.Logger
	siteURL      string
}

func NewQuestionHandler(
	questionRepo *repository.QuestionRepository,
	m *mailer.Mailer,
	logger *zap.Logger,
	siteURL string,
) *QuestionHandler {
	return &QuestionHandler{
		questionRepo: questionRepo,
		mailer:       m,
		logger:       logger,
		siteURL:      siteURL,
	}
}

// RegisterCabinetRoutes registers user-facing Q&A routes (JWT required).
func (h *QuestionHandler) RegisterCabinetRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	g := r.Group("/cabinet/questions", auth)
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.POST("/:id/messages", h.SendMessage)
	}
}

// RegisterAdminRoutes registers admin Q&A routes (JWT + admin role required).
func (h *QuestionHandler) RegisterAdminRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	// Admin questions are accessible to admin, operator, and support roles.
	// Finer-grained per-endpoint checks use RequirePermission middleware.
	g := r.Group("/admin/questions", auth)
	{
		g.GET("", h.AdminList)
		g.GET("/stats", h.AdminStats)
		g.GET("/export", h.AdminExport)
		g.GET("/:id", h.AdminGet)
		g.POST("/:id/messages", h.AdminSendMessage)
		g.PUT("/:id/status", h.AdminUpdateStatus)
		g.PUT("/:id/assign", h.AdminAssign)
	}
}

// List godoc
// @Summary Список вопросов пользователя
// @Tags Cabinet
// @Security BearerAuth
// @Router /api/cabinet/questions [get]
func (h *QuestionHandler) List(c *gin.Context) {
	userID := c.GetInt("user_id")
	questions, err := h.questionRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list questions failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions, "total": len(questions)})
}

// Create godoc
// @Summary Задать вопрос по мероприятию
// @Tags Cabinet
// @Security BearerAuth
// @Router /api/cabinet/questions [post]
func (h *QuestionHandler) Create(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req model.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Проверьте заполнение полей."})
		return
	}
	q, err := h.questionRepo.Create(c.Request.Context(), userID, req.EventID, req.Subject, req.Body)
	if err != nil {
		h.logger.Error("create question failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "question": q})
}

// Get godoc
// @Summary Просмотр вопроса + история переписки
// @Tags Cabinet
// @Security BearerAuth
// @Router /api/cabinet/questions/:id [get]
func (h *QuestionHandler) Get(c *gin.Context) {
	userID := c.GetInt("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID вопроса."})
		return
	}
	q, err := h.questionRepo.GetByID(c.Request.Context(), id, userID)
	if err != nil || q == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Вопрос не найден."})
		return
	}
	messages, err := h.questionRepo.ListMessages(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	// Mark admin messages as read by user
	_ = h.questionRepo.MarkRead(c.Request.Context(), id, "user")
	c.JSON(http.StatusOK, gin.H{"question": q, "messages": messages})
}

// SendMessage godoc
// @Summary Отправить сообщение в вопросе
// @Tags Cabinet
// @Security BearerAuth
// @Router /api/cabinet/questions/:id/messages [post]
func (h *QuestionHandler) SendMessage(c *gin.Context) {
	userID := c.GetInt("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	// Verify ownership
	q, err := h.questionRepo.GetByID(c.Request.Context(), id, userID)
	if err != nil || q == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Вопрос не найден."})
		return
	}
	if q.Status == "closed" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Вопрос закрыт."})
		return
	}
	var req model.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Текст сообщения обязателен."})
		return
	}
	msg, err := h.questionRepo.AddMessage(c.Request.Context(), id, userID, "user", req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": msg})
}

// ── Admin handlers ──────────────────────────────────────────────────────────

func (h *QuestionHandler) AdminList(c *gin.Context) {
	status := c.Query("status")
	eventID := c.Query("event_id")
	questions, err := h.questionRepo.ListAll(c.Request.Context(), status, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions, "total": len(questions)})
}

func (h *QuestionHandler) AdminStats(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Query("event_id"))
	stats, err := h.questionRepo.Stats(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *QuestionHandler) AdminExport(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Query("event_id"))
	rows, err := h.questionRepo.ForExport(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows, "total": len(rows)})
}

func (h *QuestionHandler) AdminGet(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	q, err := h.questionRepo.GetByID(c.Request.Context(), id, 0)
	if err != nil || q == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Вопрос не найден."})
		return
	}
	messages, err := h.questionRepo.ListMessages(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	_ = h.questionRepo.MarkRead(c.Request.Context(), id, "admin")
	c.JSON(http.StatusOK, gin.H{"question": q, "messages": messages})
}

func (h *QuestionHandler) AdminSendMessage(c *gin.Context) {
	adminID := c.GetInt("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	q, err := h.questionRepo.GetByID(c.Request.Context(), id, 0)
	if err != nil || q == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Вопрос не найден."})
		return
	}
	var req model.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Текст обязателен."})
		return
	}
	msg, err := h.questionRepo.AddMessage(c.Request.Context(), id, adminID, "admin", req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	// Queue email notification to user
	_ = h.questionRepo.QueueNotification(c.Request.Context(), id, q.UserID, "new_reply")
	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": msg})
}

func (h *QuestionHandler) AdminUpdateStatus(c *gin.Context) {
	adminID := c.GetInt("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	var req model.UpdateQuestionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный статус."})
		return
	}
	if err := h.questionRepo.UpdateStatus(c.Request.Context(), id, adminID, req.Status, req.Comment); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	// Notify user of status change
	q, _ := h.questionRepo.GetByID(c.Request.Context(), id, 0)
	if q != nil {
		_ = h.questionRepo.QueueNotification(c.Request.Context(), id, q.UserID, "status_changed")
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *QuestionHandler) AdminAssign(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	var req model.AssignQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Укажите admin_id."})
		return
	}
	if err := h.questionRepo.Assign(c.Request.Context(), id, req.AdminID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// RequirePermission returns a middleware checking a specific permission key.
// Usage: RequirePermission(adminRepo, "questions.assign")
func RequirePermission(adminRepo *repository.AdminRepository, permKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt("user_id")
		ok, err := adminRepo.HasPermission(c.Request.Context(), adminID, permKey)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorResponse{
				Status:  "error",
				Message: "Недостаточно прав: " + permKey,
			})
			return
		}
		c.Next()
	}
}
