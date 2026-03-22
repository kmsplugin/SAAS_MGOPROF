package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// ── Template helpers ──────────────────────────────────────────────────────────

var adminTmplOnce sync.Once
var adminTmpl *template.Template

var adminFuncs = template.FuncMap{
	"fmtTime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("02.01.2006 15:04")
	},
	"fmtTimePtr": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return "—"
		}
		return t.Format("02.01.2006 15:04")
	},
	"boolRu": func(b bool) string {
		if b {
			return "Да"
		}
		return "Нет"
	},
	"add": func(a, b int) int { return a + b },
	"statusBadge": func(s string) template.HTML {
		cls := map[string]string{
			"verified":  "bg-green-100 text-green-800",
			"pending":   "bg-yellow-100 text-yellow-800",
			"cancelled": "bg-red-100 text-red-800",
			"online":    "bg-blue-100 text-blue-800",
			"offline":   "bg-gray-100 text-gray-700",
			"hybrid":    "bg-purple-100 text-purple-800",
		}[s]
		if cls == "" {
			cls = "bg-gray-100 text-gray-700"
		}
		return template.HTML(fmt.Sprintf(
			`<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium %s">%s</span>`,
			cls, template.HTMLEscapeString(s),
		))
	},
	"navClass": func(current, target string) string {
		base := "flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium transition-colors "
		if current == target {
			return base + "bg-gray-700 text-white"
		}
		return base + "text-gray-300 hover:bg-gray-700 hover:text-white"
	},
}

func getAdminTmpl() *template.Template {
	adminTmplOnce.Do(func() {
		adminTmpl = template.Must(
			template.New("").Funcs(adminFuncs).Parse(adminTemplatesSrc),
		)
	})
	return adminTmpl
}

// renderAdminPage executes the named template with data and writes the result.
func renderAdminPage(w http.ResponseWriter, page string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := getAdminTmpl().ExecuteTemplate(w, page, data); err != nil {
		http.Error(w, fmt.Sprintf("render error [%s]: %s", page, err), http.StatusInternalServerError)
	}
}

// ── All admin HTML templates ──────────────────────────────────────────────────

