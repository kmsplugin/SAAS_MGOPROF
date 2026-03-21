'use client'

import { useEffect, useState } from 'react'
import { useRouter, useParams, useSearchParams } from 'next/navigation'
import { useAuthStore } from '@/lib/store'
import { apiClient } from '@/lib/api'
import dynamic from 'next/dynamic'

// Dynamic import to avoid SSR issues with LiveKit
const EventRoom = dynamic(
  () => import('@/components/room/event-room').then((m) => m.EventRoom),
  { ssr: false, loading: () => <RoomLoading /> }
)

export default function RoomPage() {
  const router = useRouter()
  const params = useParams()
  const searchParams = useSearchParams()
  const eventId = params.id as string
  const roomId = searchParams.get('room') ?? ''

  const { token, user } = useAuthStore()
  const [joinData, setJoinData] = useState<{ token: string; url: string; roomName: string } | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!token) { router.push('/login'); return }

    const role = resolveRole(user?.role ?? 'participant')
    const displayName = user ? `${user.first_name} ${user.last_name}`.trim() : 'Участник'

    apiClient.joinRoom(token, roomId, role, displayName).then((res) => {
      if (!res.ok) {
        setError(res.error ?? 'Не удалось подключиться к комнате')
      } else {
        setJoinData({
          token: res.data.join.token,
          url: res.data.join.livekit_url,
          roomName: res.data.join.room_name,
        })
      }
      setLoading(false)
    })
  }, [token, roomId, user, router])

  if (!token) return null

  if (loading) return <RoomLoading />

  if (error) {
    return (
      <div className="min-h-screen bg-room-bg flex items-center justify-center text-white">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button onClick={() => router.back()} className="text-gray-400 hover:text-white text-sm underline">
            Назад
          </button>
        </div>
      </div>
    )
  }

  if (!joinData) return null

  const displayName = user ? `${user.first_name} ${user.last_name}`.trim() : 'Участник'

  return (
    <EventRoom
      token={joinData.token}
      serverUrl={joinData.url}
      roomName={joinData.roomName}
      participantRole={resolveRole(user?.role ?? 'participant')}
      displayName={displayName}
      eventId={eventId}
      onLeave={() => router.push(`/events/${eventId}`)}
    />
  )
}

function resolveRole(userRole: string): 'host' | 'speaker' | 'moderator' | 'viewer' {
  if (userRole === 'tenant_owner' || userRole === 'super_admin') return 'host'
  if (userRole === 'event_admin') return 'moderator'
  if (userRole === 'speaker') return 'speaker'
  return 'viewer'
}

function RoomLoading() {
  return (
    <div className="min-h-screen bg-room-bg flex items-center justify-center">
      <div className="text-center text-white">
        <div className="w-10 h-10 border-2 border-brand border-t-transparent rounded-full animate-spin mx-auto mb-4" />
        <p className="text-gray-400">Подключение к эфиру...</p>
      </div>
    </div>
  )
}
