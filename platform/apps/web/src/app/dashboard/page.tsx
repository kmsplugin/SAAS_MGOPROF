'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { formatDate, eventStatusLabel, eventTypeLabel } from '@/lib/utils'

export default function DashboardPage() {
  const router = useRouter()
  const { token, user, logout } = useAuthStore()

  useEffect(() => {
    if (!token) router.push('/login')
  }, [token, router])

  const { data } = useQuery({
    queryKey: ['my-events'],
    queryFn: () => apiClient.listEvents(token!),
    enabled: !!token,
  })

  if (!token) return null

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* Navbar */}
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link href="/" className="font-bold text-lg">Platform</Link>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/events" className="hover:text-white">Мероприятия</Link>
            {isAdmin && <Link href="/dashboard/events/new" className="hover:text-white">Создать</Link>}
            {isAdmin && <Link href="/dashboard/analytics" className="hover:text-white">Аналитика</Link>}
            {isAdmin && <Link href="/dashboard/admin" className="hover:text-white font-medium text-brand">Админ</Link>}
            <button onClick={() => { logout(); router.push('/login') }} className="hover:text-white">
              Выйти
            </button>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-12">
        {/* Profile */}
        <div className="bg-gray-900 border border-white/10 rounded-2xl p-6 mb-10">
          <div className="flex items-center gap-4">
            <div className="w-14 h-14 rounded-full bg-brand/20 border border-brand/30 flex items-center justify-center text-xl font-bold text-brand">
              {user?.first_name?.[0]}{user?.last_name?.[0]}
            </div>
            <div>
              <h2 className="font-semibold text-lg">{user?.first_name} {user?.last_name}</h2>
              <p className="text-gray-400 text-sm">{user?.email}</p>
              <span className="text-xs bg-brand/10 border border-brand/20 text-brand rounded-full px-2 py-0.5 mt-1 inline-block">
                {user?.role}
              </span>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="grid sm:grid-cols-3 gap-4 mb-10">
          <StatCard label="Мероприятий" value={data?.total ?? 0} />
          <StatCard label="Активных" value={data?.events?.filter((e: Record<string, unknown>) => e.status === 'live').length ?? 0} />
          <StatCard label="Завершённых" value={data?.events?.filter((e: Record<string, unknown>) => e.status === 'ended').length ?? 0} />
        </div>

        {/* Events list */}
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-semibold text-xl">Мероприятия</h3>
          {isAdmin && (
            <Link
              href="/dashboard/events/new"
              className="text-sm bg-brand/10 border border-brand/20 text-brand hover:bg-brand/20 rounded-lg px-4 py-2 transition-colors"
            >
              + Создать мероприятие
            </Link>
          )}
        </div>

        <div className="space-y-3">
          {data?.events?.map((event: Record<string, unknown>) => (
            <div key={event.id as string} className="bg-gray-900 border border-white/10 rounded-xl p-5 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-1">
                  <h4 className="font-medium">{event.title as string}</h4>
                  <span className="text-xs text-gray-500">{eventTypeLabel(event.type as string)}</span>
                </div>
                <p className="text-sm text-gray-400">{formatDate(event.start_at as string)}</p>
              </div>
              <div className="flex items-center gap-3">
                <span className={`text-xs rounded-full px-3 py-1 ${
                  event.status === 'live'
                    ? 'bg-green-500/10 border border-green-500/20 text-green-400'
                    : event.status === 'published'
                    ? 'bg-brand/10 border border-brand/20 text-brand'
                    : 'bg-white/5 border border-white/10 text-gray-400'
                }`}>
                  {eventStatusLabel(event.status as string)}
                </span>
                <Link
                  href={`/events/${event.id}`}
                  className="text-sm text-gray-400 hover:text-white transition-colors"
                >
                  Открыть →
                </Link>
              </div>
            </div>
          ))}

          {data?.events?.length === 0 && (
            <div className="text-center py-12 text-gray-500">
              Мероприятий нет
            </div>
          )}
        </div>
      </main>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-gray-900 border border-white/10 rounded-xl p-5 text-center">
      <div className="text-3xl font-bold mb-1">{value}</div>
      <div className="text-sm text-gray-400">{label}</div>
    </div>
  )
}
