import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '—'
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  }).format(new Date(dateStr))
}

export function eventStatusLabel(status: string): string {
  const map: Record<string, string> = {
    draft: 'Черновик',
    published: 'Опубликовано',
    live: 'В эфире',
    ended: 'Завершено',
    archived: 'Архив',
  }
  return map[status] ?? status
}

export function eventTypeLabel(type: string): string {
  const map: Record<string, string> = {
    webinar: 'Вебинар',
    conference: 'Конференция',
    broadcast: 'Трансляция',
    hybrid: 'Гибридный',
    meeting: 'Встреча',
  }
  return map[type] ?? type
}
