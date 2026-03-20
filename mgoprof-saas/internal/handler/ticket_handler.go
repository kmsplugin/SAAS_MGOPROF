package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// TicketHandler serves the participant ticket (QR) page and admin check-in.
type TicketHandler struct {
	ticketSvc *service.TicketService
	logger    *zap.Logger
	tmpl      *template.Template
}

func NewTicketHandler(svc *service.TicketService, logger *zap.Logger) *TicketHandler {
	tmpl := template.Must(template.New("ticket").Funcs(template.FuncMap{
		"fmtDate": func(t time.Time) string { return t.Format("02.01.2006") },
		"fmtTime": func(t time.Time) string { return t.Format("15:04") },
		"jsonJS":  func(v interface{}) template.JS { b, _ := json.Marshal(v); return template.JS(b) },
	}).Parse(ticketHTMLTemplate))
	return &TicketHandler{ticketSvc: svc, logger: logger, tmpl: tmpl}
}

// RegisterRoutes wires ticket and check-in routes.
func (h *TicketHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	// Participant: authenticated ticket page
	r.GET("/cabinet/events/:id/ticket", auth, h.TicketPage)

	// Admin: JSON check-in endpoint + check-in scanner page
	admin := r.Group("/admin/checkin", auth, middleware.RequireRole("admin"))
	admin.POST("", h.CheckIn)
	admin.GET("/scan", h.ScanPage)
}

// TicketPage godoc
// @Summary     HTML-билет участника с QR-кодом
// @Tags        Cabinet
// @Security    BearerAuth
// @Produce     html
// @Param       id path int true "ID мероприятия"
// @Success     200 {string} string "HTML страница"
// @Router      /api/cabinet/events/{id}/ticket [get]
func (h *TicketHandler) TicketPage(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	userID, _ := c.Get("user_id")
	uid, ok := userID.(int)
	if !ok || uid < 1 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: "требуется авторизация"})
		return
	}

	info, err := h.ticketSvc.GetTicket(c.Request.Context(), uid, eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(c.Writer, info); err != nil {
		h.logger.Error("ticket template execute", zap.Error(err))
	}
}

