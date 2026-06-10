import { useTranslation } from 'react-i18next'
import { MapPin, Phone, ExternalLink } from 'lucide-react'
import Modal from '../Modal'
import type { Event } from '../../lib/types'

export function hasVenueInfo(event: Event): boolean {
  return !!(event.venueAddress || event.venueMapsUrl || event.venuePhone || event.venueInfo)
}

function mapsHref(event: Event): string {
  if (event.venueMapsUrl) return event.venueMapsUrl
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(event.venueAddress)}`
}

// Read-only popup showing the event venue info (address, map, phone, description).
export default function VenueInfoModal({ event, onClose }: { event: Event; onClose: () => void }) {
  const { t } = useTranslation()
  return (
    <Modal title={`${t('venue.title')} · ${event.name}`} onClose={onClose}>
      {!hasVenueInfo(event) ? (
        <p className="text-sm text-muted">{t('venue.none')}</p>
      ) : (
        <div className="space-y-3 text-sm">
          {event.venueAddress && (
            <a href={mapsHref(event)} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-brand hover:underline">
              <MapPin size={15} /> {event.venueAddress} <ExternalLink size={11} />
            </a>
          )}
          {!event.venueAddress && event.venueMapsUrl && (
            <a href={event.venueMapsUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-brand hover:underline">
              <MapPin size={15} /> {t('venue.maps')} <ExternalLink size={11} />
            </a>
          )}
          {event.venuePhone && (
            <p className="flex items-center gap-1"><Phone size={15} /> <a href={`tel:${event.venuePhone}`} className="hover:underline">{event.venuePhone}</a></p>
          )}
          {event.venueInfo && <p className="whitespace-pre-wrap leading-relaxed">{event.venueInfo}</p>}
        </div>
      )}
    </Modal>
  )
}
