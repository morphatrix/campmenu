import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { useAuth } from '../../context/AuthContext'
import { isAdmin } from '../../lib/types'
import Modal from '../Modal'
import ImageUpload from '../ImageUpload'
import type { Event } from '../../lib/types'

export default function EditEventModal({
  event, onClose, onSaved, onDeleted,
}: {
  event: Event; onClose: () => void; onSaved: () => void; onDeleted: () => void
}) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [name, setName] = useState(event.name)
  const [startDate, setStart] = useState(event.startDate.slice(0, 10))
  const [endDate, setEnd] = useState(event.endDate.slice(0, 10))
  const [participants, setParticipants] = useState(event.initialParticipants)
  const [photoUrl, setPhotoUrl] = useState(event.photoUrl)
  const [voteWeights, setVoteWeights] = useState(event.voteWeights || '3,2,1')
  const [venueAddress, setVenueAddress] = useState(event.venueAddress)
  const [venueMapsUrl, setVenueMapsUrl] = useState(event.venueMapsUrl)
  const [venuePhone, setVenuePhone] = useState(event.venuePhone)
  const [venueInfo, setVenueInfo] = useState(event.venueInfo)

  async function save() {
    await api.patch(`/events/${event.id}`, {
      name,
      startDate: new Date(startDate).toISOString(),
      endDate: new Date(endDate).toISOString(),
      initialParticipants: participants,
      photoUrl,
      voteWeights,
      venueAddress,
      venueMapsUrl,
      venuePhone,
      venueInfo,
    })
    onSaved()
  }

  async function remove() {
    if (!confirm(t('common.confirmDelete', { name: event.name }))) return
    await api.del(`/events/${event.id}`)
    onDeleted()
  }

  return (
    <Modal title={t('events.edit')} onClose={onClose} wide>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <label className="label">{t('events.name')}</label>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div>
          <label className="label">{t('events.start')}</label>
          <input className="input" type="date" value={startDate} onChange={(e) => setStart(e.target.value)} />
        </div>
        <div>
          <label className="label">{t('events.end')}</label>
          <input className="input" type="date" value={endDate} onChange={(e) => setEnd(e.target.value)} />
        </div>
        <div>
          <label className="label">{t('events.participants')}</label>
          <input className="input" type="number" min={1} value={participants} onChange={(e) => setParticipants(+e.target.value)} />
        </div>
        <div>
          <label className="label">{t('events.voteWeights')}</label>
          <input className="input" value={voteWeights} onChange={(e) => setVoteWeights(e.target.value)} placeholder="3,2,1" />
        </div>
        <div className="sm:col-span-2">
          <label className="label">{t('events.photo')}</label>
          <ImageUpload value={photoUrl} onChange={setPhotoUrl} />
        </div>

        <div className="sm:col-span-2 grid gap-4 rounded-lg border border-border p-3 sm:grid-cols-2">
          <p className="text-xs font-semibold uppercase text-muted sm:col-span-2">{t('venue.title')}</p>
          <div>
            <label className="label">{t('venue.address')}</label>
            <input className="input" value={venueAddress} onChange={(e) => setVenueAddress(e.target.value)} />
          </div>
          <div>
            <label className="label">{t('venue.maps')} (URL)</label>
            <input className="input" value={venueMapsUrl} onChange={(e) => setVenueMapsUrl(e.target.value)} placeholder="https://maps…" />
          </div>
          <div>
            <label className="label">{t('venue.phone')}</label>
            <input className="input" value={venuePhone} onChange={(e) => setVenuePhone(e.target.value)} />
          </div>
          <div className="sm:col-span-2">
            <label className="label">{t('venue.info')}</label>
            <textarea className="input min-h-20" value={venueInfo} onChange={(e) => setVenueInfo(e.target.value)} />
          </div>
        </div>
      </div>
      <div className="mt-4 flex items-center justify-between gap-2">
        {isAdmin(user) ? (
          <button className="btn-ghost text-danger" onClick={remove}>
            <Trash2 size={15} /> {t('events.deleteEvent')}
          </button>
        ) : <span />}
        <div className="flex gap-2">
          <button className="btn-ghost" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={save}>{t('common.save')}</button>
        </div>
      </div>
    </Modal>
  )
}
