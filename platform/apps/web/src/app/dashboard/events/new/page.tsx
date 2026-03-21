'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/lib/store'
import { apiClient } from '@/lib/api'

const EVENT_TYPES = [
  { value: 'webinar', label: 'Вебинар' },
  { value: 'conference', label: 'Конференция' },
  { value: 'broadcast', label: 'Трансляция' },
  { value: 'hybrid', label: 'Гибридный' },
  { value: 'meeting', label: 'Встреча' },
]

export default function NewEventPage() {
  const router = useRouter()
  const { token, user } = useAuthStore()
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const isAdmin = user?.role && ['event_admin', 'tenant_owner', 'super_admin'].includes(user.role)

  useEffect(() => {
    if (!token) router.push('/login')
    else if (!isAdmin) router.push('/dashboard')
  }, [token, isAdmin, router])

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError('')
    setLoading(true)

    const form = e.currentTarget
    const data = new FormData(form)

    const body: Record<string, unknown> = {
      title: data.get('title'),
      description: data.get('description'),
      type: data.get('type'),
      start_at: data.get('start_at'),
      end_at: data.get('end_at'),
      is_recording_enabled: data.get('is_recording_enabled') === 'on',
    }
    const cap = data.get('capacity')
    if (cap) body.capacity = Number(cap)

    const res = await apiClient.createEvent(token!, body)
    setLoading(false)

    if (!res.ok) {
      setError(res.error)
      return
    }

    const eventId = (res.data.event as Record<string, unknown>).id as string
    router.push(`/dashboard/events/${eventId}/edit`)
  }

  if (!token || !isAdmin) return null

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* Navbar */}
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur">
        <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link href="/dashboard" className="font-bold text-lg">Platform</Link>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/dashboard" className="hover:text-white">Дашборд</Link>
            <Link href="/dashboard/admin" className="hover:text-white">Админ</Link>
          </nav>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-6 py-12">
        <div className="mb-8">
          <Link href="/dashboard" className="text-gray-400 hover:text-white text-sm">
            ← Назад
          </Link>
          <h1 className="text-2xl font-bold mt-3">Создать мероприятие</h1>
          <p className="text-gray-400 text-sm mt-1">Заполните данные. После создания можно добавить комнату.</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl px-4 py-3 text-sm">
              {error}
            </div>
          )}

          <Field label="Название *">
            <input
              name="title"
              required
              placeholder="Вебинар по финансовой грамотности"
              className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 placeholder-gray-600"
            />
          </Field>

          <Field label="Описание">
            <textarea
              name="description"
              rows={3}
              placeholder="Краткое описание мероприятия..."
              className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 placeholder-gray-600 resize-none"
            />
          </Field>

          <Field label="Тип мероприятия *">
            <select
              name="type"
              required
              className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
            >
              {EVENT_TYPES.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </Field>

          <div className="grid sm:grid-cols-2 gap-5">
            <Field label="Начало *">
              <input
                name="start_at"
                type="datetime-local"
                required
                className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
              />
            </Field>
            <Field label="Окончание *">
              <input
                name="end_at"
                type="datetime-local"
                required
                className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50"
              />
            </Field>
          </div>

          <Field label="Вместимость (оставьте пустым — без ограничений)">
            <input
              name="capacity"
              type="number"
              min={1}
              placeholder="500"
              className="w-full bg-gray-900 border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-brand/50 placeholder-gray-600"
            />
          </Field>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              name="is_recording_enabled"
              type="checkbox"
              className="w-4 h-4 accent-brand"
            />
            <span className="text-sm text-gray-300">Включить запись трансляции</span>
          </label>

          <div className="pt-4">
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-brand hover:bg-brand-dark disabled:opacity-50 rounded-xl font-semibold transition-colors"
            >
              {loading ? 'Создание...' : 'Создать мероприятие →'}
            </button>
          </div>
        </form>
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
