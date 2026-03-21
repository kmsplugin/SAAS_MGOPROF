'use client'

import { useState, useEffect, useRef } from 'react'
import { MessageSquare, Send, ChevronDown, ChevronUp, CheckCircle } from 'lucide-react'
import { apiClient } from '@/lib/api'

interface QAPanelProps {
  token: string
  eventId: string
  isHost: boolean
}

interface Question {
  id: string
  text: string
  status: 'new' | 'in_progress' | 'answered'
  user_id: string
  created_at: string
}

interface Message {
  id: string
  text: string
  user_id: string
  created_at: string
}

export function QAPanel({ token, eventId, isHost }: QAPanelProps) {
  const [questions, setQuestions] = useState<Question[]>([])
  const [newText, setNewText] = useState('')
  const [sending, setSending] = useState(false)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [messages, setMessages] = useState<Record<string, Message[]>>({})
  const [replyText, setReplyText] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  // Poll questions every 5s
  useEffect(() => {
    let alive = true
    async function poll() {
      const res = await apiClient.listQuestions(token, eventId)
      if (res.ok && alive) {
        setQuestions(res.data.questions as Question[])
      }
    }
    poll()
    const id = setInterval(poll, 5_000)
    return () => { alive = false; clearInterval(id) }
  }, [token, eventId])

  // Load messages when question is expanded
  useEffect(() => {
    if (!expandedId) return
    apiClient.listMessages(token, eventId, expandedId).then((res) => {
      if (res.ok) setMessages((prev) => ({ ...prev, [expandedId]: res.data.messages as Message[] }))
    })
  }, [expandedId, token, eventId])

  async function submitQuestion() {
    if (!newText.trim()) return
    setSending(true)
    const res = await apiClient.createQuestion(token, eventId, newText.trim())
    setSending(false)
    if (res.ok) {
      setNewText('')
      // Refresh
      const listRes = await apiClient.listQuestions(token, eventId)
      if (listRes.ok) setQuestions(listRes.data.questions as Question[])
    }
  }

  async function submitReply(questionId: string) {
    if (!replyText.trim()) return
    await apiClient.addMessage(token, eventId, questionId, replyText.trim())
    setReplyText('')
    const res = await apiClient.listMessages(token, eventId, questionId)
    if (res.ok) setMessages((prev) => ({ ...prev, [questionId]: res.data.messages as Message[] }))
  }

  const unanswered = questions.filter((q) => q.status !== 'answered')
  const answered = questions.filter((q) => q.status === 'answered')

  return (
    <div
      className="flex flex-col h-full overflow-hidden"
      style={{ backgroundColor: 'var(--room-surface)', borderColor: 'var(--room-border)' }}
    >
      {/* Header */}
      <div
        className="flex items-center gap-2 px-4 py-3 border-b shrink-0"
        style={{ borderColor: 'var(--room-border)' }}
      >
        <MessageSquare size={14} style={{ color: 'var(--room-brand)' }} />
        <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--room-text-muted)' }}>
          Вопросы и ответы
        </span>
        {unanswered.length > 0 && (
          <span
            className="ml-auto text-xs rounded-full px-2 py-0.5"
            style={{ backgroundColor: 'var(--room-brand-dim)', color: 'var(--room-brand)' }}
          >
            {unanswered.length}
          </span>
        )}
      </div>

      {/* Questions list */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {questions.length === 0 && (
          <p className="text-xs text-center py-6" style={{ color: 'var(--room-text-muted)' }}>
            Вопросов пока нет
          </p>
        )}

        {unanswered.map((q) => (
          <QuestionCard
            key={q.id}
            question={q}
            messages={messages[q.id] ?? []}
            expanded={expandedId === q.id}
            isHost={isHost}
            replyText={replyText}
            onToggle={() => setExpandedId(expandedId === q.id ? null : q.id)}
            onReplyChange={setReplyText}
            onReply={() => submitReply(q.id)}
          />
        ))}

        {answered.length > 0 && (
          <div>
            <p className="text-xs px-1 pb-1" style={{ color: 'var(--room-text-muted)' }}>
              Отвечено ({answered.length})
            </p>
            {answered.map((q) => (
              <QuestionCard
                key={q.id}
                question={q}
                messages={messages[q.id] ?? []}
                expanded={expandedId === q.id}
                isHost={isHost}
                replyText={replyText}
                onToggle={() => setExpandedId(expandedId === q.id ? null : q.id)}
                onReplyChange={setReplyText}
                onReply={() => submitReply(q.id)}
              />
            ))}
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Submit question (viewers only) */}
      {!isHost && (
        <div
          className="shrink-0 p-3 border-t"
          style={{ borderColor: 'var(--room-border)' }}
        >
          <div className="flex gap-2">
            <input
              value={newText}
              onChange={(e) => setNewText(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submitQuestion() } }}
              placeholder="Задать вопрос..."
              className="flex-1 rounded-lg px-3 py-2 text-sm bg-transparent border focus:outline-none"
              style={{
                borderColor: 'var(--room-border)',
                color: 'var(--room-text)',
              }}
            />
            <button
              onClick={submitQuestion}
              disabled={sending || !newText.trim()}
              className="p-2 rounded-lg transition-opacity disabled:opacity-40"
              style={{ backgroundColor: 'var(--room-brand-dim)', color: 'var(--room-brand)' }}
            >
              <Send size={14} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function QuestionCard({
  question,
  messages,
  expanded,
  isHost,
  replyText,
  onToggle,
  onReplyChange,
  onReply,
}: {
  question: Question
  messages: Message[]
  expanded: boolean
  isHost: boolean
  replyText: string
  onToggle: () => void
  onReplyChange: (v: string) => void
  onReply: () => void
}) {
  return (
    <div
      className="rounded-xl border overflow-hidden"
      style={{ borderColor: 'var(--room-border)', backgroundColor: 'var(--room-bg)' }}
    >
      <button
        onClick={onToggle}
        className="w-full text-left px-3 py-2.5 flex items-start gap-2 hover:opacity-90 transition-opacity"
      >
        <span className="flex-1 text-sm leading-snug" style={{ color: 'var(--room-text)' }}>
          {question.text}
        </span>
        <div className="flex items-center gap-1.5 shrink-0 mt-0.5">
          {question.status === 'answered' && (
            <CheckCircle size={12} className="text-green-400" />
          )}
          {expanded ? <ChevronUp size={12} style={{ color: 'var(--room-text-muted)' }} /> : <ChevronDown size={12} style={{ color: 'var(--room-text-muted)' }} />}
        </div>
      </button>

      {expanded && (
        <div className="px-3 pb-3 space-y-2" style={{ borderTop: '1px solid var(--room-border)' }}>
          {messages.map((m) => (
            <div key={m.id} className="pt-2">
              <p className="text-xs" style={{ color: 'var(--room-text-muted)' }}>
                {new Date(m.created_at).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })}
              </p>
              <p className="text-sm mt-0.5" style={{ color: 'var(--room-text)' }}>{m.text}</p>
            </div>
          ))}

          {isHost && (
            <div className="flex gap-2 pt-1">
              <input
                value={replyText}
                onChange={(e) => onReplyChange(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); onReply() } }}
                placeholder="Ответить..."
                className="flex-1 rounded-lg px-3 py-1.5 text-xs bg-transparent border focus:outline-none"
                style={{ borderColor: 'var(--room-border)', color: 'var(--room-text)' }}
              />
              <button
                onClick={onReply}
                disabled={!replyText.trim()}
                className="px-3 py-1.5 text-xs rounded-lg disabled:opacity-40 transition-opacity"
                style={{ backgroundColor: 'var(--room-brand-dim)', color: 'var(--room-brand)' }}
              >
                Ответить
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
