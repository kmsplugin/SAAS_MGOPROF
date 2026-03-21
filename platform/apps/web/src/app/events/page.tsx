'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { formatDate, eventStatusLabel, eventTypeLabel } from '@/lib/utils'

export default function EventsPage() {
  const router = useRouter()
  const { token, user } = useAuthStore()

  useEffect(() => {
    if (!token) router.push('/login')
  }, [token, router])

  const { data, isLoading, error } = useQuery({
    queryKey: ['events'],
    queryFn: () => apiClient.listEvents(token!),
    enabled: !!token,
  })

  if (!token) return null

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <div className="max-w-6xl mx-auto px-6 py-12">
        <div className="flex items-center justify-between mb-10">
          <div>
            <h1 className="text-3xl font-bold mb-1">Мероприятия</h1>
            <p className="text-gray-400">Все запланированные и активные мероприятия</p>
          </div>
          {isAdmin && (
            <Link
              href="/dashboard/events/new"
              className="px-5 py-2.5 bg-brand hover:bg-brand-dark rounded-lg font-semibold text-sm transition-colors"
            >
              + Создать
            </Link>
          )}
        </div>

        {isLoading && (
          <div className="flex items-center justify-center py-24">
            <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin" />
          </div>
        )}

        {error && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg px-4 py-3">
            Ошибка загрузки мероприятий
          </div>
        )}

        {data && (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {data.events?.map((event: Record<string, unknown>) => (
              <Link
                key={event.id as string}
                href={`/events/${event.id}`}
                className="bg-gray-900 border border-white/10 rounded-2xl p-6 hover:border-brand/40 transition-colors group"
              >
                <div className="flex items-start justify-between mb-4">
                  <span className="text-xs bg-white/5 border border-white/10 rounded-full px-3 py-1 text-gray-400">
                    {eventTypeLabel(event.type as string)}
                  </span>
                  <span className={`text-xs rounded-full px-3 py-1 ${statusBadgeClass(event.status as string)}`}>
                    {eventStatusLabel(event.status as string)}
                  </span>
                </div>
                <h3 className="font-semibold text-lg mb-2 group-hover:text-brand transition-colors line-clamp-2">
                  {event.title as string}
                </h3>
                <p className="text-gray-400 text-sm line-clamp-2 mb-4">
                  {(event.description as string) || 'Описание не указано'}
                </p>
                <div className="text-xs text-gray-500">
                  {formatDate(event.start_at as string)}
                </div>
              </Link>
            ))}

            {data.events?.length === 0 && (
              <div className="col-span-3 text-center py-16 text-gray-500">
                Мероприятий пока нет
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function statusBadgeClass(status: string): string {
  if (status === 'live') return 'bg-green-500/10 border border-green-500/20 text-green-400'
  if (status === 'published') return 'bg-brand/10 border border-brand/20 text-brand'
  if (status === 'ended') return 'bg-gray-500/10 border border-gray-500/20 text-gray-400'
  return 'bg-white/5 border border-white/10 text-gray-400'
}
