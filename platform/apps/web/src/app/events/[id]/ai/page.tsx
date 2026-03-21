'use client'

import { useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import { useAuthStore } from '@/lib/store'
import { FileText, List, Zap, BookOpen, CheckSquare, Clock } from 'lucide-react'

interface AISummary {
  id: string
  type: string
  content: string
  language: string
  model_version: string
  generated_at: string
}

const SUMMARY_TYPES = [
  { key: 'transcript', label: 'Транскрипт', icon: FileText, description: 'Полная расшифровка речи' },
  { key: 'summary', label: 'Резюме', icon: List, description: 'Краткое содержание' },
  { key: 'highlights', label: 'Ключевые моменты', icon: Zap, description: 'Самое важное' },
  { key: 'chapters', label: 'Главы', icon: BookOpen, description: 'Таймкоды и разделы' },
  { key: 'action_items', label: 'Задачи', icon: CheckSquare, description: 'Что нужно сделать' },
]

export default function AIPage() {
  const router = useRouter()
  const params = useParams()
  const eventId = params.id as string
  const { token } = useAuthStore()

  useEffect(() => {
    if (!token) router.push('/login')
  }, [token, router])

  const { data: eventData } = useQuery({
    queryKey: ['event', eventId],
    queryFn: () => apiClient.getEvent(token!, eventId),
    enabled: !!token,
  })

  const { data, isLoading, error } = useQuery({
    queryKey: ['ai-summaries', eventId],
    queryFn: () => apiClient.getAISummaries(token!, eventId),
    enabled: !!token,
    retry: false,
  })

  if (!token) return null

  const event = eventData?.event as Record<string, unknown> | undefined
  const summaries = data?.summaries ?? {}

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <header className="border-b border-white/10 bg-gray-900/50 backdrop-blur">
        <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link href="/" className="font-bold text-lg">Platform</Link>
          <nav className="flex items-center gap-6 text-sm text-gray-400">
            <Link href="/events" className="hover:text-white">Мероприятия</Link>
            <Link href="/dashboard" className="hover:text-white">Дашборд</Link>
          </nav>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-6 py-12">
        <Link href={`/events/${eventId}`} className="text-gray-400 hover:text-white text-sm">
          ← Назад к мероприятию
        </Link>

        <div className="mt-6 mb-10">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-8 h-8 rounded-lg bg-purple-600/20 flex items-center justify-center">
              <Zap size={16} className="text-purple-400" />
            </div>
            <h1 className="text-2xl font-bold">AI-анализ</h1>
          </div>
          {event && (
            <p className="text-gray-400 text-sm ml-11">{event.title as string}</p>
          )}
        </div>

        {isLoading && (
          <div className="flex flex-col items-center justify-center py-20 gap-4">
            <div className="w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full animate-spin" />
            <p className="text-gray-400 text-sm">Загрузка AI-материалов...</p>
          </div>
        )}

        {error && (
          <div className="bg-yellow-500/10 border border-yellow-500/20 text-yellow-400 rounded-2xl p-6 text-sm">
            <p className="font-semibold mb-1">AI-анализ недоступен</p>
            <p className="text-yellow-400/70">
              Материалы появятся автоматически после завершения мероприятия и обработки записи.
              Обычно это занимает 10–30 минут.
            </p>
          </div>
        )}

        {!isLoading && !error && Object.keys(summaries).length === 0 && (
          <div className="text-center py-20">
            <div className="w-16 h-16 rounded-2xl bg-purple-600/10 flex items-center justify-center mx-auto mb-4">
              <Zap size={28} className="text-purple-400" />
            </div>
            <h2 className="font-semibold text-lg mb-2">Обработка ещё не завершена</h2>
            <p className="text-gray-400 text-sm max-w-xs mx-auto">
              AI-анализ запускается автоматически после окончания мероприятия.
              Проверьте позже.
            </p>
          </div>
        )}

        {!isLoading && !error && Object.keys(summaries).length > 0 && (
          <div className="space-y-6">
            {SUMMARY_TYPES.map(({ key, label, icon: Icon, description }) => {
              const summary = summaries[key] as AISummary | undefined
              if (!summary) return null

              return (
                <div key={key} className="bg-gray-900 border border-white/10 rounded-2xl overflow-hidden">
                  <div className="flex items-center gap-3 px-6 py-4 border-b border-white/5">
                    <div className="w-7 h-7 rounded-lg bg-purple-600/20 flex items-center justify-center">
                      <Icon size={14} className="text-purple-400" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-sm">{label}</h3>
                      <p className="text-xs text-gray-500">{description}</p>
                    </div>
                    <div className="ml-auto flex items-center gap-1.5 text-xs text-gray-500">
                      <Clock size={11} />
                      {new Date(summary.generated_at).toLocaleString('ru', {
                        day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit',
                      })}
                    </div>
                  </div>
                  <div className="px-6 py-5">
                    {key === 'transcript' ? (
                      <TranscriptContent content={summary.content} />
                    ) : key === 'chapters' ? (
                      <ChaptersContent content={summary.content} />
                    ) : key === 'action_items' ? (
                      <ActionItemsContent content={summary.content} />
                    ) : (
                      <p className="text-sm text-gray-300 leading-relaxed whitespace-pre-wrap">{summary.content}</p>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </main>
    </div>
  )
}

function TranscriptContent({ content }: { content: string }) {
  return (
    <div className="max-h-96 overflow-y-auto pr-2 space-y-3 text-sm text-gray-300 leading-relaxed">
      {content.split('\n').filter(Boolean).map((line, i) => (
        <p key={i}>{line}</p>
      ))}
    </div>
  )
}

function ChaptersContent({ content }: { content: string }) {
  const lines = content.split('\n').filter(Boolean)
  return (
    <div className="space-y-2">
      {lines.map((line, i) => {
        const match = line.match(/^(\d+:\d+)\s+(.+)$/)
        if (match) {
          return (
            <div key={i} className="flex items-start gap-3">
              <span className="font-mono text-xs text-purple-400 shrink-0 mt-0.5 w-12">{match[1]}</span>
              <span className="text-sm text-gray-300">{match[2]}</span>
            </div>
          )
        }
        return <p key={i} className="text-sm text-gray-300">{line}</p>
      })}
    </div>
  )
}

function ActionItemsContent({ content }: { content: string }) {
  const lines = content.split('\n').filter(Boolean)
  return (
    <ul className="space-y-2">
      {lines.map((line, i) => (
        <li key={i} className="flex items-start gap-2.5">
          <div className="w-4 h-4 rounded border border-white/20 shrink-0 mt-0.5" />
          <span className="text-sm text-gray-300">{line.replace(/^[-•*]\s*/, '')}</span>
        </li>
      ))}
    </ul>
  )
}