const adminTemplatesSrc = `

{{/* ── Shared navigation partial ───────────────────────────────────────── */}}
{{define "nav"}}
<aside class="fixed inset-y-0 left-0 w-60 bg-gray-900 flex flex-col z-10">
  <div class="flex items-center gap-2 px-4 py-5 border-b border-gray-700">
    <span class="text-white font-bold text-lg">WorkOS</span>
    <span class="text-gray-400 text-sm">Admin</span>
  </div>
  <nav class="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
    <a href="/panel"              class="{{navClass . "dashboard"}}">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/></svg>
      Дашборд
    </a>
    <a href="/panel/events"       class="{{navClass . "events"}}">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
      Мероприятия
    </a>
    <a href="/panel/registrations" class="{{navClass . "registrations"}}">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
      Регистрации
    </a>
    <a href="/panel/reflists"     class="{{navClass . "reflists"}}">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/></svg>
      Справочники
    </a>
    <a href="/panel/logs"         class="{{navClass . "logs"}}">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>
      Логи
    </a>
  </nav>
  <div class="px-4 py-3 border-t border-gray-700 text-xs text-gray-500">WorkOS v1.0</div>
</aside>
{{end}}

{{/* ── Page wrapper ──────────────────────────────────────────────────────── */}}
{{define "head"}}
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}} — WorkOS Admin</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; }
    .table-auto th { background: #f8fafc; font-size:.75rem; letter-spacing:.05em; text-transform:uppercase; color:#64748b; }
    .table-auto td, .table-auto th { padding:.625rem 1rem; white-space:nowrap; }
    .table-auto tr:hover td { background:#f1f5f9; }
  </style>
</head>
<body class="bg-gray-50 text-gray-900">
{{end}}

{{/* ── Dashboard ──────────────────────────────────────────────────────────── */}}
{{define "dashboard"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-semibold text-gray-800">Дашборд</h1>
    <span class="text-sm text-gray-500">{{fmtTime .Now}}</span>
  </div>

  {{/* Stats */}}
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
    <div class="bg-white rounded-xl shadow-sm border p-5">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Пользователей</p>
      <p class="text-3xl font-bold text-gray-800">{{.Stats.TotalUsers}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Подтверждено</p>
      <p class="text-3xl font-bold text-green-600">{{.Stats.VerifiedRegs}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Ожидают подтв.</p>
      <p class="text-3xl font-bold text-yellow-500">{{.Stats.PendingRegs}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Активных событий</p>
      <p class="text-3xl font-bold text-blue-600">{{.Stats.ActiveEvents}}</p>
    </div>
  </div>

  {{/* Events */}}
  <div class="bg-white rounded-xl shadow-sm border mb-8">
    <div class="flex items-center justify-between px-5 py-4 border-b">
      <h2 class="font-semibold text-gray-700">Мероприятия</h2>
      <a href="/panel/events/new" class="text-sm text-blue-600 hover:underline">+ Создать</a>
    </div>
    <div class="overflow-x-auto">
      <table class="table-auto w-full text-sm border-collapse">
        <thead><tr>
          <th class="text-left">Название</th>
          <th class="text-left">Дата</th>
          <th class="text-left">Тип</th>
          <th class="text-right">Всего</th>
          <th class="text-right">Подтв.</th>
          <th class="text-left">Действия</th>
        </tr></thead>
        <tbody>
        {{range .Events}}
        <tr class="border-t">
          <td class="font-medium">{{.Title}}</td>
          <td class="text-gray-500">{{.EventDate}}</td>
          <td>{{statusBadge .EventType}}</td>
          <td class="text-right">{{.TotalRegs}}</td>
          <td class="text-right text-green-600 font-medium">{{.VerifiedRegs}}</td>
          <td>
            <div class="flex gap-2">
              <a href="/panel/events/{{.ID}}/edit"          class="text-xs text-blue-600 hover:underline">Изменить</a>
              <a href="/panel/events/{{.ID}}/registrations" class="text-xs text-gray-600 hover:underline">Участники</a>
              <a href="/panel/events/{{.ID}}/tracking"      class="text-xs text-gray-600 hover:underline">Трекинг</a>
            </div>
          </td>
        </tr>
        {{else}}<tr><td colspan="6" class="text-center text-gray-400 py-6">Мероприятий нет</td></tr>
        {{end}}
        </tbody>
      </table>
    </div>
  </div>

  {{/* Recent registrations */}}
  <div class="bg-white rounded-xl shadow-sm border">
    <div class="flex items-center justify-between px-5 py-4 border-b">
      <h2 class="font-semibold text-gray-700">Последние регистрации</h2>
      <a href="/panel/registrations" class="text-sm text-blue-600 hover:underline">Все →</a>
    </div>
    <div class="overflow-x-auto">
      <table class="table-auto w-full text-sm border-collapse">
        <thead><tr>
          <th class="text-left">Дата</th>
          <th class="text-left">Мероприятие</th>
          <th class="text-left">Участник</th>
          <th class="text-left">Email</th>
          <th class="text-left">Статус</th>
        </tr></thead>
        <tbody>
        {{range .Recent}}
        <tr class="border-t">
          <td class="text-gray-500">{{fmtTime .RegDatetime}}</td>
          <td>{{.EventTitle}}</td>
          <td>{{.LastName}} {{.FirstName}}</td>
          <td class="text-gray-500">{{.Email}}</td>
          <td>{{statusBadge .Status}}</td>
        </tr>
        {{else}}<tr><td colspan="5" class="text-center text-gray-400 py-6">Нет регистраций</td></tr>
        {{end}}
        </tbody>
      </table>
    </div>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Events list ─────────────────────────────────────────────────────────── */}}
{{define "events"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-semibold text-gray-800">Мероприятия</h1>
    <a href="/panel/events/new"
       class="inline-flex items-center gap-1 bg-blue-600 text-white text-sm px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
      + Создать мероприятие
    </a>
  </div>
  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">ID</th>
        <th class="text-left">Название</th>
        <th class="text-left">Дата</th>
        <th class="text-left">Тип</th>
        <th class="text-right">Всего</th>
        <th class="text-right">Подтв.</th>
        <th class="text-left">Активно</th>
        <th class="text-left">Действия</th>
      </tr></thead>
      <tbody>
      {{range .Events}}
      <tr class="border-t">
        <td class="text-gray-400">{{.ID}}</td>
        <td class="font-medium">{{.Title}}</td>
        <td class="text-gray-500">{{.EventDate}}</td>
        <td>{{statusBadge .EventType}}</td>
        <td class="text-right">{{.TotalRegs}}</td>
        <td class="text-right text-green-600 font-medium">{{.VerifiedRegs}}</td>
        <td>{{if .IsActive}}<span class="text-green-600 font-medium">Да</span>{{else}}<span class="text-gray-400">Нет</span>{{end}}</td>
        <td>
          <div class="flex flex-wrap gap-2">
            <a href="/panel/events/{{.ID}}/edit"          class="text-blue-600 hover:underline">Изменить</a>
            <a href="/panel/events/{{.ID}}/registrations" class="text-gray-600 hover:underline">Участники</a>
            <a href="/panel/events/{{.ID}}/fields"        class="text-gray-600 hover:underline">Поля</a>
            <a href="/panel/events/{{.ID}}/attendance"    class="text-gray-600 hover:underline">Присутствие</a>
            <a href="/panel/events/{{.ID}}/tracking"      class="text-gray-600 hover:underline">Трекинг</a>
          </div>
        </td>
      </tr>
      {{else}}<tr><td colspan="8" class="text-center text-gray-400 py-8">Мероприятий нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Event form (create / edit) ─────────────────────────────────────────── */}}
{{define "event_form"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8 max-w-3xl">
  <div class="mb-6">
    <a href="/panel/events" class="text-sm text-gray-500 hover:underline">← Все мероприятия</a>
    <h1 class="text-2xl font-semibold text-gray-800 mt-1">
      {{if .IsCreate}}Создать мероприятие{{else}}Редактировать: {{.Event.Title}}{{end}}
    </h1>
  </div>

  {{/* Error / success banner populated by JS */}}
  <div id="banner" class="hidden mb-4 p-3 rounded-lg text-sm"></div>

  <form id="eventForm" class="space-y-6" onsubmit="submitForm(event)">
    <div class="bg-white rounded-xl shadow-sm border p-6 space-y-4">
      <h2 class="font-medium text-gray-700 border-b pb-2">Основное</h2>

      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Название *</label>
        <input name="title" required value="{{if .Event}}{{.Event.Title}}{{end}}"
               class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Описание</label>
        <textarea name="description" rows="3"
                  class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">{{if .Event}}{{.Event.Description}}{{end}}</textarea>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Дата</label>
          <input name="event_date" type="date" value="{{if .Event}}{{.Event.EventDate}}{{end}}"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Время</label>
          <input name="event_time" type="time" value="{{if .Event}}{{.Event.EventTime}}{{end}}"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Тип мероприятия</label>
          <select name="event_type" class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
            <option value="offline" {{if .Event}}{{if eq .Event.EventType "offline"}}selected{{end}}{{end}}>Офлайн</option>
            <option value="online"  {{if .Event}}{{if eq .Event.EventType "online"}} selected{{end}}{{end}}>Онлайн</option>
            <option value="hybrid"  {{if .Event}}{{if eq .Event.EventType "hybrid"}} selected{{end}}{{end}}>Гибрид</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Вместимость (0 = без лимита)</label>
          <input name="capacity" type="number" min="0" value="{{if .Event}}{{.Event.Capacity}}{{end}}"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Место проведения</label>
          <input name="venue" value="{{if .Event}}{{.Event.Venue}}{{end}}"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Адрес</label>
          <input name="address" value="{{if .Event}}{{.Event.Address}}{{end}}"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
      </div>

      <div class="flex items-center gap-6">
        <label class="flex items-center gap-2 text-sm text-gray-600 cursor-pointer">
          <input name="is_active" type="checkbox" {{if .Event}}{{if .Event.IsActive}}checked{{end}}{{end}}
                 class="rounded border-gray-300">
          Мероприятие активно
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-600 cursor-pointer">
          <input name="is_online" type="checkbox" {{if .Event}}{{if .Event.IsOnline}}checked{{end}}{{end}}
                 class="rounded border-gray-300">
          Онлайн-формат
        </label>
      </div>
    </div>

    <div class="bg-white rounded-xl shadow-sm border p-6 space-y-4">
      <h2 class="font-medium text-gray-700 border-b pb-2">Ссылки на трансляцию / кабинет</h2>
      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Ссылка для зрителей (viewer_link)</label>
        <input name="viewer_link" type="url" placeholder="https://..." value="{{if .Event}}{{.Event.ViewerLink}}{{end}}"
               class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        <p class="text-xs text-gray-400 mt-1">Показывается участникам с ролью delegate / observer / vip</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Ссылка для докладчиков (speaker_link)</label>
        <input name="speaker_link" type="url" placeholder="https://..." value="{{if .Event}}{{.Event.SpeakerLink}}{{end}}"
               class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        <p class="text-xs text-gray-400 mt-1">Показывается участникам с ролью speaker / moderator</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Ссылка в кабинете (cabinet_link, запасная)</label>
        <input name="cabinet_link" type="url" placeholder="https://..." value="{{if .Event}}{{.Event.CabinetLink}}{{end}}"
               class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
      </div>
    </div>

    <div class="bg-white rounded-xl shadow-sm border p-6 space-y-4">
      <h2 class="font-medium text-gray-700 border-b pb-2">Регистрация</h2>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Открыть регистрацию</label>
          <input name="registration_opens_at" type="datetime-local"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-600 mb-1">Закрыть регистрацию</label>
          <input name="registration_closes_at" type="datetime-local"
                 class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-600 mb-1">Режим чекина</label>
        <select name="check_in_mode" class="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
          <option value="none"        {{if .Event}}{{if eq .Event.CheckInMode "none"}}       selected{{end}}{{end}}>Нет</option>
          <option value="entry_only"  {{if .Event}}{{if eq .Event.CheckInMode "entry_only"}} selected{{end}}{{end}}>Только вход</option>
          <option value="entry_exit"  {{if .Event}}{{if eq .Event.CheckInMode "entry_exit"}} selected{{end}}{{end}}>Вход + выход</option>
        </select>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <button type="submit"
              class="bg-blue-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors">
        {{if .IsCreate}}Создать{{else}}Сохранить{{end}}
      </button>
      <a href="/panel/events" class="text-sm text-gray-500 hover:underline">Отмена</a>
    </div>
  </form>
</div>

<script>
const isCreate = {{.IsCreate}};
const eventID  = {{if .Event}}{{.Event.ID}}{{else}}0{{end}};

function showBanner(msg, ok) {
  const b = document.getElementById('banner');
  b.textContent = msg;
  b.className = 'mb-4 p-3 rounded-lg text-sm ' + (ok ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800');
  b.classList.remove('hidden');
  if (ok) setTimeout(() => window.location = '/panel/events', 1200);
}

async function submitForm(e) {
  e.preventDefault();
  const fd = new FormData(e.target);
  const body = {
    title:                    fd.get('title'),
    description:              fd.get('description') || '',
    event_date:               fd.get('event_date')  || '',
    event_time:               fd.get('event_time')  || '',
    venue:                    fd.get('venue')        || '',
    address:                  fd.get('address')      || '',
    capacity:                 parseInt(fd.get('capacity')) || 0,
    event_type:               fd.get('event_type'),
    check_in_mode:            fd.get('check_in_mode'),
    cabinet_link:             fd.get('cabinet_link')   || '',
    speaker_link:             fd.get('speaker_link')   || '',
    viewer_link:              fd.get('viewer_link')    || '',
    is_active:                fd.get('is_active') === 'on',
    is_online:                fd.get('is_online') === 'on',
    registration_opens_at:    fd.get('registration_opens_at')  || null,
    registration_closes_at:   fd.get('registration_closes_at') || null,
  };
  const token = localStorage.getItem('admin_token') || '';
  const url    = isCreate ? '/api/admin/events' : '/api/admin/events/' + eventID;
  const method = isCreate ? 'POST' : 'PUT';
  try {
    const r = await fetch(url, {
      method, headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token },
      body: JSON.stringify(body),
    });
    const d = await r.json();
    if (!r.ok) { showBanner(d.message || 'Ошибка', false); return; }
    showBanner('Сохранено!', true);
  } catch(err) {
    showBanner('Сетевая ошибка: ' + err, false);
  }
}
</script>
</body></html>
{{end}}

{{/* ── Registrations (global or per-event) ───────────────────────────────── */}}
{{define "registrations"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6 flex items-center justify-between">
    <div>
      {{if .EventTitle}}
        <a href="/panel/events" class="text-sm text-gray-500 hover:underline">← Мероприятия</a>
        <h1 class="text-2xl font-semibold text-gray-800 mt-1">Участники: {{.EventTitle}}</h1>
      {{else}}
        <h1 class="text-2xl font-semibold text-gray-800">Регистрации</h1>
      {{end}}
    </div>
    <div class="flex gap-3">
      <span class="text-sm text-gray-500 self-center">Всего: <b>{{.Total}}</b></span>
      {{if .EventTitle}}
        <a href="/api/admin/export?event_id={{.EventID}}" target="_blank"
           class="text-sm bg-green-600 text-white px-3 py-1.5 rounded-lg hover:bg-green-700">Экспорт CSV</a>
      {{else}}
        <a href="/api/admin/export" target="_blank"
           class="text-sm bg-green-600 text-white px-3 py-1.5 rounded-lg hover:bg-green-700">Экспорт CSV</a>
      {{end}}
    </div>
  </div>

  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">Рег.</th>
        <th class="text-left">OTP отправлен</th>
        <th class="text-left">OTP подтверждён</th>
        <th class="text-left">Письмо с паролем</th>
        {{if not .EventTitle}}<th class="text-left">Мероприятие</th>{{end}}
        <th class="text-left">Участник</th>
        <th class="text-left">Email</th>
        <th class="text-left">Организация</th>
        <th class="text-left">Округ</th>
        <th class="text-left">Статус</th>
        <th class="text-left">Чекин</th>
      </tr></thead>
      <tbody>
      {{range .Registrations}}
      <tr class="border-t text-sm">
        <td class="text-gray-500 whitespace-nowrap">{{fmtTime .RegDatetime}}</td>
        <td class="text-gray-500 whitespace-nowrap">{{fmtTimePtr .OTPSentAt}}</td>
        <td class="text-gray-500 whitespace-nowrap">{{fmtTimePtr .OTPVerifiedAt}}</td>
        <td class="text-gray-500 whitespace-nowrap">{{fmtTimePtr .WelcomeEmailSentAt}}</td>
        {{if not $.EventTitle}}<td>{{.EventTitle}}</td>{{end}}
        <td class="font-medium">{{.LastName}} {{.FirstName}}</td>
        <td class="text-gray-500">{{.Email}}</td>
        <td class="text-gray-500">{{.Organization}}</td>
        <td class="text-gray-500">{{.District}}</td>
        <td>{{statusBadge .Status}}</td>
        <td class="text-gray-500 whitespace-nowrap">{{fmtTimePtr .CheckedInAt}}</td>
      </tr>
      {{else}}<tr><td colspan="11" class="text-center text-gray-400 py-8">Регистраций нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Online tracking per event ──────────────────────────────────────────── */}}
{{define "tracking"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6">
    <a href="/panel/events" class="text-sm text-gray-500 hover:underline">← Мероприятия</a>
    <h1 class="text-2xl font-semibold text-gray-800 mt-1">Трекинг: {{.Event.Title}}</h1>
    <p class="text-sm text-gray-500 mt-0.5">Всего событий: <b>{{.Total}}</b></p>
  </div>

  {{/* Summary counters */}}
  <div class="grid grid-cols-4 gap-4 mb-6">
    {{range $action, $count := .Counts}}
    <div class="bg-white rounded-xl shadow-sm border p-4 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide">{{$action}}</p>
      <p class="text-2xl font-bold text-gray-800 mt-1">{{$count}}</p>
    </div>
    {{end}}
  </div>

  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">Время</th>
        <th class="text-left">Действие</th>
        <th class="text-left">Пользователь ID</th>
        <th class="text-left">IP</th>
        <th class="text-left">Страна / Город</th>
        <th class="text-left">Устройство</th>
        <th class="text-left">ОС</th>
        <th class="text-left">Браузер</th>
      </tr></thead>
      <tbody>
      {{range .Tracking}}
      <tr class="border-t">
        <td class="text-gray-500 text-xs">{{fmtTime .OccurredAt}}</td>
        <td>{{statusBadge .Action}}</td>
        <td class="text-gray-500">{{.UserID}}</td>
        <td class="text-gray-400 text-xs">{{.IPAddress}}</td>
        <td class="text-gray-400 text-xs">{{.GeoCountry}} / {{.GeoCity}}</td>
        <td class="text-gray-400 text-xs">{{.DeviceType}}</td>
        <td class="text-gray-400 text-xs">{{.OSName}}</td>
        <td class="text-gray-400 text-xs">{{.BrowserName}}</td>
      </tr>
      {{else}}<tr><td colspan="8" class="text-center text-gray-400 py-8">Событий трекинга нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── System logs ─────────────────────────────────────────────────────────── */}}
{{define "logs"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-semibold text-gray-800">Системные логи</h1>
    <span class="text-sm text-gray-500">Последние <b>{{.Total}}</b></span>
  </div>
  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">Время</th>
        <th class="text-left">Событие</th>
        <th class="text-left">Email</th>
        <th class="text-left">IP</th>
        <th class="text-left">Сообщение</th>
      </tr></thead>
      <tbody>
      {{range .Logs}}
      <tr class="border-t">
        <td class="text-gray-500 text-xs whitespace-nowrap">{{fmtTime .CreatedAt}}</td>
        <td>{{statusBadge .EventType}}</td>
        <td class="text-gray-600">{{.UserEmail}}</td>
        <td class="text-gray-400 text-xs">{{.IPAddress}}</td>
        <td class="text-gray-500 text-xs max-w-sm truncate">{{.Message}}</td>
      </tr>
      {{else}}<tr><td colspan="5" class="text-center text-gray-400 py-8">Логов нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Form builder ────────────────────────────────────────────────────────── */}}
{{define "formbuilder"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6">
    <a href="/panel/events" class="text-sm text-gray-500 hover:underline">← Мероприятия</a>
    <h1 class="text-2xl font-semibold text-gray-800 mt-1">Поля формы: {{.Event.Title}}</h1>
    <p class="text-sm text-gray-500 mt-1">Управление кастомными полями регистрации через API:
       <code class="bg-gray-100 px-1 rounded">/api/admin/events/{{.Event.ID}}/fields</code></p>
  </div>
  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">Порядок</th>
        <th class="text-left">Метка</th>
        <th class="text-left">Тип</th>
        <th class="text-left">Обязательное</th>
        <th class="text-left">В отчёте</th>
        <th class="text-left">В экспорте</th>
      </tr></thead>
      <tbody>
      {{range .Fields}}
      <tr class="border-t">
        <td class="text-gray-400">{{.SortOrder}}</td>
        <td class="font-medium">{{.Label}}</td>
        <td>{{statusBadge .FieldType}}</td>
        <td>{{boolRu .IsRequired}}</td>
        <td>{{boolRu .InReport}}</td>
        <td>{{boolRu .InExport}}</td>
      </tr>
      {{else}}<tr><td colspan="6" class="text-center text-gray-400 py-8">Полей нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Reference lists ────────────────────────────────────────────────────── */}}
{{define "reflists"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-gray-800">Справочники</h1>
    <p class="text-sm text-gray-500 mt-1">Управление справочниками через API:
       <code class="bg-gray-100 px-1 rounded">/api/admin/reflists</code></p>
  </div>
  <div class="bg-white rounded-xl shadow-sm border overflow-x-auto">
    <table class="table-auto w-full text-sm border-collapse">
      <thead><tr>
        <th class="text-left">ID</th>
        <th class="text-left">Название</th>
        <th class="text-left">Slug</th>
        <th class="text-left">Элементов</th>
      </tr></thead>
      <tbody>
      {{range .Lists}}
      <tr class="border-t">
        <td class="text-gray-400">{{.ID}}</td>
        <td class="font-medium">{{.Title}}</td>
        <td class="text-gray-500">{{.Slug}}</td>
        <td class="text-gray-500">—</td>
      </tr>
      {{else}}<tr><td colspan="4" class="text-center text-gray-400 py-8">Справочников нет</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</body></html>
{{end}}

{{/* ── Attendance ──────────────────────────────────────────────────────────── */}}
{{define "attendance"}}
{{template "head" .}}
{{template "nav" .Page}}
<div class="ml-60 p-8">
  <div class="mb-6">
    <a href="/panel/events" class="text-sm text-gray-500 hover:underline">← Мероприятия</a>
    <h1 class="text-2xl font-semibold text-gray-800 mt-1">{{.Title}}</h1>
    <span class="inline-block mt-1 text-xs px-2 py-0.5 rounded-full font-medium
      {{if eq .Event.EventType "online"}}bg-blue-100 text-blue-700
      {{else if eq .Event.EventType "hybrid"}}bg-purple-100 text-purple-700
      {{else}}bg-green-100 text-green-700{{end}}">
      {{if eq .Event.EventType "online"}}Онлайн{{else if eq .Event.EventType "hybrid"}}Гибридное{{else}}Очное{{end}}
    </span>
  </div>

  {{/* Online channel — shown for online and hybrid */}}
  {{if or (eq .Event.EventType "online") (eq .Event.EventType "hybrid")}}
  <h2 class="text-sm font-semibold text-gray-600 uppercase tracking-wide mb-2">
    Онлайн-канал <span class="font-normal normal-case text-gray-400">stream_connect / stream_disconnect</span>
  </h2>
  <div class="grid grid-cols-3 gap-4 mb-6">
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Подключений</p>
      <p class="text-4xl font-bold text-green-600">{{.StreamConnects}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Отключений</p>
      <p class="text-4xl font-bold text-red-500">{{.StreamDisconnects}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Сейчас онлайн</p>
      <p class="text-4xl font-bold text-blue-600">{{.OnlineActive}}</p>
    </div>
  </div>
  {{end}}

  {{/* Offline channel — shown for offline and hybrid */}}
  {{if ne .Event.EventType "online"}}
  <h2 class="text-sm font-semibold text-gray-600 uppercase tracking-wide mb-2">
    Очный канал <span class="font-normal normal-case text-gray-400">QR check_in / check_out</span>
  </h2>
  <div class="grid grid-cols-3 gap-4 mb-6">
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Вошли</p>
      <p class="text-4xl font-bold text-green-600">{{.Entries}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Вышли</p>
      <p class="text-4xl font-bold text-red-500">{{.Exits}}</p>
    </div>
    <div class="bg-white rounded-xl shadow-sm border p-5 text-center">
      <p class="text-xs text-gray-500 uppercase tracking-wide mb-1">Сейчас внутри</p>
      <p class="text-4xl font-bold text-blue-600">{{.Present}}</p>
    </div>
  </div>
  {{end}}

  <div class="bg-white rounded-xl shadow-sm border p-5">
    <a href="/panel/events/{{.Event.ID}}/tracking"
       class="text-sm text-blue-600 hover:underline">Детальный трекинг →</a>
    <span class="mx-3 text-gray-300">|</span>
    <a href="/panel/events/{{.Event.ID}}/registrations"
       class="text-sm text-blue-600 hover:underline">Список участников →</a>
  </div>
</div>
</body></html>
{{end}}
`
