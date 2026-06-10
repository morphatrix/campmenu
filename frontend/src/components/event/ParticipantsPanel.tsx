import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { UserPlus, X, Users } from 'lucide-react'
import { api } from '../../lib/api'
import { displayName } from '../../lib/types'
import type { Event, User } from '../../lib/types'

export default function ParticipantsPanel({ event, onChange }: { event: Event; onChange: () => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [users, setUsers] = useState<User[]>([])
  const participants = event.participants ?? []
  const participantIds = new Set(participants.map((p) => p.userId))

  useEffect(() => {
    if (open && users.length === 0) api.get<User[]>('/users').then(setUsers)
  }, [open])

  async function add(userId: string) {
    await api.post(`/events/${event.id}/participants`, { userId })
    onChange()
  }
  async function remove(userId: string) {
    await api.del(`/events/${event.id}/participants/${userId}`)
    onChange()
  }
  async function toggleCounted(userId: string, counted: boolean) {
    await api.patch(`/events/${event.id}/participants/${userId}`, { counted })
    onChange()
  }

  return (
    <div className="card mb-4 p-4">
      <button className="flex items-center gap-2 text-sm font-semibold" onClick={() => setOpen((v) => !v)}>
        <Users size={16} /> {t('events.participants')} ({participants.length})
      </button>
      {open && (
        <div className="mt-3 grid gap-4 sm:grid-cols-2">
          <div>
            <p className="mb-2 text-xs font-semibold uppercase text-muted">{t('events.participants')}</p>
            <ul className="space-y-1">
              {participants.map((p) => (
                <li key={p.id} className="flex items-center justify-between rounded-lg bg-surface px-2 py-1 text-sm">
                  <label className="flex items-center gap-2">
                    <input type="checkbox" checked={p.counted} onChange={(e) => toggleCounted(p.userId, e.target.checked)} />
                    {displayName(p.user)}
                  </label>
                  <button onClick={() => remove(p.userId)} className="text-danger"><X size={14} /></button>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <p className="mb-2 text-xs font-semibold uppercase text-muted">+</p>
            <ul className="space-y-1">
              {users.filter((u) => !participantIds.has(u.id)).map((u) => (
                <li key={u.id} className="flex items-center justify-between rounded-lg px-2 py-1 text-sm">
                  <span>{displayName(u)} <span className="text-muted">· {u.email}</span></span>
                  <button onClick={() => add(u.id)} className="text-brand"><UserPlus size={14} /></button>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  )
}
