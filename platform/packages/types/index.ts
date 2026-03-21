// ── Tenant / Org ─────────────────────────────────────────────────────────────

export interface Tenant {
  id: string
  slug: string
  name: string
  plan: 'starter' | 'pro' | 'enterprise'
  logoUrl?: string
  primaryColor?: string
  customDomain?: string
  featureFlags: FeatureFlags
  createdAt: string
}

export interface FeatureFlags {
  aiEnabled: boolean
  recordingEnabled: boolean
  liveStreamEnabled: boolean
  customBranding: boolean
  whiteLabel: boolean
  advancedAnalytics: boolean
  maxParticipants: number
  maxViewers: number
}

// ── Users / Roles ─────────────────────────────────────────────────────────────

export type SystemRole =
  | 'super_admin'
  | 'tenant_owner'
  | 'event_admin'
  | 'moderator'
  | 'speaker'
  | 'support'
  | 'analyst'
  | 'viewer'
  | 'participant'
  | 'guest'

export interface User {
  id: string
  tenantId: string
  email: string
  firstName: string
  lastName: string
  avatarUrl?: string
  role: SystemRole
  isActive: boolean
  lastLoginAt?: string
  createdAt: string
}

// ── Events ───────────────────────────────────────────────────────────────────

export type EventStatus = 'draft' | 'published' | 'live' | 'ended' | 'archived'
export type EventType = 'webinar' | 'conference' | 'broadcast' | 'hybrid' | 'meeting'

export interface Event {
  id: string
  tenantId: string
  title: string
  description: string
  type: EventType
  status: EventStatus
  startAt: string
  endAt: string
  timezone: string
  coverUrl?: string
  capacity: number        // 0 = unlimited
  viewerCapacity: number  // for broadcast mode
  isPublic: boolean
  registrationRequired: boolean
  createdBy: string
  createdAt: string
}

// ── Rooms / Sessions ──────────────────────────────────────────────────────────

export type RoomStatus = 'pending' | 'active' | 'ended'
export type RoomMode = 'meeting' | 'webinar' | 'broadcast' | 'stage'

export interface Room {
  id: string
  eventId: string
  tenantId: string
  livekitRoomName: string
  mode: RoomMode
  status: RoomStatus
  maxParticipants: number
  isRecording: boolean
  hlsUrl?: string
  rtmpUrl?: string
  startedAt?: string
  endedAt?: string
}

// ── Participants ──────────────────────────────────────────────────────────────

export type ParticipantRole = 'host' | 'co_host' | 'speaker' | 'moderator' | 'viewer' | 'guest'

export interface Participant {
  id: string
  roomId: string
  userId: string
  role: ParticipantRole
  displayName: string
  joinedAt?: string
  leftAt?: string
  isAudioEnabled: boolean
  isVideoEnabled: boolean
  isHandRaised: boolean
}

// ── Registration ──────────────────────────────────────────────────────────────

export type RegistrationStatus = 'pending' | 'confirmed' | 'cancelled' | 'attended'

export interface Registration {
  id: string
  eventId: string
  userId: string
  tenantId: string
  status: RegistrationStatus
  ticketCode: string
  checkInAt?: string
  consentGiven: boolean
  consentVersion: string
  createdAt: string
}

// ── Q&A ───────────────────────────────────────────────────────────────────────

export type QuestionStatus = 'new' | 'in_progress' | 'answered' | 'closed'

export interface Question {
  id: string
  eventId: string
  userId: string
  tenantId: string
  subject: string
  status: QuestionStatus
  isPublic: boolean
  priority: number
  assignedTo?: string
  firstReplyAt?: string
  createdAt: string
}

export interface QuestionMessage {
  id: string
  questionId: string
  senderId: string
  senderRole: 'user' | 'admin' | 'moderator'
  body: string
  isRead: boolean
  createdAt: string
}

// ── Analytics ─────────────────────────────────────────────────────────────────

export interface EventAnalytics {
  eventId: string
  totalRegistrations: number
  confirmedRegistrations: number
  attendedCount: number
  peakViewers: number
  avgWatchTimeMinutes: number
  questionsTotal: number
  questionsAnswered: number
  engagementScore: number
}

// ── AI ────────────────────────────────────────────────────────────────────────

export interface AISummary {
  id: string
  eventId: string
  type: 'transcript' | 'summary' | 'highlights' | 'chapters' | 'action_items'
  content: string
  language: string
  generatedAt: string
  modelVersion: string
}

// ── API responses ─────────────────────────────────────────────────────────────

export interface ApiResponse<T> {
  data: T
  status: 'success' | 'error'
  message?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  limit: number
  hasMore: boolean
}

export interface ErrorResponse {
  status: 'error'
  code: string
  message: string
  details?: Record<string, string>
}
