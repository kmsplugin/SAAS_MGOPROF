package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// ReportHandler provides per-event analytics endpoints.
type ReportHandler struct {
	svc    *service.ReportService
	logger *zap.Logger
	tmpl   *template.Template
}

func NewReportHandler(svc *service.ReportService, logger *zap.Logger) *ReportHandler {
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"pct": func(part, total int) string {
			if total == 0 {
				return "0.0"
			}
			return fmt.Sprintf("%.1f", float64(part)*100/float64(total))
		},
		"fmtDate": func(t time.Time) string { return t.Format("02.01.2006") },
		"fmtTime": func(t time.Time) string { return t.Format("15:04") },
		"jsonMarshal": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"add": func(a, b int) int { return a + b },
	}).Parse(reportHTMLTemplate))
	return &ReportHandler{svc: svc, logger: logger, tmpl: tmpl}
}

func (h *ReportHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	grp := r.Group("/admin/events/:id", auth, middleware.RequireRole("admin"))
	grp.GET("/report", h.HTMLReport)
	grp.GET("/report.json", h.JSONReport)
	grp.GET("/export", h.CSVExport)
}

// HTMLReport renders the beautiful HTML report page.
func (h *ReportHandler) HTMLReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	full, err := h.svc.GetEventReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(c.Writer, full); err != nil {
		h.logger.Error("report template execute", zap.Error(err))
	}
}

// JSONReport returns the raw report data as JSON (for SPA front-ends).
func (h *ReportHandler) JSONReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	full, err := h.svc.GetEventReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"event":        full.Event,
		"report":       full.Report,
		"generated_at": full.GeneratedAt,
	})
}

// CSVExport streams a per-event CSV file.
func (h *ReportHandler) CSVExport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	event, rows, err := h.svc.GetEventExport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	slug := sanitizeFilename(event.Title)
	filename := fmt.Sprintf("reg_%s_%s.csv", slug, time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write([]byte("\xEF\xBB\xBF")) // UTF-8 BOM for Excel

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"Статус", "Дата регистрации",
		"Фамилия", "Имя", "Отчество",
		"Организация", "Округ", "Email",
		"Член профсоюза", "Номер билета", "Примечание",
		"IP", "Страна", "Регион", "Город",
		"Провайдер (ISP)", "ASN",
		"Устройство", "ОС", "Браузер",
	})
	for _, r := range rows {
		unionMember := "Нет"
		if r.IsUnionMember {
			unionMember = "Да"
		}
		statusRu := "Ожидает OTP"
		if r.Status == "verified" {
			statusRu = "Подтверждено"
		}
		_ = w.Write([]string{
			statusRu,
			r.RegDatetime.Format("02.01.2006 15:04"),
			r.LastName, r.FirstName, r.Patronymic,
			r.Organization, r.District, r.Email,
			unionMember, r.UnionTicket, r.ExtraInfo,
			r.IPAddress, r.GeoCountry, r.GeoRegion, r.GeoCity,
			r.ISPName, r.ISPASN,
			r.DeviceType, r.OSName, r.BrowserName,
		})
	}
	w.Flush()
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '_':
			b.WriteRune('_')
		}
	}
	name := b.String()
	if len(name) > 40 {
		name = name[:40]
	}
	if name == "" {
		name = "event"
	}
	return name
}

// ─── HTML template ────────────────────────────────────────────────────────────

const reportHTMLTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Отчёт: {{.Event.Title}}</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.3/dist/chart.umd.min.js"></script>
<style>
:root{
  --o:#ff7c2c;--g:#009b35;--bg:#f0f4f8;--card:#fff;
  --text:#1e293b;--muted:#64748b;--border:#e2e8f0;
  --grad:linear-gradient(135deg,#ff7c2c,#009b35);
  --shadow:0 4px 24px rgba(0,0,0,.07);
  --shadow-lg:0 12px 40px rgba(0,0,0,.12);
  --radius:18px;
}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);min-height:100vh}

