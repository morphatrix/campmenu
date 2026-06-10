import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarDays, Info, Plus, Users } from 'lucide-react'
import { api, resolveAsset } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import ImageUpload from '../components/ImageUpload'
import VenueInfoModal, { hasVenueInfo } from '../components/event/VenueInfoModal'
import { isStaff } from '../lib/types'
import type { Event } from '../lib/types'

function dayCount(start: string, end: string): number {
  const ms = new Date(end).getTime() - new Date(start).getTime()
  return Math.max(1, Math.round(ms / 86400000) + 1)
}

export default function EventsPage() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [events, setEvents] = useState<Event[]>([])
  const [showForm, setShowForm] = useState(false)
  const [venueEvent, setVenueEvent] = useState<Event | null>(null)

  async function load() {
    setEvents(await api.get<Event[]>('/events'))
  }
  useEffect(() => { load() }, [])

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('events.title')}</h1>
        {isStaff(user) && (
          <button className="btn-primary" onClick={() => setShowForm((v) => !v)}>
            <Plus size={16} /> {t('events.create')}
          </button>
        )}
      </div>

      {showForm && <CreateEventForm onCreated={() => { setShowForm(false); load() }} />}

      {events.length === 0 ? (
        <p className="text-muted">{t('events.none')}</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {events.map((ev) => (
            <Link key={ev.id} to={`/events/${ev.id}`} className="card relative block h-44 overflow-hidden p-0 transition hover:border-brand">
              {ev.photoUrl ? (
                <img src={resolveAsset(ev.photoUrl)} alt={ev.name} className="absolute inset-0 h-full w-full object-cover" />
              ) : (
                <div className="h-full w-full bg-gradient-to-br from-brand/20 to-accent/20" />
              )}
              {hasVenueInfo(ev) && (
                <button
                  onClick={(e) => { e.preventDefault(); e.stopPropagation(); setVenueEvent(ev) }}
                  className="absolute right-2 top-2 grid h-8 w-8 place-items-center rounded-full bg-card/80 text-brand backdrop-blur hover:bg-card"
                  title={t('venue.button')}
                >
                  <Info size={16} />
                </button>
              )}
              <div className={`absolute inset-x-0 bottom-0 p-4 ${ev.photoUrl ? 'bg-card/55 backdrop-blur-md' : ''}`}>
                <h2 className="mb-1 text-lg font-semibold">{ev.name}</h2>
                <p className="flex items-center gap-2 text-sm text-muted">
                  <CalendarDays size={15} />
                  {new Date(ev.startDate).toLocaleDateString()} → {new Date(ev.endDate).toLocaleDateString()}
                </p>
                <div className="mt-2 flex gap-2">
                  <span className="chip">{t('events.days', { count: dayCount(ev.startDate, ev.endDate) })}</span>
                  <span className="chip"><Users size={12} /> {ev.initialParticipants}</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {venueEvent && <VenueInfoModal event={venueEvent} onClose={() => setVenueEvent(null)} />}
    </div>
  )
}

function CreateEventForm({ onCreated }: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [startDate, setStart] = useState('')
  const [endDate, setEnd] = useState('')
  const [participants, setParticipants] = useState(6)
  const [photoUrl, setPhotoUrl] = useState('')
  const [tabs, setTabs] = useState({ menus: true, breakfast: true, slopes: false, locations: false })

  function toggle(key: keyof typeof tabs) {
    setTabs((s) => ({ ...s, [key]: !s[key] }))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    await api.post('/events', {
      name,
      startDate: new Date(startDate).toISOString(),
      endDate: new Date(endDate).toISOString(),
      initialParticipants: participants,
      photoUrl,
      includeMenus: tabs.menus,
      includeBreakfast: tabs.breakfast,
      includeSlopes: tabs.slopes,
      includeLocations: tabs.locations,
    })
    onCreated()
  }

  const checkbox = (key: keyof typeof tabs, label: string) => (
    <label className="flex items-center gap-2 text-sm">
      <input type="checkbox" checked={tabs[key]} onChange={() => toggle(key)} /> {label}
    </label>
  )

  return (
    <form onSubmit={onSubmit} className="card mb-6 grid gap-4 p-5 sm:grid-cols-2">
      <div className="sm:col-span-2">
        <label className="label">{t('events.name')}</label>
        <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div>
        <label className="label">{t('events.start')}</label>
        <input className="input" type="date" value={startDate} onChange={(e) => setStart(e.target.value)} required />
      </div>
      <div>
        <label className="label">{t('events.end')}</label>
        <input className="input" type="date" value={endDate} onChange={(e) => setEnd(e.target.value)} required />
      </div>
      <div>
        <label className="label">{t('events.participants')}</label>
        <input className="input" type="number" min={1} value={participants} onChange={(e) => setParticipants(+e.target.value)} />
      </div>
      <div className="sm:col-span-2">
        <label className="label">{t('events.photo')}</label>
        <ImageUpload value={photoUrl} onChange={setPhotoUrl} />
      </div>
      <div className="sm:col-span-2">
        <label className="label">{t('events.tabsLabel')}</label>
        <div className="flex flex-wrap gap-4 rounded-lg border border-border p-3">
          {checkbox('menus', t('events.includeMenus'))}
          {checkbox('breakfast', t('events.includeBreakfast'))}
          {checkbox('slopes', t('events.includeSlopes'))}
          {checkbox('locations', t('events.includeLocations'))}
        </div>
      </div>
      <div className="sm:col-span-2">
        <button className="btn-primary">{t('common.save')}</button>
      </div>
    </form>
  )
}
