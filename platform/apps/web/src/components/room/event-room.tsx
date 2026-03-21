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

interface EventRoomProps {
  token: string
  serverUrl: string
  roomName: string
  participantRole: ParticipantRole
  displayName: string
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
        <PublisherView displayName={displayName} onLeave={onLeave} />
      ) : (
        <ViewerView participantRole={participantRole} onLeave={onLeave} />
      )}
    </LiveKitRoom>
  )
}

// ── Publisher view (host / speaker) ──────────────────────────────────────────

function PublisherView({ displayName, onLeave }: { displayName: string; onLeave?: () => void }) {
  const [showParticipants, setShowParticipants] = useState(false)

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
          <button
            onClick={() => setShowParticipants((v) => !v)}
            className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs transition-colors"
            style={{
              color: 'var(--room-text-muted)',
              backgroundColor: showParticipants ? 'var(--room-surface-hover)' : 'transparent',
            }}
          >
            <Users size={14} />
            Участники
          </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1">
          <VideoConference />
        </div>
        {showParticipants && <ParticipantSidebar />}
      </div>
    </div>
  )
}

// ── Viewer view (moderator / viewer / guest) ──────────────────────────────────

function ViewerView({
  participantRole,
  onLeave,
}: {
  participantRole: ParticipantRole
  onLeave?: () => void
}) {
  const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare], {
    onlySubscribed: true,
  })

  return (
    <div className="flex h-full flex-col room-bg">
      {/* Stage area */}
      <div className="flex flex-1 items-center justify-center overflow-hidden p-4">
        {tracks.length > 0 ? (
          <TrackLoop tracks={tracks}>
            <ParticipantTile className="max-h-full max-w-full rounded-xl overflow-hidden" />
          </TrackLoop>
        ) : (
          <EmptyStage />
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
