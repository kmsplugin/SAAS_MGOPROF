'use client'

import { useEffect, useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { eventStatusLabel } from '@/lib/utils'

const EVENT_TYPES = [
  { value: 'webinar', label: 'Вебинар' },
  { value: 'conference', label: 'Конференция' },
  { value: 'broadcast', label: 'Трансляция' },
  { value: 'hybrid', label: 'Гибридный' },
  { value: 'meeting', label: 'Встреча' },
]

const ROOM_MODES = [
  { value: 'meeting', label: 'Встреча (все с камерой)' },
  { value: 'webinar', label: 'Вебинар (спикер + зрители)' },
  { value: 'broadcast', label: 'Трансляция (только просмотр)' },
  { value: 'stage', label: 'Сцена (несколько спикеров)' },
]

export default function EditEventPage() {
  const router = useRouter()
  const params = useParams()
  const eventId = params.id as string
  const { token, user } = useAuthStore()
  const qc = useQueryClient()

  const [saveError, setSaveError] = useState('')
  const [saveOk, setSaveOk] = useState(false)
  const [roomMode, setRoomMode] = useState('webinar')
  const [roomMax, setRoomMax] = useState('')
  const [roomError, setRoomError] = useState('')
  const [roomOk, setRoomOk] = useState(false)

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  useEffect(() => {
    if (!token) router.push('/login')
    else if (!isAdmin) router.push('/dashboard')
  }, [token, isAdmin, router])

  const { data, isLoading } = useQuery({
    queryKey: ['event', eventId],
    queryFn: () => apiClient.getEvent(token!, eventId),
    enabled: !!token,
  })

  const { data: roomsData, refetch: refetchRooms } = useQuery({
    queryKey: ['rooms', eventId],
    queryFn: () => apiClient.listRooms(token!, eventId),
    enabled: !!token,
  })

  const publishMutation = useMutation({
    mutationFn: () => apiClient.publishEvent(token!, eventId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['event', eventId] }),
  })

  if (!token || !isAdmin) return null

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  const event = data?.event as Record<string, unknown>
  if (!event) return null

  const rooms: Record<string, unknown>[] = roomsData?.rooms ?? []
  const activeRoom = rooms.find((r) => r.status === 'active')

  async function handleSave(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSaveError('')
    setSaveOk(false)
    const form = e.currentTarget
    const fd = new FormData(form)
    const body: Record<string, unknown> = {
      title: fd.get('title'),
      description: fd.get('description'),
      type: fd.get('type'),
      start_at: fd.get('start_at'),
      end_at: fd.get('end_at'),
      is_recording_enabled: fd.get('is_recording_enabled') === 'on',
    }
    const cap = fd.get('capacity')
    if (cap) body.capacity = Number(cap)

    const res = await apiClient.updateEvent(token!, eventId, body)
    if (!res.ok) { setSaveError(res.error); return }
    setSaveOk(true)
    qc.invalidateQueries({ queryKey: ['event', eventId] })
  }

  async function handleCreateRoom() {
    setRoomError('')
    setRoomOk(false)
    const body: { event_id: string; mode: string; max_participants?: number } = {
      event_id: eventId,
      mode: roomMode,
    }
    if (roomMax) body.max_participants = Number(roomMax)
    const res = await apiClient.createRoom(token!, body)
    if (!res.ok) { setRoomError(res.error); return }
    setRoomOk(true)
    refetchRooms()
  }

  async function handleEndRoom(roomId: string) {
    await apiClient.endRoom(token!, roomId)
    refetchRooms()
  }

  function toLocalDateTime(iso: string | undefined) {
    if (!iso) return ''
    return new Date(iso as string).toISOString().slice(0, 16)
  }

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur">
        <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link href="/dashboard" className="font-bold text-lg">Platform</Link>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/dashboard" className="hover:text-white">Дашборд</Link>
            <Link href="/dashboard/admin" className="hover:text-white">Админ</Link>
          </nav>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-6 py-12 space-y-10">
        <div>
          <Link href="/dashboard/admin" className="text-gray-400 hover:text-white text-sm">← Назад</Link>
          <div className="flex items-center gap-3 mt-3">
            <h1 className="text-2xl font-bold">Редактировать мероприятие</h1>
            <span className="text-xs rounded-full px-3 py-1 bg-white/5 border border-white/10 text-gray-400">
              {eventStatusLabel(event.status as string)}
            </span>
          </div>
        </div>

        {/* Edit form */}
        <section className="bg-gray-900 border border-white/10 rounded-2xl p-6">
          <h2 className="font-semibold mb-5">Данные мероприятия</h2>
          <form onSubmit={handleSave} className="space-y-4">
            {saveError && (
              <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl px-4 py-3 text-sm">{saveError}</div>
            )}
            {saveOk && (
              <div className="bg-green-500/10 border border-green-500/20 text-green-400 rounded-xl px-4 py-3 text-sm">Сохранено ✓</div>
            )}

            <Field label="Название *">
              <input
                name="title"
                required
                defaultValue={event.title as string}
                className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
              />
            </Field>

            <Field label="Описание">
              <textarea
                name="description"
                rows={3}
                defaultValue={event.description as string}
                className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 resize-none"
              />
            </Field>

            <Field label="Тип">
              <select
                name="type"
                defaultValue={event.type as string}
                className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
              >
                {EVENT_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </Field>

            <div className="grid sm:grid-cols-2 gap-4">
              <Field label="Начало *">
                <input
                  name="start_at"
                  type="datetime-local"
                  required
                  defaultValue={toLocalDateTime(event.start_at as string)}
                  className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
                />
              </Field>
              <Field label="Окончание *">
                <input
                  name="end_at"
                  type="datetime-local"
                  required
                  defaultValue={toLocalDateTime(event.end_at as string)}
                  className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
                />
              </Field>
            </div>

            <Field label="Вместимость">
              <input
                name="capacity"
                type="number"
                min={1}
                defaultValue={(event.capacity as number) || ''}
                placeholder="Без ограничений"
                className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 placeholder-gray-600"
              />
            </Field>

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                name="is_recording_enabled"
                type="checkbox"
                defaultChecked={event.is_recording_enabled as boolean}
                className="w-4 h-4 accent-brand"
              />
              <span className="text-sm text-gray-300">Запись трансляции</span>
            </label>

            <div className="flex items-center gap-3 pt-2">
              <button
                type="submit"
                className="px-6 py-2.5 bg-brand hover:bg-brand-dark rounded-xl text-sm font-semibold transition-colors"
              >
                Сохранить
              </button>
              {event.status === 'draft' && (
                <button
                  type="button"
                  onClick={() => publishMutation.mutate()}
                  disabled={publishMutation.isPending}
                  className="px-6 py-2.5 bg-green-600 hover:bg-green-700 disabled:opacity-50 rounded-xl text-sm font-semibold transition-colors"
                >
                  {publishMutation.isPending ? 'Публикация...' : '→ Опубликовать'}
                </button>
              )}
            </div>
          </form>
        </section>

        {/* Room management */}
        <section className="bg-gray-900 border border-white/10 rounded-2xl p-6">
          <h2 className="font-semibold mb-5">Комнаты (видеотрансляция)</h2>

          {rooms.length > 0 && (
            <div className="space-y-3 mb-6">
              {rooms.map((room) => (
                <div key={room.id as string} className="flex items-center justify-between bg-gray-800 rounded-xl px-4 py-3">
                  <div>
                    <p className="text-sm font-medium">{room.room_name as string}</p>
                    <p className="text-xs text-gray-500 mt-0.5">
                      {room.mode as string} · {room.max_participants ? `до ${room.max_participants} чел.` : 'без лимита'}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    {room.status === 'active' ? (
                      <>
                        <span className="flex items-center gap-1 text-xs text-green-400">
                          <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
                          Активна
                        </span>
                        <Link
                          href={`/events/${eventId}/room?room=${room.id}`}
                          className="text-xs text-brand hover:text-white"
                        >
                          Открыть
                        </Link>
                        <button
                          onClick={() => handleEndRoom(room.id as string)}
                          className="text-xs text-red-400 hover:text-red-300"
                        >
                          Завершить
                        </button>
                      </>
                    ) : (
                      <span className="text-xs text-gray-500">Завершена</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {!activeRoom && (
            <div className="space-y-4">
              <div className="grid sm:grid-cols-2 gap-4">
                <Field label="Режим комнаты">
                  <select
                    value={roomMode}
                    onChange={(e) => setRoomMode(e.target.value)}
                    className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
                  >
                    {ROOM_MODES.map((m) => (
                      <option key={m.value} value={m.value}>{m.label}</option>
                    ))}
                  </select>
                </Field>
                <Field label="Макс. участников (необязательно)">
                  <input
                    type="number"
                    min={2}
                    value={roomMax}
                    onChange={(e) => setRoomMax(e.target.value)}
                    placeholder="500"
                    className="w-full bg-gray-800 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 placeholder-gray-600"
                  />
                </Field>
              </div>
              {roomError && (
                <p className="text-red-400 text-sm">{roomError}</p>
              )}
              {roomOk && (
                <p className="text-green-400 text-sm">Комната создана ✓</p>
              )}
              <button
                onClick={handleCreateRoom}
                className="px-6 py-2.5 bg-brand/10 hover:bg-brand/20 border border-brand/20 text-brand rounded-xl text-sm font-semibold transition-colors"
              >
                + Создать комнату
              </button>
            </div>
          )}
        </section>

        {/* Links */}
        <div className="flex gap-4 text-sm">
          <Link href={`/events/${eventId}`} className="text-gray-400 hover:text-white">
            Страница мероприятия →
          </Link>
          <Link href={`/events/${eventId}/ai`} className="text-gray-400 hover:text-white">
            AI-анализ →
          </Link>
        </div>
      </main>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-sm text-gray-400 mb-2">{label}</label>
      {children}
    </div>
  )
}
