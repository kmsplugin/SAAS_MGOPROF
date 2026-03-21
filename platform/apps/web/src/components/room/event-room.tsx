'use client'

import {
  LiveKitRoom,
  VideoConference,
  useRoomContext,
  useParticipants,
  useTracks,
  TrackLoop,
  ParticipantTile,
  ControlBar,
  RoomAudioRenderer,
} from '@livekit/components-react'
import '@livekit/components-styles'
import { Track } from 'livekit-client'
import { useState, useCallback } from 'react'
import { Mic, MicOff, Video, VideoOff, PhoneOff, Users, MessageSquare, Hand } from 'lucide-react'
import type { ParticipantRole } from '@platform/types'
import { QAPanel } from './qa-panel'

interface EventRoomProps {
  token: string
  serverUrl: string
  roomName: string
  participantRole: ParticipantRole
  displayName: string
  eventId?: string
  onLeave?: () => void
}

/**
 * EventRoom is the core room UI for webinars, conferences and meetings.
 *
 * Layout adapts to role:
 * - host / speaker:    full VideoConference with controls
 * - moderator:         subscribe-only + moderation sidebar
 * - viewer:            subscribe-only HLS or WebRTC passive view
 */
export function EventRoom({
  token,
  serverUrl,
  roomName,
  participantRole,
  displayName,
  eventId,
  onLeave,
}: EventRoomProps) {
  const isPublisher = participantRole === 'host' || participantRole === 'speaker' || participantRole === 'co_host'

  return (
    <LiveKitRoom
      token={token}
      serverUrl={serverUrl}
      connect={true}
      video={isPublisher}
      audio={isPublisher}
      data-lk-theme="default"
      className="room-bg h-screen w-full"
      onDisconnected={onLeave}
    >
      <RoomAudioRenderer />
      {isPublisher ? (
        <PublisherView displayName={displayName} participantRole={participantRole} eventId={eventId} lkToken={token} onLeave={onLeave} />
      ) : (
        <ViewerView participantRole={participantRole} eventId={eventId} lkToken={token} onLeave={onLeave} />
      )}
    </LiveKitRoom>
  )
}

// ── Publisher view (host / speaker) ──────────────────────────────────────────

type Sidebar = 'participants' | 'qa' | null

function PublisherView({
  displayName,
  participantRole,
  eventId,
  lkToken,
  onLeave,
}: {
  displayName: string
  participantRole: ParticipantRole
  eventId?: string
  lkToken: string
  onLeave?: () => void
}) {
  const [sidebar, setSidebar] = useState<Sidebar>(null)
  const isHost = participantRole === 'host' || participantRole === 'co_host'

  function toggleSidebar(panel: Sidebar) {
    setSidebar((v) => (v === panel ? null : panel))
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div
        className="flex h-14 items-center justify-between border-b px-4"
        style={{ borderColor: 'var(--room-border)', backgroundColor: 'var(--room-surface)' }}
      >
        <span className="text-sm font-medium" style={{ color: 'var(--room-text)' }}>
          {displayName}
        </span>
        <div className="flex items-center gap-2">
          <LiveIndicator />
          <SidebarToggle
            icon={<Users size={14} />}
            label="Участники"
            active={sidebar === 'participants'}
            onClick={() => toggleSidebar('participants')}
          />
          {eventId && (
            <SidebarToggle
              icon={<MessageSquare size={14} />}
              label="Q&A"
              active={sidebar === 'qa'}
              onClick={() => toggleSidebar('qa')}
            />
          )}
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1">
          <VideoConference />
        </div>
        {sidebar === 'participants' && <ParticipantSidebar />}
        {sidebar === 'qa' && eventId && (
          <div className="w-80 border-l" style={{ borderColor: 'var(--room-border)' }}>
            <QAPanel token={lkToken} eventId={eventId} isHost={isHost} />
          </div>
        )}
      </div>
    </div>
  )
}

// ── Viewer view (moderator / viewer / guest) ──────────────────────────────────