// CheckIn godoc
// @Summary     Отметить присутствие по токену (admin)
// @Tags        Admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body model.CheckInRequest true "Токен участника"
// @Success     200 {object} model.CheckInResult
// @Router      /api/admin/checkin [post]
func (h *TicketHandler) CheckIn(c *gin.Context) {
	var req model.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	result, err := h.ticketSvc.CheckIn(c.Request.Context(), req.Token)
	if err != nil {
		h.logger.Warn("checkin failed", zap.String("token", req.Token), zap.Error(err))
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ScanPage serves a simple camera-scan page for admin at the entrance.
func (h *TicketHandler) ScanPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(c.Writer, scanHTMLPage)
}

// ─── Ticket HTML ──────────────────────────────────────────────────────────────

const ticketHTMLTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Билет: {{.Event.Title}}</title>
<script src="https://cdn.jsdelivr.net/npm/qrcode@1.5.3/build/qrcode.min.js"></script>
<style>
:root{--o:#ff7c2c;--g:#009b35;--bg:#f0f4f8;--card:#fff;
  --text:#1e293b;--muted:#64748b;--grad:linear-gradient(135deg,#ff7c2c,#009b35)}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);
  min-height:100vh;display:flex;align-items:center;justify-content:center;padding:20px}
.ticket{background:var(--card);border-radius:24px;box-shadow:0 12px 48px rgba(0,0,0,.12);
  width:100%;max-width:420px;overflow:hidden}
.ticket-header{background:linear-gradient(135deg,#0f172a 60%,#1e3a2f);color:#fff;
  padding:24px 28px 20px;position:relative;overflow:hidden}
.ticket-header::after{content:'';position:absolute;right:-40px;top:-40px;width:180px;height:180px;
  border-radius:50%;background:rgba(255,124,44,.12);pointer-events:none}
.event-title{font-size:18px;font-weight:700;line-height:1.3;position:relative;z-index:1}
.event-meta{font-size:13px;color:rgba(255,255,255,.65);margin-top:6px;position:relative;z-index:1}
.divider{height:3px;background:var(--grad)}
.ticket-body{padding:24px 28px}
.participant-name{font-size:20px;font-weight:800;letter-spacing:-.3px;margin-bottom:4px}
.participant-meta{font-size:13px;color:var(--muted);margin-bottom:20px}
.qr-section{display:flex;flex-direction:column;align-items:center;gap:12px}
canvas#qr{border-radius:12px}
.token-box{background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;
  padding:10px 16px;text-align:center;width:100%}
.token-label{font-size:11px;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;margin-bottom:4px}
.token-val{font-family:monospace;font-size:18px;font-weight:700;letter-spacing:2px;color:var(--text)}
{{if .Registration.CheckedInAt}}
.checked-in-badge{display:inline-flex;align-items:center;gap:6px;background:#dcfce7;color:#166534;
  border:1px solid #bbf7d0;border-radius:20px;padding:6px 14px;font-size:13px;font-weight:600;margin-bottom:16px}
{{end}}
.info-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-top:16px}
.info-item{background:#f8fafc;border-radius:10px;padding:10px 12px}
.info-item-label{font-size:10px;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;margin-bottom:2px}
.info-item-val{font-size:13px;font-weight:600;color:var(--text)}
.ticket-footer{padding:16px 28px 24px;text-align:center}
.btn-back{display:inline-flex;align-items:center;gap:6px;padding:10px 20px;border-radius:10px;
  background:var(--grad);color:#fff;text-decoration:none;font-size:13px;font-weight:600;
  box-shadow:0 4px 14px rgba(255,124,44,.3)}
@media print{.ticket-footer,.btn-back{display:none}
  body{background:#fff;padding:0}
  .ticket{box-shadow:none;border-radius:0;max-width:100%}}
</style>
</head>
<body>
<div class="ticket">
  <div class="ticket-header">
    <div class="event-title">{{.Event.Title}}</div>
    <div class="event-meta">{{.Event.EventDate}} · {{.Event.EventTime}}</div>
  </div>
  <div class="divider"></div>
  <div class="ticket-body">
    <div class="participant-name">{{.User.LastName}} {{.User.FirstName}} {{.User.Patronymic}}</div>
    <div class="participant-meta">{{.User.Organization}} · {{.User.District}}</div>

    {{if .Registration.CheckedInAt}}
    <div class="checked-in-badge">✓ Вход зарегистрирован</div>
    {{end}}

    <div class="qr-section">
      <canvas id="qr"></canvas>
      <div class="token-box">
        <div class="token-label">Код участника</div>
        <div class="token-val">{{.Registration.ParticipantToken}}</div>
      </div>
    </div>

    <div class="info-grid">
      <div class="info-item">
        <div class="info-item-label">Мероприятие</div>
        <div class="info-item-val">{{.Event.EventDate}}</div>
      </div>
      <div class="info-item">
        <div class="info-item-label">Статус</div>
        <div class="info-item-val" style="color:#009b35">Подтверждено</div>
      </div>
      <div class="info-item">
        <div class="info-item-label">Членство</div>
        <div class="info-item-val">{{if .User.IsUnionMember}}Да{{else}}Нет{{end}}</div>
      </div>
      <div class="info-item">
        <div class="info-item-label">Email</div>
        <div class="info-item-val" style="font-size:11px">{{.User.Email}}</div>
      </div>
    </div>
  </div>
  <div class="ticket-footer">
    <a class="btn-back" href="javascript:history.back()">← Назад</a>
    &nbsp;
    <button onclick="window.print()" style="margin-left:8px;padding:10px 20px;border-radius:10px;
      border:1px solid #e2e8f0;background:#fff;font-size:13px;font-weight:600;cursor:pointer">
      🖨 Печать
    </button>
  </div>
</div>
<script>
(function(){
  var token = {{jsonJS .Registration.ParticipantToken}};
  var ticketURL = {{jsonJS .TicketURL}};
  QRCode.toCanvas(document.getElementById('qr'), ticketURL || token, {
    width: 200,
    margin: 1,
    color: { dark: '#1e293b', light: '#ffffff' }
  });
})();
</script>
</body>
</html>`

// ─── Admin scan page ──────────────────────────────────────────────────────────

const scanHTMLPage = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Сканер QR — MGOPROF</title>
<script src="https://cdn.jsdelivr.net/npm/jsqr@1.4.0/dist/jsQR.js"></script>
<style>
:root{--o:#ff7c2c;--g:#009b35;--bg:#0f172a;--card:#1e293b;--text:#f1f5f9;--muted:#94a3b8;
  --grad:linear-gradient(135deg,#ff7c2c,#009b35)}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);
  min-height:100vh;display:flex;flex-direction:column;align-items:center;padding:20px;gap:16px}
h1{font-size:20px;font-weight:700;margin-top:8px}
.cam-wrap{position:relative;width:100%;max-width:380px;border-radius:18px;overflow:hidden;
  box-shadow:0 8px 32px rgba(0,0,0,.4)}
video{width:100%;display:block}
.scan-line{position:absolute;left:0;right:0;height:2px;background:var(--grad);
  animation:scan 2s linear infinite;box-shadow:0 0 8px rgba(255,124,44,.8)}
@keyframes scan{0%{top:10%}50%{top:85%}100%{top:10%}}
.manual{display:flex;gap:8px;width:100%;max-width:380px}
.manual input{flex:1;padding:12px 16px;border-radius:10px;border:1px solid #334155;
  background:#1e293b;color:#f1f5f9;font-size:16px;font-family:monospace;letter-spacing:1px}
.manual input:focus{outline:none;border-color:var(--o)}
.manual button,.btn-cam{padding:12px 20px;border-radius:10px;border:none;background:var(--grad);
  color:#fff;font-size:14px;font-weight:600;cursor:pointer;white-space:nowrap}
.btn-cam{width:100%;max-width:380px}
.result{width:100%;max-width:380px;border-radius:14px;padding:18px 20px;
  background:var(--card);border:1px solid #334155;display:none}
.result.ok{border-color:#22c55e;background:#052e16}
.result.err{border-color:#ef4444;background:#2d0a0a}
.result.dup{border-color:#f59e0b;background:#1c1200}
.result-status{font-size:16px;font-weight:700;margin-bottom:10px}
.result-name{font-size:20px;font-weight:800;margin-bottom:4px}
.result-sub{font-size:13px;color:var(--muted)}
.history{width:100%;max-width:380px}
.history h2{font-size:13px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;margin-bottom:8px}
.history-list{display:flex;flex-direction:column;gap:6px}
.history-item{background:var(--card);border-radius:10px;padding:12px 14px;font-size:13px;
  border-left:3px solid #334155}
.history-item.ok{border-left-color:#22c55e}
.history-item.dup{border-left-color:#f59e0b}
.history-item.err{border-left-color:#ef4444}
</style>
</head>
<body>
<h1>Сканер билетов</h1>

<button class="btn-cam" id="btnCam">📷 Включить камеру</button>

<div class="cam-wrap" id="camWrap" style="display:none">
  <video id="video" playsinline autoplay muted></video>
  <div class="scan-line"></div>
  <canvas id="canvas" style="display:none"></canvas>
</div>

<div class="manual">
  <input id="tokenInput" type="text" placeholder="Код участника" maxlength="20"
    autocomplete="off" autocorrect="off" spellcheck="false">
  <button onclick="submitToken()">✓</button>
</div>

<div class="result" id="result"></div>

<div class="history">
  <h2>Последние отметки</h2>
  <div class="history-list" id="historyList"></div>
</div>

<script>
var scanning = false;
var lastScanned = '';
var lastTime = 0;
var history = [];

// Camera
document.getElementById('btnCam').addEventListener('click', function(){
  startCamera();
});

function startCamera(){
  navigator.mediaDevices.getUserMedia({video:{facingMode:'environment'}})
    .then(function(stream){
      var video = document.getElementById('video');
      video.srcObject = stream;
      video.play();
      document.getElementById('camWrap').style.display = 'block';
      document.getElementById('btnCam').style.display = 'none';
      scanning = true;
      requestAnimationFrame(tick);
    })
    .catch(function(e){
      alert('Камера недоступна: ' + e.message);
    });
}

function tick(){
  if(!scanning) return;
  var video = document.getElementById('video');
  var canvas = document.getElementById('canvas');
  if(video.readyState === video.HAVE_ENOUGH_DATA){
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    var ctx = canvas.getContext('2d');
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    var img = ctx.getImageData(0, 0, canvas.width, canvas.height);
    var code = jsQR(img.data, img.width, img.height, {inversionAttempts:'dontInvert'});
    if(code){
      var now = Date.now();
      if(code.data !== lastScanned || now - lastTime > 3000){
        lastScanned = code.data;
        lastTime = now;
        var token = extractToken(code.data);
        checkin(token);
      }
    }
  }
  requestAnimationFrame(tick);
}

function extractToken(raw){
  // If it's a full URL extract last path segment
  try {
    var url = new URL(raw);
    var parts = url.pathname.split('/').filter(Boolean);
    return parts[parts.length - 1] || raw;
  } catch(e){
    return raw.trim();
  }
}

// Manual input
document.getElementById('tokenInput').addEventListener('keydown', function(e){
  if(e.key === 'Enter') submitToken();
});
function submitToken(){
  var v = document.getElementById('tokenInput').value.trim();
  if(v) checkin(v);
}

function checkin(token){
  if(!token) return;
  var jwt = (document.cookie.match(/jwt=([^;]+)/)||[])[1] || '';
  fetch('/api/admin/checkin', {
    method:'POST',
    headers:{'Content-Type':'application/json','Authorization':'Bearer '+jwt},
    body:JSON.stringify({token:token})
  })
  .then(function(r){ return r.json().then(function(d){ return {ok:r.ok, data:d}; }); })
  .then(function(res){
    showResult(res.ok ? res.data : null, res.data, token);
    document.getElementById('tokenInput').value = '';
  })
  .catch(function(e){ showError(token, e.message); });
}

function showResult(data, raw, token){
  var box = document.getElementById('result');
  box.style.display = 'block';
  if(!data || !data.registration){
    box.className = 'result err';
    box.innerHTML = '<div class="result-status">❌ Не найдено</div><div class="result-sub">Токен: '+escH(token)+'</div>';
    addHistory('err', '❌ Не найдено', token);
    return;
  }
  var u = data.user, e = data.event;
  var name = u.last_name+' '+u.first_name+' '+u.patronymic;
  if(data.status === 'already'){
    box.className = 'result dup';
    box.innerHTML = '<div class="result-status">⚠️ Уже отмечен</div>'
      +'<div class="result-name">'+escH(name)+'</div>'
      +'<div class="result-sub">'+escH(e.title)+'</div>'
      +'<div class="result-sub">Первый вход: '+escH(data.checked_in_at)+'</div>';
    addHistory('dup', '⚠️ '+name, e.title);
  } else {
    box.className = 'result ok';
    box.innerHTML = '<div class="result-status">✅ Вход разрешён</div>'
      +'<div class="result-name">'+escH(name)+'</div>'
      +'<div class="result-sub">'+escH(e.title)+' · '+escH(u.organization)+'</div>';
    addHistory('ok', '✅ '+name, e.title);
  }
}

function showError(token, msg){
  var box = document.getElementById('result');
  box.style.display = 'block';
  box.className = 'result err';
  box.innerHTML = '<div class="result-status">❌ Ошибка</div><div class="result-sub">'+escH(msg)+'</div>';
  addHistory('err', '❌ Ошибка', token);
}

function addHistory(cls, title, sub){
  history.unshift({cls:cls, title:title, sub:sub, time:new Date().toLocaleTimeString('ru')});
  if(history.length > 10) history.pop();
  var list = document.getElementById('historyList');
  list.innerHTML = history.map(function(h){
    return '<div class="history-item '+h.cls+'">'
      +'<b>'+escH(h.title)+'</b> — '+escH(h.sub)
      +' <span style="color:#64748b;font-size:11px">'+h.time+'</span></div>';
  }).join('');
}

function escH(s){ return String(s||'').replace(/[&<>"']/g,function(c){
  return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];
}); }
</script>
</body>
</html>`
