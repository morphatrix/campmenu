import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Info, Pencil } from 'lucide-react'
import { api } from '../lib/api'
import { useLive } from '../context/LiveContext'
import { useAuth } from '../context/AuthContext'
import { useActiveEvent } from '../context/ActiveEventContext'
import { isStaff } from '../lib/types'
import type { Event, EventTab } from '../lib/types'
import TabBar from '../components/event/TabBar'
import MenuGrid from '../components/event/MenuGrid'
import MatrixTab from '../components/event/MatrixTab'
import ShoppingTab from '../components/event/ShoppingTab'
import LocationsTab from '../components/event/LocationsTab'
import ParticipantsPanel from '../components/event/ParticipantsPanel'
import ParticipantsReadOnly from '../components/event/ParticipantsReadOnly'
import EditEventModal from '../components/event/EditEventModal'
import VenueInfoModal, { hasVenueInfo } from '../components/event/VenueInfoModal'

interface EventResponse {
  event: Event
  effectiveParticipants: number
}

export default function EventDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { user } = useAuth()
  const { setActive } = useActiveEvent()
  const isAdmin = isStaff(user) // staff (admin or collaborator) manage events
  const [data, setData] = useState<EventResponse | null>(null)
  const [activeTab, setActiveTab] = useState<string>('')
  const [editing, setEditing] = useState(false)
  const [showVenue, setShowVenue] = useState(false)

  const load = useCallback(async () => {
    const res = await api.get<EventResponse>(`/events/${id}`)
    setData(res)
    setActiveTab((prev) => prev || res.event.tabs?.[0]?.id || '')
  }, [id])

  useEffect(() => { load() }, [load])
  useLive(load)

  // Mark this event as the active one (persists across navigation to other sections).
  useEffect(() => {
    if (data?.event) setActive({ id: data.event.id, name: data.event.name })
  }, [data?.event?.id, data?.event?.name, setActive])

  if (!data) return <p className="text-muted">{t('common.loading')}</p>

  const { event, effectiveParticipants } = data
  const tab: EventTab | undefined = event.tabs?.find((x) => x.id === activeTab)

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-2xl font-bold">{event.name}</h1>
          <p className="text-sm text-muted">
            {new Date(event.startDate).toLocaleDateString()} → {new Date(event.endDate).toLocaleDateString()} ·{' '}
            {effectiveParticipants} {t('menu.persons')}
          </p>
        </div>
        <div className="flex gap-2">
          {hasVenueInfo(event) && (
            <button className="btn-ghost" onClick={() => setShowVenue(true)}>
              <Info size={15} /> {t('venue.button')}
            </button>
          )}
          {isAdmin && (
            <button className="btn-ghost" onClick={() => setEditing(true)}>
              <Pencil size={15} /> {t('events.edit')}
            </button>
          )}
        </div>
      </div>

      {isAdmin ? <ParticipantsPanel event={event} onChange={load} /> : <ParticipantsReadOnly event={event} />}

      <TabBar
        tabs={event.tabs ?? []}
        active={activeTab}
        isAdmin={isAdmin}
        onSelect={setActiveTab}
        eventId={event.id}
        onChange={load}
      />

      <div className="mt-4">
        {tab?.kind === 'MENUS' && <MenuGrid event={event} effectiveParticipants={effectiveParticipants} />}
        {tab?.kind === 'MATRIX' && <MatrixTab tab={tab} event={event} isAdmin={isAdmin} effectiveParticipants={effectiveParticipants} onChange={load} />}
        {tab?.kind === 'LOCATIONS' && <LocationsTab event={event} isAdmin={isAdmin} effectiveParticipants={effectiveParticipants} />}
        {tab?.kind === 'SHOPPING' && <ShoppingTab event={event} />}
      </div>

      {editing && <EditEventModal event={event} onClose={() => setEditing(false)} onSaved={() => { setEditing(false); load() }} onDeleted={() => navigate('/')} />}
      {showVenue && <VenueInfoModal event={event} onClose={() => setShowVenue(false)} />}
    </div>
  )
}
