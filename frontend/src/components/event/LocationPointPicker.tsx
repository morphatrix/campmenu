import { lazy, Suspense, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, MapPin, Mountain, Search, X } from 'lucide-react'
import { api } from '../../lib/api'
// Leaflet is ~150 KB: only pull it once a map is actually on screen.
const LocationMap = lazy(() => import('../LocationMap'))

function MapFallback({ height = 340 }: { height?: number }) {
  return (
    <div style={{ height }} className="grid w-full place-items-center rounded-lg border border-border text-muted">
      <Loader2 size={20} className="animate-spin" />
    </div>
  )
}

export interface LocationPoint {
  lat: number
  lon: number
  alt: number | null
}

interface GeoResult {
  label: string
  lat: number
  lon: number
}

// Lets an organizer paste an address to jump the map to the right valley, then
// click the exact roof. Altitude is resolved server-side and stored with the
// point so the card can show it without re-querying.
export default function LocationPointPicker({
  value,
  onChange,
}: {
  value: LocationPoint | null
  onChange: (p: LocationPoint | null) => void
}) {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<GeoResult[]>([])
  const [searching, setSearching] = useState(false)
  const [error, setError] = useState('')
  const [focus, setFocus] = useState<{ lat: number; lon: number; zoom?: number; nonce: number } | null>(null)

  async function search() {
    const q = query.trim()
    if (!q) return
    setSearching(true)
    setError('')
    try {
      const res = await api.get<GeoResult[]>(`/geo/search?q=${encodeURIComponent(q)}`)
      setResults(res)
      if (res.length === 0) setError(t('locations.noGeoResult'))
      else setFocus({ lat: res[0].lat, lon: res[0].lon, zoom: 14, nonce: Date.now() })
    } catch {
      setError(t('locations.geoFailed'))
    } finally {
      setSearching(false)
    }
  }

  async function pick(p: { lat: number; lon: number }) {
    // Show the pin immediately; the altitude lands a moment later.
    onChange({ ...p, alt: null })
    try {
      const res = await api.get<{ elevation: number }>(`/geo/elevation?lat=${p.lat}&lon=${p.lon}`)
      onChange({ ...p, alt: Math.round(res.elevation) })
    } catch {
      /* altitude is a bonus — a failed lookup must not lose the point */
    }
  }

  return (
    <div>
      <label className="label">{t('locations.mapPoint')}</label>
      <p className="mb-2 text-xs text-muted">{t('locations.mapHint')}</p>

      <div className="mb-2 flex flex-wrap gap-2">
        <div className="relative min-w-56 flex-1">
          <Search size={15} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
          <input
            className="input pl-8"
            placeholder={t('locations.searchAddress')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); search() } }}
          />
        </div>
        <button type="button" className="btn-ghost" onClick={search} disabled={searching || !query.trim()}>
          {searching ? <Loader2 size={15} className="animate-spin" /> : <Search size={15} />} {t('locations.searchGo')}
        </button>
      </div>

      {error && <p className="mb-2 text-xs text-danger">{error}</p>}

      {results.length > 0 && (
        <ul className="mb-2 max-h-32 overflow-y-auto rounded-lg border border-border">
          {results.map((r, i) => (
            <li key={i}>
              <button
                type="button"
                className="block w-full px-3 py-1.5 text-left text-xs hover:bg-surface"
                onClick={() => setFocus({ lat: r.lat, lon: r.lon, zoom: 16, nonce: Date.now() })}
              >
                <MapPin size={11} className="mr-1 inline text-brand" />{r.label}
              </button>
            </li>
          ))}
        </ul>
      )}

      <Suspense fallback={<MapFallback />}>
        <LocationMap value={value} onPick={pick} focus={focus} />
      </Suspense>

      <div className="mt-2 flex flex-wrap items-center gap-3 text-sm">
        {value ? (
          <>
            <span className="inline-flex items-center gap-1 text-muted">
              <MapPin size={14} className="text-brand" />
              {value.lat.toFixed(5)}, {value.lon.toFixed(5)}
            </span>
            {value.alt != null && (
              <span className="inline-flex items-center gap-1 text-muted">
                <Mountain size={14} className="text-brand" /> {value.alt} m
              </span>
            )}
            <button type="button" className="btn-ghost text-xs" onClick={() => onChange(null)}>
              <X size={13} /> {t('locations.clearPoint')}
            </button>
          </>
        ) : (
          <span className="text-xs text-muted">{t('locations.noPoint')}</span>
        )}
      </div>
    </div>
  )
}
