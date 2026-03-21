'use client'

import { useEffect, useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { formatDate, eventTypeLabel, eventStatusLabel } from '@/lib/utils'

export default function EventDetailPage() {
  const router = useRouter()
  const params = useParams()
  const eventId = params.id as string
  const { token } = useAuthStore()
  const [registered, setRegistered] = useState(false)
  const [regError, setRegError] = useState('')

  useEffect(() => {
    if (!token) router.push('/login')
  }, [token, router])

  const { data, isLoading } = useQuery({
    queryKey: ['event', eventId],
    queryFn: () => apiClient.getEvent(token!, eventId),
    enabled: !!token,
  })

  const { data: roomsData } = useQuery({
    queryKey: ['rooms', eventId],
    queryFn: () => apiClient.listRooms(token!, eventId),
    enabled: !!token,
  })

  const registerMutation = useMutation({
    mutationFn: () => apiClient.registerForEvent(token!, eventId),
    onSuccess: () => setRegistered(true),
    onError: (err: Error) => setRegError(err.message),
  })

  if (!token) return null
  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  const event = data?.event
  if (!event) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center text-white">
        Мероприятие не найдено
      </div>
    )
  }

  const rooms: Record<string, unknown>[] = roomsData?.rooms ?? []
  const activeRoom = rooms.find((r) => r.status === 'active')

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <div className="max-w-4xl mx-auto px-6 py-12">
        <Link href="/events" className="text-gray-400 hover:text-white text-sm mb-6 inline-block">
          ← Все мероприятия
        </Link>

        <div className="flex flex-wrap items-center gap-3 mb-6">
          <span className="text-sm bg-white/5 border border-white/10 rounded-full px-3 py-1 text-gray-400">
            {eventTypeLabel(event.type as string)}
          </span>
          <span className={`text-sm rounded-full px-3 py-1 ${
            event.status === 'live'
              ? 'bg-green-500/10 border border-green-500/20 text-green-400'
              : 'bg-brand/10 border border-brand/20 text-brand'
          }`}>
            {eventStatusLabel(event.status as string)}
          </span>
        </div>

        <h1 className="text-4xl font-bold mb-4">{event.title as string}</h1>
        <p className="text-gray-400 text-lg mb-8">{event.description as string}</p>

        <div className="grid sm:grid-cols-3 gap-4 mb-10">
          <InfoCard label="Начало" value={formatDate(event.start_at as string)} />
          <InfoCard label="Окончание" value={formatDate(event.end_at as string)} />
          <InfoCard label="Вместимость" value={event.capacity ? `${event.capacity} чел.` : 'Без ограничений'} />
        </div>

        {/* Register */}
        {event.status !== 'ended' && !registered && (
          <div className="mb-8">
            {regError && (
              <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg px-4 py-3 text-sm mb-4">
                {regError}
              </div>
            )}
            <button
              onClick={() => registerMutation.mutate()}
              disabled={registerMutation.isPending}
              className="px-8 py-3 bg-brand hover:bg-brand-dark disabled:opacity-50 rounded-xl font-semibold transition-colors"
            >
              {registerMutation.isPending ? 'Регистрация...' : 'Зарегистрироваться'}
            </button>
          </div>
        )}

        {registered && (
          <div className="bg-brand/10 border border-brand/20 text-brand rounded-lg px-4 py-3 text-sm mb-8">
            Вы успешно зарегистрированы на мероприятие
          </div>
        )}

        {/* Active room */}
        {activeRoom && (
          <div className="bg-gray-900 border border-brand/20 rounded-2xl p-6">
            <div className="flex items-center gap-2 mb-3">
              <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
              <span className="text-green-400 font-semibold">Эфир идёт</span>
            </div>
            <p className="text-gray-400 text-sm mb-5">Комната активна. Войдите в эфир.</p>
            <Link
              href={`/events/${eventId}/room?room=${activeRoom.id}`}
              className="inline-block px-6 py-3 bg-brand hover:bg-brand-dark rounded-xl font-semibold transition-colors"
            >
              Войти в эфир →
            </Link>
          </div>
        )}
      </div>
    </div>
  )
}

function InfoCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-gray-900 border border-white/10 rounded-xl p-4">
      <p className="text-xs text-gray-500 mb-1">{label}</p>
      <p className="font-semibold">{value}</p>
    </div>
  )
}