function ViewerView({
  participantRole,
  eventId,
  lkToken,
  onLeave,
}: {
  participantRole: ParticipantRole
  eventId?: string
  lkToken: string
  onLeave?: () => void
}) {
  const [showQA, setShowQA] = useState(false)
  const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare], {
    onlySubscribed: true,
  })

  return (
    <div className="flex h-full flex-col room-bg">
      {/* Stage + optional Q&A sidebar */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex flex-1 items-center justify-center overflow-hidden p-4">
          {tracks.length > 0 ? (
            <TrackLoop tracks={tracks}>
              <ParticipantTile className="max-h-full max-w-full rounded-xl overflow-hidden" />
            </TrackLoop>
          ) : (
            <EmptyStage />
          )}
        </div>
        {showQA && eventId && (
          <div className="w-80 border-l shrink-0" style={{ borderColor: 'var(--room-border)' }}>
            <QAPanel token={lkToken} eventId={eventId} isHost={false} />
          </div>
        )}
      </div>

      {/* Minimal viewer controls */}
      <div
        className="flex h-16 items-center justify-center gap-4 border-t"
        style={{ borderColor: 'var(--room-border)', backgroundColor: 'var(--room-surface)' }}
      >
        {participantRole === 'moderator' && (
          <span
            className="rounded-full px-3 py-1 text-xs font-medium"
            style={{ backgroundColor: 'var(--room-brand-dim)', color: 'var(--room-brand)' }}
          >
            Модератор
          </span>
        )}
        {eventId && (
          <SidebarToggle
            icon={<MessageSquare size={14} />}
            label="Q&A"
            active={showQA}
            onClick={() => setShowQA((v) => !v)}
          />
        )}
        <button
          onClick={onLeave}
          className="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90"
          style={{ backgroundColor: 'var(--color-error)' }}
        >
          <PhoneOff size={16} />
          Выйти
        </button>
      </div>
    </div>
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function SidebarToggle({
  icon,
  label,
  active,
  onClick,
}: {
  icon: React.ReactNode
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs transition-colors"
      style={{
        color: active ? 'var(--room-brand)' : 'var(--room-text-muted)',
        backgroundColor: active ? 'var(--room-brand-dim)' : 'transparent',
      }}
    >
      {icon}
      {label}
    </button>
  )
}

function LiveIndicator() {
  return (
    <div className="flex items-center gap-1.5">
      <span
        className="h-2 w-2 animate-pulse rounded-full"
        style={{ backgroundColor: 'var(--color-error)' }}
      />
      <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--color-error)' }}>
        LIVE
      </span>
    </div>
  )
}

function EmptyStage() {
  return (
    <div className="flex flex-col items-center gap-3 text-center">
      <div
        className="flex h-16 w-16 items-center justify-center rounded-full"
        style={{ backgroundColor: 'var(--room-surface)' }}
      >
        <Video size={28} style={{ color: 'var(--room-text-muted)' }} />
      </div>
      <p className="text-sm" style={{ color: 'var(--room-text-muted)' }}>
        Трансляция ещё не начата
      </p>
    </div>
  )
}

function ParticipantSidebar() {
  const participants = useParticipants()

  return (
    <div
      className="w-72 overflow-y-auto border-l p-4"
      style={{ borderColor: 'var(--room-border)', backgroundColor: 'var(--room-surface)' }}
    >
      <p className="mb-3 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--room-text-muted)' }}>
        Участники ({participants.length})
      </p>
      <ul className="space-y-2">
        {participants.map((p) => (
          <li key={p.identity} className="flex items-center gap-2">
            <div
              className="flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium"
              style={{ backgroundColor: 'var(--room-brand-dim)', color: 'var(--room-brand)' }}
            >
              {(p.name ?? p.identity)[0]?.toUpperCase()}
            </div>
            <span className="text-sm truncate" style={{ color: 'var(--room-text)' }}>
              {p.name ?? p.identity}
            </span>
            <div className="ml-auto flex gap-1">
              {p.isMicrophoneEnabled ? (
                <Mic size={12} style={{ color: 'var(--room-brand)' }} />
              ) : (
                <MicOff size={12} style={{ color: 'var(--room-text-muted)' }} />
              )}
              {p.isCameraEnabled ? (
                <Video size={12} style={{ color: 'var(--room-brand)' }} />
              ) : (
                <VideoOff size={12} style={{ color: 'var(--room-text-muted)' }} />
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
