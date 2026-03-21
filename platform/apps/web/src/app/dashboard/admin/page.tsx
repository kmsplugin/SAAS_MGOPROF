'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { formatDate, eventStatusLabel, eventTypeLabel } from '@/lib/utils'

export default function AdminPage() {
  const router = useRouter()
  const { token, user } = useAuthStore()
  const qc = useQueryClient()

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  useEffect(() => {
    if (!token) router.push('/login')
    else if (!isAdmin) router.push('/dashboard')
  }, [token, isAdmin, router])

  const { data, isLoading } = useQuery({
    queryKey: ['all-events'],
    queryFn: () => apiClient.listEvents(token!),
    enabled: !!token,
    refetchInterval: 15_000,
  })

  const publishMutation = useMutation({
    mutationFn: (id: string) => apiClient.publishEvent(token!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['all-events'] }),
  })

  if (!token || !isAdmin) return null

  const events: Record<string, unknown>[] = data?.events ?? []

  const byStatus = {
    live: events.filter((e) => e.status === 'live'),
    published: events.filter((e) => e.status === 'published'),
    draft: events.filter((e) => e.status === 'draft'),
    ended: events.filter((e) => e.status === 'ended'),
  }

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* Navbar */}
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link href="/" className="font-bold text-lg">Platform</Link>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/events" className="hover:text-white">Мероприятия</Link>
            <Link href="/dashboard" className="hover:text-white">Дашборд</Link>
            <Link href="/dashboard/admin" className="text-white font-medium">Админ</Link>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-12">
        {/* Header */}
        <div className="flex items-center justify-between mb-10">
          <div>
            <h1 className="text-2xl font-bold">Панель администратора</h1>
            <p className="text-gray-400 text-sm mt-1">Управление мероприятиями и трансляциями</p>
          </div>
          <Link
            href="/dashboard/events/new"
            className="px-5 py-2.5 bg-brand hover:bg-brand-dark rounded-xl text-sm font-semibold transition-colors"
          >
            + Создать мероприятие
          </Link>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-10">
          <StatCard label="В эфире" value={byStatus.live.length} accent="green" />
          <StatCard label="Опубликовано" value={byStatus.published.length} accent="brand" />
          <StatCard label="Черновик" value={byStatus.draft.length} accent="gray" />
          <StatCard label="Завершено" value={byStatus.ended.length} accent="gray" />
        </div>

        {/* Live events — top priority */}
        {byStatus.live.length > 0 && (
          <Section title="🔴 Сейчас в эфире">
            {byStatus.live.map((event) => (
              <EventRow
                key={event.id as string}
                event={event}
                onPublish={() => publishMutation.mutate(event.id as string)}
              />
            ))}
          </Section>
        )}

        {/* Published */}
        {byStatus.published.length > 0 && (
          <Section title="Опубликованные">
            {byStatus.published.map((event) => (
              <EventRow
                key={event.id as string}
                event={event}
                onPublish={() => publishMutation.mutate(event.id as string)}
              />
            ))}
          </Section>
        )}

        {/* Drafts */}
        {byStatus.draft.length > 0 && (
          <Section title="Черновики">
            {byStatus.draft.map((event) => (
              <EventRow
                key={event.id as string}
                event={event}
                onPublish={() => publishMutation.mutate(event.id as string)}
                showPublish
              />
            ))}
          </Section>
        )}

        {/* Ended */}
        {byStatus.ended.length > 0 && (
          <Section title="Завершённые">
            {byStatus.ended.map((event) => (
              <EventRow
                key={event.id as string}
                event={event}
                onPublish={() => {}}
                showAI
              />
            ))}
          </Section>
        )}

        {isLoading && (
          <div className="flex justify-center py-16">
            <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin" />
          </div>
        )}

        {!isLoading && events.length === 0 && (
          <div className="text-center py-16 text-gray-500">
            <p className="mb-4">Мероприятий нет</p>
            <Link
              href="/dashboard/events/new"
              className="text-sm text-brand hover:text-white underline"
            >
              Создать первое мероприятие
            </Link>
          </div>
        )}
      </main>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-10">
      <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-4">{title}</h2>
      <div className="space-y-3">{children}</div>
    </div>
  )
}

function EventRow({
  event,
  onPublish,
  showPublish = false,
  showAI = false,
}: {
  event: Record<string, unknown>
  onPublish: () => void
  showPublish?: boolean
  showAI?: boolean
}) {
  const id = event.id as string

  return (
    <div className="bg-gray-900 border border-white/10 hover:border-white/20 rounded-xl p-5 transition-colors">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-1.5 flex-wrap">
            <StatusDot status={event.status as string} />
            <h3 className="font-medium truncate">{event.title as string}</h3>
            <span className="text-xs text-gray-500 shrink-0">{eventTypeLabel(event.type as string)}</span>
          </div>
          <p className="text-sm text-gray-400">
            {formatDate(event.start_at as string)} — {formatDate(event.end_at as string)}
          </p>
          {event.capacity && (
            <p className="text-xs text-gray-500 mt-1">Вместимость: {event.capacity as number} чел.</p>
          )}
        </div>

        <div className="flex items-center gap-2 flex-wrap shrink-0">
          <Link
            href={`/events/${id}`}
            className="text-xs px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 transition-colors"
          >
            Страница
          </Link>
          <Link
            href={`/dashboard/events/${id}/edit`}
            className="text-xs px-3 py-1.5 rounded-lg bg-brand/10 hover:bg-brand/20 border border-brand/20 text-brand transition-colors"
          >
            Редактировать
          </Link>
          {showPublish && (
            <button
              onClick={onPublish}
              className="text-xs px-3 py-1.5 rounded-lg bg-green-600/20 hover:bg-green-600/30 border border-green-600/20 text-green-400 transition-colors"
            >
              Опубликовать
            </button>
          )}
          {showAI && (
            <Link
              href={`/events/${id}/ai`}
              className="text-xs px-3 py-1.5 rounded-lg bg-purple-600/10 hover:bg-purple-600/20 border border-purple-600/20 text-purple-400 transition-colors"
            >
              AI-анализ
            </Link>
          )}
        </div>
      </div>
    </div>
  )
}

function StatusDot({ status }: { status: string }) {
  const color =
    status === 'live' ? 'bg-green-500 animate-pulse' :
    status === 'published' ? 'bg-brand' :
    status === 'draft' ? 'bg-gray-500' :
    'bg-gray-600'
  const label = eventStatusLabel(status)
  return (
    <span className="flex items-center gap-1.5 shrink-0">
      <span className={`w-1.5 h-1.5 rounded-full ${color}`} />
      <span className="text-xs text-gray-400">{label}</span>
    </span>
  )
}

function StatCard({ label, value, accent }: { label: string; value: number; accent: 'green' | 'brand' | 'gray' }) {
  const colorMap = {
    green: 'text-green-400',
    brand: 'text-brand',
    gray: 'text-gray-300',
  }
  return (
    <div className="bg-gray-900 border border-white/10 rounded-xl p-4 text-center">
      <div className={`text-3xl font-bold mb-1 ${colorMap[accent]}`}>{value}</div>
      <div className="text-xs text-gray-500">{label}</div>
    </div>
  )
}
