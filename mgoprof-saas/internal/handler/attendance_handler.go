package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// AttendanceHandler provides QR scan and attendance tracking endpoints.
type AttendanceHandler struct {
	scanSvc *service.ScanService
	logger  *zap.Logger
}

func NewAttendanceHandler(scanSvc *service.ScanService, logger *zap.Logger) *AttendanceHandler {
	return &AttendanceHandler{scanSvc: scanSvc, logger: logger}
}

// RegisterRoutes registers all scan/attendance endpoints under /api.
func (h *AttendanceHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	adminRole := middleware.RequireRole("admin", "super_admin")
	g := r.Group("/admin/events/:id", auth, adminRole)
	{
		// POST /api/admin/events/:id/scan
		g.POST("/scan", h.Scan)
		// GET  /api/admin/events/:id/scan/history
		g.GET("/scan/history", h.ScanHistory)
		// GET  /api/admin/events/:id/attendance
		g.GET("/attendance", h.AttendanceSummary)
		// GET  /api/admin/events/:id/scanner  — mobile scanner UI page
		g.GET("/scanner", h.ScannerPage)
	}
}

// Scan godoc
// @Summary     Обработать QR-скан
// @Tags        Attendance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path int                 true  "ID мероприятия"
// @Param       body body model.ScanRequest  true  "Токен и режим сканирования"
// @Success     200 {object} model.ScanResponse
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/admin/events/{id}/scan [post]
func (h *AttendanceHandler) Scan(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}

	var req model.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный формат запроса."})
		return
	}

	// Resolve operator from JWT claims (optional — may be nil for API calls without auth)
	var operatorID *int
	if id, exists := c.Get("admin_id"); exists {
		if idInt, ok := id.(int); ok {
			operatorID = &idInt
		}
	}

	ip := middleware.ExtractIP(c)

	resp, err := h.scanSvc.ProcessScan(c.Request.Context(), eventID, req, operatorID, ip)
	if err != nil {
		h.logger.Error("scan processing error", zap.Int("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка обработки скана."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ScanHistory godoc
// @Summary     История сканирований мероприятия
// @Tags        Attendance
// @Produce     json
// @Security    BearerAuth
// @Param       id    path  int true  "ID мероприятия"
// @Param       limit query int false "Количество записей (по умолчанию 100)"
// @Success     200 {array} model.ScanLog
// @Router      /api/admin/events/{id}/scan/history [get]
func (h *AttendanceHandler) ScanHistory(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	logs, err := h.scanSvc.GetScanHistory(c.Request.Context(), eventID, limit)
	if err != nil {
		h.logger.Error("scan history", zap.Int("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка получения истории."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"event_id": eventID, "logs": logs, "count": len(logs)})
}

// AttendanceSummary godoc
// @Summary     Сводка присутствия на мероприятии
// @Tags        Attendance
// @Produce     json
// @Security    BearerAuth
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} map[string]int
// @Router      /api/admin/events/{id}/attendance [get]
func (h *AttendanceHandler) AttendanceSummary(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}

	entries, exits, present, err := h.scanSvc.GetAttendanceSummary(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("attendance summary", zap.Int("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка получения данных."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id": eventID,
		"entries":  entries,
		"exits":    exits,
		"present":  present,
	})
}

// ScannerPage serves the mobile-optimised QR scanner web UI.
func (h *AttendanceHandler) ScannerPage(c *gin.Context) {
	eventID := c.Param("id")
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, scannerPageHTML, eventID)
}

// scannerPageHTML is the inline mobile QR scanner UI.
// Uses jsQR (CDN) for decoding, Fetch API for reporting scans.
const scannerPageHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">
<title>Сканер QR · Мероприятие %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0f172a;color:#f1f5f9;font-family:system-ui,sans-serif;min-height:100vh;display:flex;flex-direction:column}
header{background:#1e293b;padding:12px 16px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #334155}
header h1{font-size:15px;font-weight:600}
.mode-bar{display:flex;gap:6px;padding:10px 16px;background:#1e293b;border-bottom:1px solid #334155}
.mode-btn{flex:1;padding:8px 4px;border-radius:8px;border:1px solid #334155;background:transparent;color:#94a3b8;font-size:12px;font-weight:600;cursor:pointer;transition:all .15s}
.mode-btn.active[data-mode=entry]{background:#16a34a22;border-color:#16a34a;color:#4ade80}
.mode-btn.active[data-mode=exit]{background:#dc262622;border-color:#dc2626;color:#f87171}
.mode-btn.active[data-mode=verify]{background:#6366f122;border-color:#6366f1;color:#818cf8}
.scanner-wrap{position:relative;width:100%;background:#000;aspect-ratio:1}
#video{width:100%;height:100%;object-fit:cover;display:block}
.crosshair{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);width:200px;height:200px;border:2px solid rgba(255,255,255,.5);border-radius:12px;pointer-events:none}
.crosshair::before,.crosshair::after{content:'';position:absolute;background:#fff}
.flash{position:absolute;inset:0;background:#fff;opacity:0;pointer-events:none;transition:opacity .1s}
.flash.show{opacity:.4}
.result-panel{padding:16px;flex:1}
#result-box{background:#1e293b;border-radius:12px;padding:16px;min-height:120px;border:1px solid #334155}
#result-box .ok{color:#4ade80}
#result-box .error{color:#f87171}
#result-box .duplicate{color:#fbbf24}
.person-name{font-size:18px;font-weight:700;margin-bottom:4px}
.person-org{font-size:13px;color:#94a3b8;margin-bottom:8px}
.status-badge{display:inline-block;padding:4px 10px;border-radius:20px;font-size:11px;font-weight:600;background:#16a34a22;color:#4ade80;border:1px solid #16a34a}
.manual-row{display:flex;gap:8px;padding:0 16px 16px}
#manual-input{flex:1;background:#1e293b;border:1px solid #334155;border-radius:8px;padding:10px 12px;color:#f1f5f9;font-size:14px}
#manual-btn{background:#6366f1;color:#fff;border:none;border-radius:8px;padding:10px 16px;font-size:13px;font-weight:600;cursor:pointer}
.stats-row{display:flex;gap:8px;padding:0 16px 12px}
.stat-card{flex:1;background:#1e293b;border:1px solid #334155;border-radius:10px;padding:10px;text-align:center}
.stat-card .num{font-size:22px;font-weight:700;color:#f1f5f9}
.stat-card .lbl{font-size:11px;color:#64748b;margin-top:2px}
</style>
</head>
<body>
<header>
  <h1>QR Сканер — Мероприятие #%s</h1>
  <span id="scan-count" style="font-size:12px;color:#64748b">0 сканов</span>
</header>

<div class="mode-bar">
  <button class="mode-btn active" data-mode="entry">✅ Вход</button>
  <button class="mode-btn" data-mode="exit">🚪 Выход</button>
  <button class="mode-btn" data-mode="verify">🔍 Проверка</button>
</div>

<div class="scanner-wrap">
  <video id="video" autoplay muted playsinline></video>
  <div class="crosshair"></div>
  <div class="flash" id="flash"></div>
</div>

<div class="stats-row" id="stats-row">
  <div class="stat-card"><div class="num" id="stat-entries">—</div><div class="lbl">Входов</div></div>
  <div class="stat-card"><div class="num" id="stat-exits">—</div><div class="lbl">Выходов</div></div>
  <div class="stat-card"><div class="num" id="stat-present">—</div><div class="lbl">Сейчас</div></div>
</div>

<div class="result-panel">
  <div id="result-box">
    <p style="color:#475569;text-align:center;padding-top:30px">Направьте камеру на QR-код участника</p>
  </div>
</div>

<div class="manual-row">
  <input id="manual-input" type="text" placeholder="Ввести токен вручную…">
  <button id="manual-btn">Ввести</button>
</div>

<script src="https://cdn.jsdelivr.net/npm/jsqr@1.4.0/dist/jsQR.min.js"></script>
<script>
const EVENT_ID = '%s';
const API_BASE = '/api/admin/events/' + EVENT_ID;
let mode = 'entry';
let scanning = true;
let totalScans = 0;
const token = localStorage.getItem('admin_token') || '';

// Mode buttons
document.querySelectorAll('.mode-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.mode-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    mode = btn.dataset.mode;
  });
});

// Camera
const video = document.getElementById('video');
const canvas = document.createElement('canvas');
const ctx = canvas.getContext('2d');

async function startCamera() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } }
    });
    video.srcObject = stream;
    video.play();
    requestAnimationFrame(tick);
  } catch(e) {
    showResult({ result: 'error', message: 'Камера недоступна: ' + e.message });
  }
}

function tick() {
  if (video.readyState === video.HAVE_ENOUGH_DATA) {
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    const img = ctx.getImageData(0, 0, canvas.width, canvas.height);
    const code = jsQR(img.data, img.width, img.height);
    if (code && scanning) {
      scanning = false;
      flash();
      processToken(code.data);
      setTimeout(() => { scanning = true; }, 2500);
    }
  }
  requestAnimationFrame(tick);
}

function flash() {
  const f = document.getElementById('flash');
  f.classList.add('show');
  setTimeout(() => f.classList.remove('show'), 150);
}

async function processToken(tokenValue) {
  try {
    const resp = await fetch(API_BASE + '/scan', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + token
      },
      body: JSON.stringify({ token: tokenValue, mode, device_info: navigator.userAgent })
    });
    const data = await resp.json();
    totalScans++;
    document.getElementById('scan-count').textContent = totalScans + ' сканов';
    showResult(data);
    refreshStats();
  } catch(e) {
    showResult({ result: 'error', message: 'Сетевая ошибка: ' + e.message });
  }
}

function showResult(data) {
  const box = document.getElementById('result-box');
  let cls = 'ok';
  if (data.result === 'duplicate') cls = 'duplicate';
  if (['error','not_found','wrong_event','cancelled'].includes(data.result)) cls = 'error';

  let html = '<div class="' + cls + '">';
  if (data.user) {
    html += '<div class="person-name">' + esc(data.user.last_name) + ' ' + esc(data.user.first_name) + '</div>';
    if (data.user.organization) html += '<div class="person-org">' + esc(data.user.organization) + '</div>';
  }
  html += '<div>' + esc(data.message) + '</div>';
  if (data.status_extended) {
    html += '<div style="margin-top:8px"><span class="status-badge">' + esc(data.status_extended) + '</span></div>';
  }
  html += '</div>';
  box.innerHTML = html;
}

function esc(s) {
  return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

async function refreshStats() {
  try {
    const resp = await fetch(API_BASE + '/attendance', {
      headers: { 'Authorization': 'Bearer ' + token }
    });
    const data = await resp.json();
    document.getElementById('stat-entries').textContent = data.entries ?? '—';
    document.getElementById('stat-exits').textContent = data.exits ?? '—';
    document.getElementById('stat-present').textContent = data.present ?? '—';
  } catch(_) {}
}

// Manual token input
document.getElementById('manual-btn').addEventListener('click', () => {
  const val = document.getElementById('manual-input').value.trim();
  if (val) { processToken(val); document.getElementById('manual-input').value = ''; }
});
document.getElementById('manual-input').addEventListener('keydown', e => {
  if (e.key === 'Enter') document.getElementById('manual-btn').click();
});

startCamera();
refreshStats();
setInterval(refreshStats, 15000);
</script>
</body>
</html>`
