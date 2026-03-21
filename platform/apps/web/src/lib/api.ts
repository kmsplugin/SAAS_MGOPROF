const API_BASE = (process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080') + '/api/v1'

type Result<T> = { ok: true; data: T } | { ok: false; error: string }

async function request<T>(
  path: string,
  token: string | null,
  init?: RequestInit
): Promise<Result<T>> {
  try {
    const res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init?.headers,
      },
    })
    const data = await res.json().catch(() => ({ message: 'Ошибка парсинга ответа' }))
    if (!res.ok) {
      return { ok: false, error: data?.message ?? 'Ошибка запроса' }
    }
    return { ok: true, data: data as T }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : 'Сетевая ошибка' }
  }
}

export const apiClient = {
  // Auth
  register: (body: {
    email: string
    password: string
    first_name: string
    last_name: string
    tenant_slug: string
    consent_given: boolean
  }) =>
    request<{ token: string; user: Record<string, unknown> }>(
      '/auth/register', null, { method: 'POST', body: JSON.stringify(body) }
    ),

  login: (email: string, password: string, tenant_slug: string) =>
    request<{ token: string; user: Record<string, unknown> }>(
      '/auth/login', null, {
        method: 'POST',
        body: JSON.stringify({ email, password, tenant_slug }),
      }
    ),

  me: (token: string) =>
    request<{ user: Record<string, unknown> }>('/auth/me', token),

  // Events
  listEvents: (token: string, status?: string) =>
    request<{ events: Record<string, unknown>[]; total: number }>(
      `/events${status ? `?status=${status}` : ''}`, token
    ),

  getEvent: (token: string, id: string) =>
    request<{ event: Record<string, unknown> }>(`/events/${id}`, token),

  createEvent: (token: string, body: Record<string, unknown>) =>
    request<{ event: Record<string, unknown> }>(
      '/events', token, { method: 'POST', body: JSON.stringify(body) }
    ),

  publishEvent: (token: string, id: string) =>
    request<{ status: string }>(`/events/${id}/publish`, token, { method: 'POST' }),

  registerForEvent: (token: string, eventId: string) =>
    request<{ registration: Record<string, unknown> }>(
      `/events/${eventId}/register`, token, {
        method: 'POST',
        body: JSON.stringify({ event_id: eventId, consent_given: true }),
      }
    ),

  // Rooms
  listRooms: (token: string, eventId: string) =>
    request<{ rooms: Record<string, unknown>[] }>(`/events/${eventId}/rooms`, token),

  createRoom: (token: string, body: { event_id: string; mode: string; max_participants?: number }) =>
    request<{ room: Record<string, unknown> }>(
      '/rooms', token, { method: 'POST', body: JSON.stringify(body) }
    ),

  joinRoom: (token: string, roomId: string, role: string, displayName: string) =>
    request<{ join: { token: string; livekit_url: string; room_name: string } }>(
      `/rooms/${roomId}/join`, token, {
        method: 'POST',
        body: JSON.stringify({ role, display_name: displayName }),
      }
    ),

  endRoom: (token: string, roomId: string) =>
    request<{ status: string }>(`/rooms/${roomId}`, token, { method: 'DELETE' }),

  updateEvent: (token: string, id: string, body: Record<string, unknown>) =>
    request<{ event: Record<string, unknown> }>(
      `/events/${id}`, token, { method: 'PUT', body: JSON.stringify(body) }
    ),

  // Q&A
  listQuestions: (token: string, eventId: string) =>
    request<{ questions: Record<string, unknown>[] }>(`/events/${eventId}/questions`, token),

  createQuestion: (token: string, eventId: string, text: string) =>
    request<{ question: Record<string, unknown> }>(
      `/events/${eventId}/questions`, token, {
        method: 'POST',
        body: JSON.stringify({ text }),
      }
    ),

  listMessages: (token: string, eventId: string, questionId: string) =>
    request<{ messages: Record<string, unknown>[] }>(
      `/events/${eventId}/questions/${questionId}/messages`, token
    ),

  addMessage: (token: string, eventId: string, questionId: string, text: string) =>
    request<{ message: Record<string, unknown> }>(
      `/events/${eventId}/questions/${questionId}/messages`, token, {
        method: 'POST',
        body: JSON.stringify({ text }),
      }
    ),

  // AI
  getAISummaries: (token: string, eventId: string) =>
    request<{ summaries: Record<string, Record<string, unknown>> }>(
      `/events/${eventId}/ai`, token
    ),
}
