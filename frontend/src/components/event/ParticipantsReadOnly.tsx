import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Users } from 'lucide-react'
import { displayName } from '../../lib/types'
import type { Event, User } from '../../lib/types'
import Avatar from '../Avatar'
import UserInfoModal from '../UserInfoModal'

// Read-only participant list shown to non-staff users.
export default function ParticipantsReadOnly({ event }: { event: Event }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [info, setInfo] = useState<User | null>(null)
  const participants = (event.participants ?? []).filter((p) => p.user)

  return (
    <div className="card mb-4 p-4">
      <button className="flex items-center gap-2 text-sm font-semibold" onClick={() => setOpen((v) => !v)}>
        <Users size={16} /> {t('events.participants')} ({participants.length})
      </button>
      {open && (
        <ul className="mt-3 flex flex-wrap gap-2">
          {participants.map((p) => (
            <li key={p.id}>
              <button className="chip flex items-center gap-1.5 hover:bg-surface" onClick={() => p.user && setInfo(p.user)}>
                <Avatar user={p.user} size={18} />
                {displayName(p.user)}
              </button>
            </li>
          ))}
        </ul>
      )}
      {info && <UserInfoModal user={info} onClose={() => setInfo(null)} />}
    </div>
  )
}
