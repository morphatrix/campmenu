import { useEffect, useRef } from 'react'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

// Tiles are plain <img> requests, which the CSP allows from any https host —
// unlike fetch/XHR (connect-src 'self'), which is why geocoding and elevation
// go through our own /api/geo/* proxy instead.
const BASE_PLAN = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
const BASE_SAT = 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}'
const OVERLAY_PISTES = 'https://www.opensnowmap.org/tiles-pistes/{z}/{x}/{y}.png'

// A pure-HTML marker: Leaflet's default icon points at PNG files by relative
// URL, which a bundler rewrites and breaks. This has no asset to resolve.
const pin = L.divIcon({
  className: '',
  html: `<svg width="30" height="40" viewBox="0 0 30 40" xmlns="http://www.w3.org/2000/svg">
    <path d="M15 0C7 0 .8 6.2.8 14c0 10 12 24.4 13 25.5a1.6 1.6 0 0 0 2.4 0c1-1.1 13-15.5 13-25.5C29.2 6.2 23 0 15 0z"
      fill="#e11d48" stroke="#fff" stroke-width="2"/>
    <circle cx="15" cy="14" r="5" fill="#fff"/>
  </svg>`,
  iconSize: [30, 40],
  iconAnchor: [15, 40],
})

export interface MapPoint {
  lat: number
  lon: number
}

export default function LocationMap({
  value,
  onPick,
  focus,
  height = 340,
}: {
  value: MapPoint | null
  /** Omit to render read-only (no click-to-place). */
  onPick?: (p: MapPoint) => void
  /** Bump `nonce` to recentre the map (e.g. after an address search). */
  focus?: { lat: number; lon: number; zoom?: number; nonce: number } | null
  height?: number
}) {
  const boxRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<L.Map | null>(null)
  const markerRef = useRef<L.Marker | null>(null)
  // Kept in a ref so the map is built once: rebinding the handler must not
  // tear down and recreate the map on every parent render.
  const onPickRef = useRef(onPick)
  onPickRef.current = onPick

  useEffect(() => {
    if (!boxRef.current || mapRef.current) return
    const map = L.map(boxRef.current, { scrollWheelZoom: true }).setView(
      value ? [value.lat, value.lon] : [45.9, 6.87], // Mont-Blanc area as a neutral default
      value ? 15 : 8,
    )
    mapRef.current = map

    const plan = L.tileLayer(BASE_PLAN, {
      maxZoom: 19,
      attribution: '&copy; OpenStreetMap',
    }).addTo(map)
    const sat = L.tileLayer(BASE_SAT, {
      maxZoom: 19,
      attribution: 'Imagery &copy; Esri',
    })
    const pistes = L.tileLayer(OVERLAY_PISTES, {
      maxZoom: 18,
      opacity: 0.9,
      attribution: 'Pistes &copy; OpenSnowMap',
    }).addTo(map)

    L.control.layers({ Plan: plan, Satellite: sat }, { 'Pistes & remontées': pistes }).addTo(map)
    L.control.scale({ imperial: false }).addTo(map)

    if (value) markerRef.current = L.marker([value.lat, value.lon], { icon: pin }).addTo(map)

    map.on('click', (e: L.LeafletMouseEvent) => {
      const handler = onPickRef.current
      if (!handler) return
      handler({ lat: +e.latlng.lat.toFixed(6), lon: +e.latlng.lng.toFixed(6) })
    })

    // The map is often mounted inside a modal that sizes after paint; without
    // this Leaflet renders a single tile in the corner.
    const ro = new ResizeObserver(() => map.invalidateSize())
    ro.observe(boxRef.current)
    setTimeout(() => map.invalidateSize(), 0)

    return () => {
      ro.disconnect()
      map.remove()
      mapRef.current = null
      markerRef.current = null
    }
  }, [])

  // Keep the marker in sync with the value owned by the parent.
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    if (!value) {
      markerRef.current?.remove()
      markerRef.current = null
      return
    }
    if (markerRef.current) markerRef.current.setLatLng([value.lat, value.lon])
    else markerRef.current = L.marker([value.lat, value.lon], { icon: pin }).addTo(map)
  }, [value?.lat, value?.lon])

  useEffect(() => {
    if (!focus || !mapRef.current) return
    mapRef.current.setView([focus.lat, focus.lon], focus.zoom ?? 16)
  }, [focus?.nonce])

  return <div ref={boxRef} style={{ height }} className="w-full overflow-hidden rounded-lg border border-border" />
}
