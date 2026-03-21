package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/model"
	"platform/api/internal/service"
)

type QuestionHandler struct {
	svc    *service.QuestionService
	logger *zap.Logger
}

func NewQuestionHandler(svc *service.QuestionService, logger *zap.Logger) *QuestionHandler {
	return &QuestionHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers Q&A routes under /events/:id/questions
func (h *QuestionHandler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	g := r.Group("/events/:id/questions", authMW)
	g.GET("", h.ListQuestions)
	g.POST("", h.CreateQuestion)
	g.GET("/:qid/messages", h.ListMessages)
	g.POST("/:qid/messages", h.AddMessage)
}

func (h *QuestionHandler) ListQuestions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	role := c.GetString("role")
	questions, err := h.svc.ListQuestions(c.Request.Context(), tenantID, c.Param("id"), role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions, "total": len(questions)})
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	var req model.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	q, err := h.svc.CreateQuestion(c.Request.Context(), tenantID, c.Param("id"), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"question": q})
}

func (h *QuestionHandler) ListMessages(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	role := c.GetString("role")
	messages, err := h.svc.ListMessages(c.Request.Context(), tenantID, c.Param("qid"), userID, role)
	if err != nil {
		h.logger.Warn("list messages", zap.Error(err))
		c.JSON(http.StatusForbidden, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages, "total": len(messages)})
}

func (h *QuestionHandler) AddMessage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	role := c.GetString("role")
	var req model.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	msg, err := h.svc.AddMessage(c.Request.Context(), tenantID, c.Param("qid"), userID, role, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": msg})
}