/* ── Header ── */
.header{background:linear-gradient(135deg,#0f172a 60%,#1e3a2f);color:#fff;padding:28px 32px 24px;position:relative;overflow:hidden}
.header::before{content:'';position:absolute;right:-60px;top:-60px;width:280px;height:280px;
  border-radius:50%;background:rgba(255,124,44,.12);pointer-events:none}
.header::after{content:'';position:absolute;right:60px;bottom:-80px;width:200px;height:200px;
  border-radius:50%;background:rgba(0,155,53,.1);pointer-events:none}
.header-inner{max-width:1280px;margin:0 auto;position:relative;z-index:1}
.header h1{font-size:clamp(18px,3vw,26px);font-weight:700;margin-bottom:6px;letter-spacing:-.3px}
.header-sub{font-size:14px;color:rgba(255,255,255,.65);display:flex;gap:16px;flex-wrap:wrap;align-items:center}
.header-badge{background:rgba(255,255,255,.1);border:1px solid rgba(255,255,255,.15);
  padding:3px 10px;border-radius:20px;font-size:12px;white-space:nowrap}
.header-actions{display:flex;gap:10px;margin-top:16px;flex-wrap:wrap}
.btn{display:inline-flex;align-items:center;gap:6px;padding:9px 18px;border-radius:10px;
  font-size:13px;font-weight:600;cursor:pointer;text-decoration:none;border:none;
  transition:transform .15s,box-shadow .15s}
.btn:hover{transform:translateY(-1px);box-shadow:0 6px 20px rgba(0,0,0,.18)}
.btn-primary{background:var(--grad);color:#fff}
.btn-ghost{background:rgba(255,255,255,.12);color:#fff;border:1px solid rgba(255,255,255,.2)}

/* ── Layout ── */
.main{max-width:1280px;margin:0 auto;padding:28px 20px}

/* ── Stat cards ── */
.stat-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:24px}
@media(max-width:900px){.stat-grid{grid-template-columns:repeat(2,1fr)}}
@media(max-width:500px){.stat-grid{grid-template-columns:1fr}}

.stat-card{background:var(--card);border-radius:var(--radius);padding:22px 24px;
  box-shadow:var(--shadow);position:relative;overflow:hidden;
  animation:fadeUp .4s ease both}
.stat-card::before{content:'';position:absolute;top:0;left:0;right:0;height:3px;background:var(--grad)}
.stat-num{font-size:clamp(28px,4vw,40px);font-weight:800;letter-spacing:-1px;
  background:var(--grad);-webkit-background-clip:text;-webkit-text-fill-color:transparent;
  background-clip:text;line-height:1}
.stat-label{font-size:12px;color:var(--muted);margin-top:4px;font-weight:500;text-transform:uppercase;letter-spacing:.5px}
.stat-sub{font-size:13px;color:var(--o);font-weight:600;margin-top:6px}
.stat-card:nth-child(1){animation-delay:.05s}
.stat-card:nth-child(2){animation-delay:.1s}
.stat-card:nth-child(3){animation-delay:.15s}
.stat-card:nth-child(4){animation-delay:.2s}

/* ── Row grids ── */
.row-2{display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:24px}
.row-3-1{display:grid;grid-template-columns:3fr 2fr;gap:16px;margin-bottom:24px}
@media(max-width:860px){.row-2,.row-3-1{grid-template-columns:1fr}}
@media(max-width:860px){.row-3{grid-template-columns:1fr!important}}

/* ── Card ── */
.card{background:var(--card);border-radius:var(--radius);padding:24px;box-shadow:var(--shadow);animation:fadeUp .4s ease both}
.card-title{font-size:14px;font-weight:700;text-transform:uppercase;letter-spacing:.5px;
  color:var(--muted);margin-bottom:18px;display:flex;align-items:center;gap:8px}
.card-title::before{content:'';display:block;width:12px;height:12px;border-radius:3px;background:var(--grad)}

/* ── District bars ── */
.district-list{display:flex;flex-direction:column;gap:10px}
.district-row{display:grid;grid-template-columns:90px 1fr 48px;align-items:center;gap:10px;font-size:13px}
.district-name{font-weight:600;color:var(--text);text-align:right;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.bar-track{background:#f1f5f9;border-radius:6px;height:20px;overflow:hidden;position:relative}
.bar-fill{height:100%;border-radius:6px;background:var(--grad);position:relative;
  transition:width .8s cubic-bezier(.4,0,.2,1);min-width:4px}
.bar-fill::after{content:attr(data-v);position:absolute;right:6px;top:50%;transform:translateY(-50%);
  font-size:11px;color:#fff;font-weight:700;white-space:nowrap}
.bar-total{font-size:13px;font-weight:700;color:var(--text);text-align:right}

/* ── Donut charts ── */
.donut-wrap{display:flex;flex-direction:column;gap:20px}
.donut-card{flex:1}
.donut-inner{display:flex;gap:20px;align-items:center}
.donut-chart-wrap{position:relative;width:120px;height:120px;flex-shrink:0}
.donut-center{position:absolute;inset:0;display:flex;flex-direction:column;
  align-items:center;justify-content:center;pointer-events:none}
.donut-center-num{font-size:22px;font-weight:800;color:var(--text)}
.donut-center-label{font-size:10px;color:var(--muted);text-align:center;line-height:1.2}
.legend{display:flex;flex-direction:column;gap:8px;flex:1}
.legend-item{display:flex;align-items:center;gap:8px;font-size:13px}
.legend-dot{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.legend-val{font-weight:700;margin-left:auto}
.legend-pct{color:var(--muted);font-size:11px}

/* ── Timeline chart ── */
.timeline-card{margin-bottom:24px}
.chart-container{position:relative;height:220px}

/* ── Org list ── */
.org-list{display:flex;flex-direction:column;gap:8px}
.org-row{display:flex;align-items:center;gap:10px;font-size:13px}
.org-rank{width:22px;height:22px;border-radius:6px;background:var(--bg);
  display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:700;color:var(--muted);flex-shrink:0}
.org-name{flex:1;font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.org-bar{flex:0 0 80px;background:#f1f5f9;border-radius:4px;height:8px;overflow:hidden}
.org-fill{height:100%;border-radius:4px;background:var(--grad)}
.org-cnt{font-weight:700;font-size:13px;width:30px;text-align:right;flex-shrink:0}

/* ── Footer ── */
.report-footer{text-align:center;font-size:12px;color:var(--muted);padding:20px 0 36px}

/* ── Animations ── */
@keyframes fadeUp{from{opacity:0;transform:translateY(16px)}to{opacity:1;transform:translateY(0)}}
@keyframes countUp{from{opacity:0}to{opacity:1}}

/* ── Print ── */
@media print{
  .header-actions,.btn{display:none!important}
  body{background:#fff}
  .card,.stat-card{box-shadow:none;border:1px solid #ddd}
  .bar-fill,.org-fill{print-color-adjust:exact;-webkit-print-color-adjust:exact}
}
</style>
</head>
<body>

<div class="header">
  <div class="header-inner">
    <h1>Отчёт: {{.Event.Title}}</h1>
    <div class="header-sub">
      <span>{{fmtDate .GeneratedAt}} · Сформирован в {{fmtTime .GeneratedAt}}</span>
      <span class="header-badge">Московская городская организация профсоюза образования</span>
      {{if .Event.EventDate}}<span class="header-badge">Мероприятие: {{.Event.EventDate}}</span>{{end}}
    </div>
    <div class="header-actions">
      <a class="btn btn-primary" href="export">⬇ Скачать CSV</a>
      <button class="btn btn-ghost" onclick="window.print()">🖨 Печать</button>
      <a class="btn btn-ghost" href="/api/admin/events">← К мероприятиям</a>
    </div>
  </div>
</div>

<div class="main">

  <!-- Stat cards -->
  <div class="stat-grid">
    <div class="stat-card">
      <div class="stat-num" id="n-total">{{.Report.Stats.TotalRegs}}</div>
      <div class="stat-label">Зарегистрировано</div>
      <div class="stat-sub">подключений (форма)</div>
    </div>
    <div class="stat-card">
      <div class="stat-num" id="n-verified">{{.Report.Stats.VerifiedRegs}}</div>
      <div class="stat-label">Верифицировано</div>
      <div class="stat-sub">{{pct .Report.Stats.VerifiedRegs .Report.Stats.TotalRegs}}% подтверждений</div>
    </div>
    <div class="stat-card">
      <div class="stat-num" id="n-union">{{.Report.Stats.UnionMembers}}</div>
      <div class="stat-label">Членов профсоюза</div>
      <div class="stat-sub">{{pct .Report.Stats.UnionMembers .Report.Stats.VerifiedRegs}}% от верифицированных</div>
    </div>
    <div class="stat-card">
      <div class="stat-num" id="n-pending">{{.Report.Stats.PendingRegs}}</div>
      <div class="stat-label">Ожидают OTP</div>
      <div class="stat-sub">{{pct .Report.Stats.PendingRegs .Report.Stats.TotalRegs}}% незавершённых</div>
    </div>
  </div>

  <!-- Districts + Donuts -->
  <div class="row-3-1">

    <div class="card">
      <div class="card-title">Участники по округам и регионам</div>
      <div class="district-list">
        {{$max := 1}}
        {{range .Report.Districts}}{{if gt .Total $max}}{{$max = .Total}}{{end}}{{end}}
        {{range .Report.Districts}}
        <div class="district-row">
          <span class="district-name" title="{{.District}}">{{.District}}</span>
          <div class="bar-track">
            <div class="bar-fill"
              data-v="{{.Verified}}"
              style="width:{{pct .Total $max}}%"></div>
          </div>
          <span class="bar-total">{{.Total}}</span>
        </div>
        {{end}}
        {{if not .Report.Districts}}<div style="color:var(--muted);font-size:14px">Нет данных</div>{{end}}
      </div>
    </div>

    <div class="card">
      <div class="card-title">Статусы</div>
      <div class="donut-wrap">

        <div class="donut-card">
          <div style="font-size:12px;font-weight:700;color:var(--muted);margin-bottom:12px">СТАТУС РЕГИСТРАЦИИ</div>
          <div class="donut-inner">
            <div class="donut-chart-wrap">
              <canvas id="chartStatus"></canvas>
              <div class="donut-center">
                <span class="donut-center-num">{{.Report.Stats.VerifiedRegs}}</span>
                <span class="donut-center-label">verified</span>
              </div>
            </div>
            <div class="legend">
              <div class="legend-item">
                <span class="legend-dot" style="background:#009b35"></span>
                <span>Verified</span>
                <span class="legend-val">{{.Report.Stats.VerifiedRegs}}</span>
                <span class="legend-pct">{{pct .Report.Stats.VerifiedRegs .Report.Stats.TotalRegs}}%</span>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background:#ff7c2c"></span>
                <span>Pending</span>
                <span class="legend-val">{{.Report.Stats.PendingRegs}}</span>
                <span class="legend-pct">{{pct .Report.Stats.PendingRegs .Report.Stats.TotalRegs}}%</span>
              </div>
            </div>
          </div>
        </div>

        <div class="donut-card">
          <div style="font-size:12px;font-weight:700;color:var(--muted);margin-bottom:12px">ЧЛЕНСТВО В ПРОФСОЮЗЕ</div>
          <div class="donut-inner">
            <div class="donut-chart-wrap">
              <canvas id="chartUnion"></canvas>
              <div class="donut-center">
                <span class="donut-center-num">{{.Report.Stats.UnionMembers}}</span>
                <span class="donut-center-label">членов</span>
              </div>
            </div>
            <div class="legend">
              <div class="legend-item">
                <span class="legend-dot" style="background:#009b35"></span>
                <span>Да</span>
                <span class="legend-val">{{.Report.Stats.UnionMembers}}</span>
                <span class="legend-pct">{{pct .Report.Stats.UnionMembers .Report.Stats.VerifiedRegs}}%</span>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background:#e2e8f0"></span>
                <span>Нет</span>
                <span class="legend-val">{{.Report.Stats.NonUnion}}</span>
                <span class="legend-pct">{{pct .Report.Stats.NonUnion .Report.Stats.VerifiedRegs}}%</span>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>

  <!-- Timeline -->
  <div class="card timeline-card">
    <div class="card-title">Динамика регистраций (5-минутные интервалы)</div>
    <div class="chart-container">
      <canvas id="chartTimeline"></canvas>
    </div>
  </div>

  <!-- Orgs -->
  {{if .Report.Orgs}}
  <div class="card" style="margin-bottom:24px;animation-delay:.3s">
    <div class="card-title">Топ организаций</div>
    <div class="org-list">
      {{$maxOrg := 1}}
      {{range .Report.Orgs}}{{if gt .Total $maxOrg}}{{$maxOrg = .Total}}{{end}}{{end}}
      {{range $i,$o := .Report.Orgs}}
      <div class="org-row">
        <span class="org-rank">{{add $i 1}}</span>
        <span class="org-name" title="{{$o.Organization}}">{{$o.Organization}}</span>
        <div class="org-bar"><div class="org-fill" style="width:{{pct $o.Total $maxOrg}}%"></div></div>
        <span class="org-cnt">{{$o.Total}}</span>
      </div>
      {{end}}
    </div>
  </div>
  {{end}}

  <!-- Device / OS / Browser breakdown -->
  <div class="row-3" style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:16px;margin-bottom:24px">

    <div class="card" style="animation-delay:.35s">
      <div class="card-title">Устройства</div>
      {{if .Report.Devices}}
      <div class="org-list">
        {{$maxD := 1}}{{range .Report.Devices}}{{if gt .Total $maxD}}{{$maxD = .Total}}{{end}}{{end}}
        {{range .Report.Devices}}
        <div class="org-row">
          <span class="org-name">{{.Name}}</span>
          <div class="org-bar"><div class="org-fill" style="width:{{pct .Total $maxD}}%"></div></div>
          <span class="org-cnt">{{.Total}}</span>
        </div>
        {{end}}
      </div>
      {{else}}<div style="color:var(--muted);font-size:13px">Нет данных</div>{{end}}
    </div>

    <div class="card" style="animation-delay:.4s">
      <div class="card-title">Операционные системы</div>
      {{if .Report.OSes}}
      <div class="org-list">
        {{$maxO := 1}}{{range .Report.OSes}}{{if gt .Total $maxO}}{{$maxO = .Total}}{{end}}{{end}}
        {{range .Report.OSes}}
        <div class="org-row">
          <span class="org-name">{{.Name}}</span>
          <div class="org-bar"><div class="org-fill" style="width:{{pct .Total $maxO}}%"></div></div>
          <span class="org-cnt">{{.Total}}</span>
        </div>
        {{end}}
      </div>
      {{else}}<div style="color:var(--muted);font-size:13px">Нет данных</div>{{end}}
    </div>

    <div class="card" style="animation-delay:.45s">
      <div class="card-title">Браузеры</div>
      {{if .Report.Browsers}}
      <div class="org-list">
        {{$maxB := 1}}{{range .Report.Browsers}}{{if gt .Total $maxB}}{{$maxB = .Total}}{{end}}{{end}}
        {{range .Report.Browsers}}
        <div class="org-row">
          <span class="org-name">{{.Name}}</span>
          <div class="org-bar"><div class="org-fill" style="width:{{pct .Total $maxB}}%"></div></div>
          <span class="org-cnt">{{.Total}}</span>
        </div>
        {{end}}
      </div>
      {{else}}<div style="color:var(--muted);font-size:13px">Нет данных</div>{{end}}
    </div>

  </div>

</div><!-- /main -->

<div class="report-footer">
  Отчёт сформирован автоматически · {{fmtDate .GeneratedAt}} {{fmtTime .GeneratedAt}} · MGOPROF
</div>

<script>
(function(){
'use strict';

// Animate stat numbers
function animateCounter(el){
  var target = parseInt(el.textContent, 10) || 0;
  if(!target) return;
  var start = 0, dur = 900, step = 16;
  var timer = setInterval(function(){
    start += Math.ceil(target / (dur / step));
    if(start >= target){ start = target; clearInterval(timer); }
    el.textContent = start.toLocaleString('ru');
  }, step);
}
['n-total','n-verified','n-union','n-pending'].forEach(function(id){
  var el = document.getElementById(id);
  if(el) animateCounter(el);
});

// ── Donut: Status ──
new Chart(document.getElementById('chartStatus'), {
  type: 'doughnut',
  data: {
    labels: ['Verified','Pending'],
    datasets:[{
      data: [{{.Report.Stats.VerifiedRegs}}, {{.Report.Stats.PendingRegs}}],
      backgroundColor: ['#009b35','#ff7c2c'],
      borderWidth: 0,
      hoverOffset: 4
    }]
  },
  options:{
    cutout:'72%', responsive:true, maintainAspectRatio:true,
    plugins:{legend:{display:false},tooltip:{
      callbacks:{label:function(ctx){return ' '+ctx.label+': '+ctx.parsed;}}
    }},
    animation:{animateRotate:true,duration:900}
  }
});

// ── Donut: Union ──
new Chart(document.getElementById('chartUnion'), {
  type: 'doughnut',
  data: {
    labels: ['Да','Нет'],
    datasets:[{
      data: [{{.Report.Stats.UnionMembers}}, {{.Report.Stats.NonUnion}}],
      backgroundColor: ['#009b35','#e2e8f0'],
      borderWidth: 0,
      hoverOffset: 4
    }]
  },
  options:{
    cutout:'72%', responsive:true, maintainAspectRatio:true,
    plugins:{legend:{display:false},tooltip:{
      callbacks:{label:function(ctx){return ' '+ctx.label+': '+ctx.parsed;}}
    }},
    animation:{animateRotate:true,duration:900,delay:200}
  }
});

// ── Timeline ──
var tl = {{jsonMarshal .Report.Timeline}};
if(tl && tl.length > 0){
  var labels = tl.map(function(p){
    var d = new Date(p.bucket);
    return d.getHours().toString().padStart(2,'0')+':'+d.getMinutes().toString().padStart(2,'0');
  });
  var totals   = tl.map(function(p){ return p.count; });
  var verified = tl.map(function(p){ return p.verified; });

  new Chart(document.getElementById('chartTimeline'), {
    type: 'bar',
    data:{
      labels: labels,
      datasets:[
        {
          label:'Всего',
          data: totals,
          backgroundColor:'rgba(255,124,44,.75)',
          borderRadius:4,
          order:2
        },
        {
          label:'Верифицировано',
          data: verified,
          backgroundColor:'rgba(0,155,53,.85)',
          borderRadius:4,
          order:1
        }
      ]
    },
    options:{
      responsive:true,maintainAspectRatio:false,
      interaction:{mode:'index',intersect:false},
      plugins:{
        legend:{position:'top',labels:{boxWidth:12,font:{size:12}}},
        tooltip:{callbacks:{title:function(items){return 'Время: '+items[0].label;}}}
      },
      scales:{
        x:{grid:{display:false},ticks:{maxRotation:45,font:{size:11}}},
        y:{beginAtZero:true,grid:{color:'#f1f5f9'},ticks:{precision:0,font:{size:11}},
           title:{display:true,text:'чел.',font:{size:11}}}
      },
      animation:{duration:800}
    }
  });
} else {
  var c = document.getElementById('chartTimeline');
  c.parentElement.innerHTML = '<div style="height:220px;display:flex;align-items:center;justify-content:center;color:#94a3b8;font-size:14px">Нет данных по временно́й динамике</div>';
}

})();
</script>
</body>
</html>`
