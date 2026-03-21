'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { formatDate, eventTypeLabel, eventStatusLabel } from '@/lib/utils'
import { BarChart3, Download, Calendar, Users, Video, CheckCircle, TrendingUp, Filter } from 'lucide-react'

export default function AnalyticsPage() {
  const router = useRouter()
  const { token, user } = useAuthStore()
  const [filterStatus, setFilterStatus] = useState<string>('all')
  const [filterType, setFilterType] = useState<string>('all')

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  useEffect(() => {
    if (!token) router.push('/login')
    else if (!isAdmin) router.push('/dashboard')
  }, [token, isAdmin, router])

  const { data, isLoading } = useQuery({
    queryKey: ['analytics-events'],
    queryFn: () => apiClient.listEvents(token!),
    enabled: !!token,
    staleTime: 60_000,
  })

  if (!token || !isAdmin) return null

  const allEvents: Record<string, unknown>[] = data?.events ?? []

  // Apply filters
  const filtered = allEvents.filter((e) => {
    if (filterStatus !== 'all' && e.status !== filterStatus) return false
    if (filterType !== 'all' && e.type !== filterType) return false
    return true
  })

  // Aggregate stats
  const totalEvents = filtered.length
  const liveCount = filtered.filter((e) => e.status === 'live').length
  const endedCount = filtered.filter((e) => e.status === 'ended').length
  const publishedCount = filtered.filter((e) => e.status === 'published').length
  const draftCount = filtered.filter((e) => e.status === 'draft').length
  const withRecording = filtered.filter((e) => e.is_recording_enabled).length
  const onlineCount = filtered.filter((e) =>
    ['webinar', 'broadcast', 'conference'].includes(e.type as string)
  ).length

  // Count by type
  const byType: Record<string, number> = {}
  filtered.forEach((e) => {
    const t = e.type as string
    byType[t] = (byType[t] || 0) + 1
  })
  const typeEntries = Object.entries(byType).sort((a, b) => b[1] - a[1])
  const maxType = Math.max(...typeEntries.map((x) => x[1]), 1)

  // Count by status
  const statusData = [
    { label: 'В эфире', count: liveCount, color: 'bg-green-500' },
    { label: 'Опубликовано', count: publishedCount, color: 'bg-brand' },
    { label: 'Черновик', count: draftCount, color: 'bg-gray-500' },
    { label: 'Завершено', count: endedCount, color: 'bg-gray-600' },
  ].filter((s) => s.count > 0)
  const maxStatus = Math.max(...statusData.map((s) => s.count), 1)

  function downloadJSON() {
    const blob = new Blob([JSON.stringify({ events: filtered, generated_at: new Date().toISOString() }, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `analytics_${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  function downloadCSV() {
    const headers = ['ID', 'Название', 'Тип', 'Статус', 'Начало', 'Окончание', 'Вместимость', 'Запись']
    const rows = filtered.map((e) => [
      e.id,
      e.title,
      eventTypeLabel(e.type as string),
      eventStatusLabel(e.status as string),
      e.start_at ? formatDate(e.start_at as string) : '',
      e.end_at ? formatDate(e.end_at as string) : '',
      e.capacity || 'Без ограничений',
      e.is_recording_enabled ? 'Да' : 'Нет',
    ])
    const csv = '\uFEFF' + [headers, ...rows].map((r) => r.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `events_${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* Navbar */}
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link href="/" className="font-bold text-lg">Platform</Link>
            <span className="text-white/20">/</span>
            <span className="text-sm text-gray-400">Аналитика</span>
          </div>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/events" className="hover:text-white">Мероприятия</Link>
            <Link href="/dashboard" className="hover:text-white">Дашборд</Link>
            <Link href="/dashboard/admin" className="hover:text-white">Админ</Link>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-10">

        {/* Page header */}
        <div className="flex items-start justify-between mb-8">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <div className="w-9 h-9 rounded-xl bg-brand/20 flex items-center justify-center">
                <BarChart3 size={18} className="text-brand" />
              </div>
              <h1 className="text-2xl font-bold">Аналитика</h1>
            </div>
            <p className="text-gray-400 text-sm">Статистика по мероприятиям платформы</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={downloadCSV}
              className="flex items-center gap-1.5 text-xs px-3 py-2 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 transition-colors"
            >
              <Download size={12} />
              CSV
            </button>
            <button
              onClick={downloadJSON}
              className="flex items-center gap-1.5 text-xs px-3 py-2 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 transition-colors"
            >
              <Download size={12} />
              JSON
            </button>
            <Link
              href="/dashboard/events/new"
              className="flex items-center gap-1.5 text-xs px-4 py-2 rounded-lg bg-brand hover:bg-brand-dark text-white font-semibold transition-colors"
            >
              + Создать
            </Link>
          </div>
        </div>

        {/* Filters */}
        <div className="bg-gray-900 border border-white/10 rounded-xl p-4 mb-8 flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2 text-xs text-gray-400">
            <Filter size={12} />
            <span>Фильтры:</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">Статус:</span>
            {['all', 'live', 'published', 'draft', 'ended'].map((s) => (
              <button
                key={s}
                onClick={() => setFilterStatus(s)}
                className={`text-xs px-3 py-1 rounded-full transition-colors ${
                  filterStatus === s
                    ? 'bg-brand/20 border border-brand/30 text-brand'
                    : 'bg-white/5 border border-white/10 text-gray-400 hover:text-white'
                }`}
              >
                {s === 'all' ? 'Все' : eventStatusLabel(s)}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">Тип:</span>
            {['all', 'webinar', 'conference', 'broadcast', 'meeting'].map((t) => (
              <button
                key={t}
                onClick={() => setFilterType(t)}
                className={`text-xs px-3 py-1 rounded-full transition-colors ${
                  filterType === t
                    ? 'bg-brand/20 border border-brand/30 text-brand'
                    : 'bg-white/5 border border-white/10 text-gray-400 hover:text-white'
                }`}
              >
                {t === 'all' ? 'Все' : eventTypeLabel(t)}
              </button>
            ))}
          </div>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-20">
            <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin" />
          </div>
        ) : (
          <>
            {/* KPI cards */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
              <KPICard icon={<Calendar size={16} />} label="Всего мероприятий" value={totalEvents} color="brand" />
              <KPICard icon={<div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />} label="В эфире сейчас" value={liveCount} color="green" />
              <KPICard icon={<Video size={16} />} label="С записью" value={withRecording} color="purple" />
              <KPICard icon={<CheckCircle size={16} />} label="Завершено" value={endedCount} color="gray" />
            </div>

            {/* Charts row */}
            <div className="grid sm:grid-cols-2 gap-6 mb-8">

              {/* By status */}
              <div className="bg-gray-900 border border-white/10 rounded-2xl p-6">
                <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-5">По статусу</h3>
                <div className="space-y-3">
                  {statusData.map((s) => (
                    <div key={s.label} className="flex items-center gap-3">
                      <span className="text-xs text-gray-400 w-28 shrink-0">{s.label}</span>
                      <div className="flex-1 h-6 bg-white/5 rounded-lg overflow-hidden">
                        <div
                          className={`h-full ${s.color} rounded-lg transition-all duration-700 flex items-center justify-end pr-2`}
                          style={{ width: `${(s.count / maxStatus) * 100}%`, minWidth: s.count > 0 ? '2rem' : '0' }}
                        >
                          <span className="text-xs font-bold text-white">{s.count}</span>
                        </div>
                      </div>
                      <span className="text-xs text-gray-500 w-8 text-right">
                        {totalEvents > 0 ? Math.round((s.count / totalEvents) * 100) : 0}%
                      </span>
                    </div>
                  ))}
                  {statusData.length === 0 && (
                    <p className="text-xs text-gray-500 text-center py-4">Нет данных</p>
                  )}
                </div>
              </div>

              {/* By type */}
              <div className="bg-gray-900 border border-white/10 rounded-2xl p-6">
                <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-5">По типу</h3>
                <div className="space-y-3">
                  {typeEntries.map(([type, count]) => (
                    <div key={type} className="flex items-center gap-3">
                      <span className="text-xs text-gray-400 w-28 shrink-0">{eventTypeLabel(type)}</span>
                      <div className="flex-1 h-6 bg-white/5 rounded-lg overflow-hidden">
                        <div
                          className="h-full bg-brand rounded-lg transition-all duration-700 flex items-center justify-end pr-2"
                          style={{ width: `${(count / maxType) * 100}%`, minWidth: '2rem' }}
                        >
                          <span className="text-xs font-bold text-white">{count}</span>
                        </div>
                      </div>
                      <span className="text-xs text-gray-500 w-8 text-right">
                        {totalEvents > 0 ? Math.round((count / totalEvents) * 100) : 0}%
                      </span>
                    </div>
                  ))}
                  {typeEntries.length === 0 && (
                    <p className="text-xs text-gray-500 text-center py-4">Нет данных</p>
                  )}
                </div>
              </div>
            </div>

            {/* Online vs offline donut-like summary */}
            <div className="bg-gray-900 border border-white/10 rounded-2xl p-6 mb-8">
              <div className="flex items-center justify-between mb-5">
                <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Формат мероприятий</h3>
                <div className="flex items-center gap-4 text-xs text-gray-500">
                  <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-brand" />Онлайн: {onlineCount}</span>
                  <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-gray-500" />Офлайн: {totalEvents - onlineCount}</span>
                </div>
              </div>
              {totalEvents > 0 ? (
                <div className="h-8 bg-white/5 rounded-xl overflow-hidden flex">
                  {onlineCount > 0 && (
                    <div
                      className="h-full bg-brand flex items-center justify-center text-xs font-semibold text-white"
                      style={{ width: `${(onlineCount / totalEvents) * 100}%` }}
                    >
                      {Math.round((onlineCount / totalEvents) * 100)}%
                    </div>
                  )}
                  {totalEvents - onlineCount > 0 && (
                    <div
                      className="h-full bg-gray-600 flex items-center justify-center text-xs font-semibold text-white"
                      style={{ width: `${((totalEvents - onlineCount) / totalEvents) * 100}%` }}
                    >
                      {Math.round(((totalEvents - onlineCount) / totalEvents) * 100)}%
                    </div>
                  )}
                </div>
              ) : (
                <div className="h-8 bg-white/5 rounded-xl" />
              )}
            </div>

            {/* Events table */}
            <div className="bg-gray-900 border border-white/10 rounded-2xl overflow-hidden">
              <div className="flex items-center justify-between px-6 py-4 border-b border-white/5">
                <h3 className="text-sm font-semibold">
                  Мероприятия
                  <span className="text-xs text-gray-500 font-normal ml-2">({filtered.length})</span>
                </h3>
                <Link
                  href="/dashboard/admin"
                  className="text-xs text-brand hover:text-white transition-colors"
                >
                  Управление →
                </Link>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-white/5">
                      <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Название</th>
                      <th className="text-left px-4 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Тип</th>
                      <th className="text-left px-4 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Статус</th>
                      <th className="text-left px-4 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Начало</th>
                      <th className="text-left px-4 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Вместимость</th>
                      <th className="text-left px-4 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Запись</th>
                      <th className="px-4 py-3"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((event, i) => (
                      <tr
                        key={event.id as string}
                        className={`border-b border-white/5 hover:bg-white/2 transition-colors ${i % 2 === 0 ? '' : 'bg-white/[0.01]'}`}
                      >
                        <td className="px-6 py-3">
                          <div className="font-medium truncate max-w-xs">{event.title as string}</div>
                        </td>
                        <td className="px-4 py-3 text-gray-400 text-xs">{eventTypeLabel(event.type as string)}</td>
                        <td className="px-4 py-3">
                          <StatusBadge status={event.status as string} />
                        </td>
                        <td className="px-4 py-3 text-gray-400 text-xs">{formatDate(event.start_at as string)}</td>
                        <td className="px-4 py-3 text-gray-400 text-xs">
                          {event.capacity ? `${event.capacity} чел.` : '∞'}
                        </td>
                        <td className="px-4 py-3 text-xs">
                          {event.is_recording_enabled ? (
                            <span className="text-green-400">✓ Да</span>
                          ) : (
                            <span className="text-gray-600">—</span>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Link
                              href={`/events/${event.id}`}
                              className="text-xs text-gray-400 hover:text-white transition-colors"
                            >
                              →
                            </Link>
                            {(event.status as string) === 'ended' && (
                              <Link
                                href={`/events/${event.id}/ai`}
                                className="text-xs text-purple-400 hover:text-white transition-colors"
                              >
                                AI
                              </Link>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                    {filtered.length === 0 && (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-gray-500">
                          Нет мероприятий по выбранным фильтрам
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>

          </>
        )}
      </main>
    </div>
  )
}

function KPICard({
  icon,
  label,
  value,
  color,
}: {
  icon: React.ReactNode
  label: string
  value: number
  color: 'brand' | 'green' | 'purple' | 'gray'
}) {
  const colors = {
    brand: 'text-brand bg-brand/10',
    green: 'text-green-400 bg-green-500/10',
    purple: 'text-purple-400 bg-purple-500/10',
    gray: 'text-gray-400 bg-white/5',
  }
  return (
    <div className="bg-gray-900 border border-white/10 rounded-2xl p-5">
      <div className={`inline-flex items-center justify-center w-8 h-8 rounded-lg mb-3 ${colors[color]}`}>
        {icon}
      </div>
      <div className="text-3xl font-bold mb-1">{value}</div>
      <div className="text-xs text-gray-400">{label}</div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    live: 'bg-green-500/10 border-green-500/20 text-green-400',
    published: 'bg-brand/10 border-brand/20 text-brand',
    draft: 'bg-white/5 border-white/10 text-gray-400',
    ended: 'bg-white/5 border-white/10 text-gray-500',
    archived: 'bg-white/5 border-white/10 text-gray-600',
  }
  const labels: Record<string, string> = {
    live: 'В эфире', published: 'Опубликовано', draft: 'Черновик', ended: 'Завершено', archived: 'Архив',
  }
  return (
    <span className={`text-xs rounded-full px-2 py-0.5 border ${map[status] ?? 'bg-white/5 border-white/10 text-gray-400'}`}>
      {labels[status] ?? status}
    </span>
  )
}
