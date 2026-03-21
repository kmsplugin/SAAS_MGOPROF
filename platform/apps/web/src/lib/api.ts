import type { ApiResponse, PaginatedResponse } from '@platform/types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080/api'

class ApiError extends Error {
  constructor(public status: number, message: string, public code?: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  })

  const data = await res.json().catch(() => ({ status: 'error', message: 'Invalid JSON response' }))

  if (!res.ok) {
    throw new ApiError(res.status, data.message ?? 'Request failed', data.code)
  }

  return data as T
}

// ── Auth ──────────────────────────────────────────────────────────────────────

export const auth = {
  login: (email: string, password: string) =>
    request<{ token: string; user: Record<string, unknown> }>('/cabinet/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  me: () => request<{ user: Record<string, unknown> }>('/cabinet/me'),

  forgotPassword: (email: string) =>
    request('/cabinet/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),
}

// ── Events ────────────────────────────────────────────────────────────────────

export const events = {
  list: () => request<{ events: unknown[] }>('/events'),
  get: (id: number) => request<{ event: unknown }>(`/events/${id}`),
}

// ── Room tokens ───────────────────────────────────────────────────────────────

export const media = {
  getToken: (params: {
    roomName: string
    identity: string
    displayName: string
    role: 'host' | 'speaker' | 'moderator' | 'viewer'
  }) =>
    request<{ token: string; room_name: string; livekit_url: string }>('/media/token', {
      method: 'POST',
      body: JSON.stringify(params),
    }),
}

// ── Q&A ───────────────────────────────────────────────────────────────────────

export const questions = {
  list: () => request<{ questions: unknown[] }>('/cabinet/questions'),
  create: (eventId: number, subject: string, body: string) =>
    request('/cabinet/questions', {
      method: 'POST',
      body: JSON.stringify({ event_id: eventId, subject, body }),
    }),
  get: (id: number) => request<{ question: unknown; messages: unknown[] }>(`/cabinet/questions/${id}`),
  sendMessage: (questionId: number, body: string) =>
    request(`/cabinet/questions/${questionId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    }),
}

export { ApiError }
