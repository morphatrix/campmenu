import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { BedDouble, Bath, ChevronLeft, ChevronRight, Euro, Images, MapPin, ExternalLink, Loader2, Pencil, Plus, Sparkles, Trash2, Trophy, Phone, Vote } from 'lucide-react'
import { api, resolveAsset } from '../../lib/api'
import { useLive } from '../../context/LiveContext'
import { useAuth } from '../../context/AuthContext'
import { displayName } from '../../lib/types'
import Modal from '../Modal'
import Avatar from '../Avatar'
import ImageUpload from '../ImageUpload'
import type { Event, Location, LocationsResponse, SiteConfig, User } from '../../lib/types'

interface ImportLocationDraft {
  title?: string; address?: string; websiteUrl?: string; mapsUrl?: string
  beds?: number; singleBeds?: number; doubleBeds?: number; toilets?: number
  price?: number; phone?: string; description?: string; amenities?: string[]; images?: string[]
}

const AMENITIES = [
  'Machine à laver', 'Lave-vaisselle', 'Barbecue', 'Voiture de prêt', 'Wifi',
  'Cheminée', 'Jacuzzi', 'Parking', 'Animaux acceptés', 'Télévision', 'Terrasse', 'Vue montagne',
]

function mapsHref(loc: Location): string {
  if (loc.mapsUrl) return loc.mapsUrl
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(loc.address || loc.title)}`
}

export default function LocationsTab({ event, isAdmin, effectiveParticipants }: { event: Event; isAdmin: boolean; effectiveParticipants: number }) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [data, setData] = useState<LocationsResponse | null>(null)
  const [editing, setEditing] = useState<Location | null>(null)
  const [creating, setCreating] = useState(false)
  const [gallery, setGallery] = useState<Location | null>(null)

  async function load() {
    setData(await api.get<LocationsResponse>(`/events/${event.id}/locations`))
  }
  useEffect(() => { load() }, [event.id])
  useLive(load)

  // Resolve voter ids to participant users (for avatars, like the participants list).
  const userById = useMemo(() => {
    const m = new Map<string, User>()
    ;(event.participants ?? []).forEach((p) => { if (p.user) m.set(p.userId, p.user) })
    return m
  }, [event.participants])

  if (!data) return <p className="text-muted">{t('common.loading')}</p>

  const weights = data.voteWeights
  const locations = [...data.locations].sort((a, b) => Number(b.isWinner) - Number(a.isWinner) || b.score - a.score)

  function rankOf(locId: string): number | null {
    const entry = Object.entries(data!.myVotes).find(([, id]) => id === locId)
    return entry ? +entry[0] : null
  }

  async function applyVote(locId: string, rank: number | null) {
    const mv: Record<string, string> = { ...data!.myVotes }
    for (const k of Object.keys(mv)) if (mv[k] === locId) delete mv[k]
    if (rank) mv[String(rank)] = locId
    setData({ ...data!, myVotes: mv })
    const votes = Object.entries(mv).map(([r, id]) => ({ rank: +r, locationId: id }))
    await api.put(`/events/${event.id}/votes`, { votes })
    load()
  }

  async function promote(loc: Location) {
    await api.post(`/locations/${loc.id}/promote`)
    load()
  }
  async function remove(loc: Location) {
    if (!confirm(`${t('common.delete')} « ${loc.title} » ?`)) return
    await api.del(`/locations/${loc.id}`)
    load()
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm text-muted">{t('locations.score')} = Σ poids ({weights.join(' / ')})</p>
        <button className="btn-primary" onClick={() => setCreating(true)}><Plus size={16} /> {t('locations.add')}</button>
      </div>

      {locations.length === 0 ? (
        <p className="text-muted">{t('locations.empty')}</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {locations.map((loc) => {
            const canManage = isAdmin || loc.createdBy === user?.id
            const myRank = rankOf(loc.id)
            return (
              <div key={loc.id} className={`card overflow-hidden ${loc.isWinner ? 'ring-2 ring-success' : ''}`}>
                {loc.images.length > 0 && (
                  <button type="button" onClick={() => setGallery(loc)} className="relative block w-full">
                    <img src={resolveAsset(loc.images[0])} alt="" className="h-40 w-full object-cover" />
                    <span className="absolute bottom-2 right-2 inline-flex items-center gap-1 rounded-full bg-card/75 px-2.5 py-1 text-xs font-medium backdrop-blur">
                      <Images size={13} /> {loc.images.length}
                    </span>
                  </button>
                )}
                <div className="p-4">
                  <div className="mb-2 flex items-start justify-between gap-2">
                    <h3 className="text-lg font-semibold">
                      {loc.isWinner && <Trophy size={16} className="mr-1 inline text-success" />}
                      {loc.title}
                    </h3>
                    <span className="chip text-brand">{loc.score} pts</span>
                  </div>

                  {loc.address && (
                    <a href={mapsHref(loc)} target="_blank" rel="noreferrer" className="mb-2 inline-flex items-center gap-1 text-sm text-brand hover:underline">
                      <MapPin size={14} /> {loc.address} <ExternalLink size={11} />
                    </a>
                  )}

                  <div className="mb-2 flex flex-wrap gap-3 text-sm text-muted">
                    {loc.beds > 0 && <span className="inline-flex items-center gap-1"><BedDouble size={14} /> {loc.beds}</span>}
                    {(loc.singleBeds > 0 || loc.doubleBeds > 0) && <span>{loc.singleBeds} simple / {loc.doubleBeds} double</span>}
                    {loc.toilets > 0 && <span className="inline-flex items-center gap-1"><Bath size={14} /> {loc.toilets}</span>}
                    {loc.phone && <span className="inline-flex items-center gap-1"><Phone size={14} /> {loc.phone}</span>}
                  </div>

                  {loc.price > 0 && (
                    <div className="mb-2 inline-flex items-center gap-2 rounded-lg bg-surface px-2 py-1 text-sm">
                      <Euro size={14} className="text-brand" />
                      <span className="font-semibold">{loc.price} €</span>
                      {effectiveParticipants > 0 && (
                        <span className="text-muted">· {Math.round((loc.price / effectiveParticipants) * 100) / 100} €{t('locations.perPerson')}</span>
                      )}
                    </div>
                  )}

                  {loc.amenities.length > 0 && (
                    <div className="mb-2 flex flex-wrap gap-1">
                      {loc.amenities.map((a) => <span key={a} className="chip">{a}</span>)}
                    </div>
                  )}

                  {loc.voters && loc.voters.length > 0 && (
                    <div className="mb-2 flex flex-wrap items-center gap-1.5">
                      <span className="inline-flex items-center gap-1 text-xs text-muted">
                        <Vote size={13} /> {loc.voters.length}
                      </span>
                      {[...loc.voters].sort((a, b) => a.rank - b.rank).map((v) => (
                        <span key={v.userId} title={`${displayName(userById.get(v.userId)) || '?'} · n°${v.rank}`}>
                          <Avatar user={userById.get(v.userId)} size={22} />
                        </span>
                      ))}
                    </div>
                  )}

                  {loc.description && <p className="mb-2 line-clamp-5 whitespace-pre-wrap text-sm" title={loc.description}>{loc.description}</p>}
                  {loc.usefulInfo && <p className="mb-2 text-sm text-muted">{loc.usefulInfo}</p>}
                  {loc.observation && (
                    <p className="mb-2 rounded-lg bg-surface px-2 py-1 text-sm">
                      <span className="font-medium text-muted">{t('locations.observation')} : </span>{loc.observation}
                    </p>
                  )}

                  <div className="flex flex-wrap items-center gap-3 border-t border-border pt-3">
                    {loc.websiteUrl && (
                      <a href={loc.websiteUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-sm text-brand hover:underline">
                        <ExternalLink size={13} /> {t('locations.website')}
                      </a>
                    )}
                    <label className="ml-auto inline-flex items-center gap-1 text-sm">
                      {t('locations.myVote')}:
                      <select
                        className="input h-8 w-28 py-1"
                        value={myRank ?? ''}
                        onChange={(e) => applyVote(loc.id, e.target.value ? +e.target.value : null)}
                      >
                        <option value="">{t('locations.noVote')}</option>
                        {weights.map((wgt, i) => (
                          <option key={i} value={i + 1}>{i + 1} (×{wgt})</option>
                        ))}
                      </select>
                    </label>
                    {canManage && (
                      <>
                        <button className="btn-ghost" onClick={() => setEditing(loc)}><Pencil size={14} /></button>
                        <button className="btn-ghost text-danger" onClick={() => remove(loc)}><Trash2 size={14} /></button>
                      </>
                    )}
                    {isAdmin && (
                      <button className={`btn-ghost ${loc.isWinner ? 'text-muted' : 'text-success'}`} onClick={() => promote(loc)}>
                        <Trophy size={14} /> {loc.isWinner ? t('locations.unpromote') : t('locations.promote')}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {(creating || editing) && (
        <LocationForm
          eventId={event.id}
          initial={editing}
          showVenueInfo={isAdmin}
          participants={effectiveParticipants}
          onClose={() => { setCreating(false); setEditing(null) }}
          onSaved={() => { setCreating(false); setEditing(null); load() }}
        />
      )}

      {gallery && <GalleryModal images={gallery.images} title={gallery.title} onClose={() => setGallery(null)} />}
    </div>
  )
}

// GalleryModal shows a location's photos by URL (no download, just <img> links),
// with a large view, prev/next and a thumbnail strip.
function GalleryModal({ images, title, onClose }: { images: string[]; title: string; onClose: () => void }) {
  const [i, setI] = useState(0)
  const n = images.length
  const go = (d: number) => setI((c) => (c + d + n) % n)
  return (
    <Modal title={title} onClose={onClose} wider>
      <div className="space-y-3">
        <div className="relative">
          <img src={resolveAsset(images[i])} alt="" className="max-h-[70vh] w-full rounded-lg bg-surface object-contain" />
          {n > 1 && (
            <>
              <button onClick={() => go(-1)} className="absolute left-2 top-1/2 -translate-y-1/2 grid h-9 w-9 place-items-center rounded-full bg-card/80 backdrop-blur hover:bg-card" aria-label="prev"><ChevronLeft size={18} /></button>
              <button onClick={() => go(1)} className="absolute right-2 top-1/2 -translate-y-1/2 grid h-9 w-9 place-items-center rounded-full bg-card/80 backdrop-blur hover:bg-card" aria-label="next"><ChevronRight size={18} /></button>
              <span className="absolute bottom-2 right-2 rounded-full bg-card/75 px-2 py-0.5 text-xs backdrop-blur">{i + 1} / {n}</span>
            </>
          )}
        </div>
        {n > 1 && (
          <div className="flex gap-2 overflow-x-auto pb-1">
            {images.map((img, idx) => (
              <button key={idx} onClick={() => setI(idx)} className={`shrink-0 overflow-hidden rounded-lg border-2 ${idx === i ? 'border-brand' : 'border-transparent'}`}>
                <img src={resolveAsset(img)} alt="" className="h-16 w-24 object-cover" />
              </button>
            ))}
          </div>
        )}
      </div>
    </Modal>
  )
}

function LocationForm({
  eventId, initial, showVenueInfo, participants, onClose, onSaved,
}: {
  eventId: string; initial: Location | null; showVenueInfo: boolean; participants: number; onClose: () => void; onSaved: () => void
}) {
  const { t } = useTranslation()
  const [f, setF] = useState(() => ({
    title: initial?.title ?? '',
    address: initial?.address ?? '',
    websiteUrl: initial?.websiteUrl ?? '',
    mapsUrl: initial?.mapsUrl ?? '',
    beds: initial?.beds ?? 0,
    singleBeds: initial?.singleBeds ?? 0,
    doubleBeds: initial?.doubleBeds ?? 0,
    toilets: initial?.toilets ?? 0,
    price: initial?.price ?? 0,
    phone: initial?.phone ?? '',
    usefulInfo: initial?.usefulInfo ?? '',
    description: initial?.description ?? '',
    observation: initial?.observation ?? '',
  }))
  const [amenities, setAmenities] = useState<string[]>(initial?.amenities ?? [])
  const [images, setImages] = useState<string[]>(initial?.images?.length ? initial.images : [''])
  const [customAmenity, setCustomAmenity] = useState('')

  // AI import from a URL (only on creation, when an AI provider is configured).
  const [aiEnabled, setAiEnabled] = useState(false)
  const [importUrl, setImportUrl] = useState('')
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState(0)
  const [importError, setImportError] = useState('')

  useEffect(() => {
    if (!initial) api.get<SiteConfig>('/config').then((c) => setAiEnabled(!!c.aiEnabled)).catch(() => {})
  }, [initial])

  async function importFromUrl() {
    if (!importUrl.trim()) return
    setImporting(true)
    setImportError('')
    setProgress(8)
    const timer = setInterval(() => setProgress((p) => (p < 95 ? p + Math.max(0.4, (95 - p) * 0.04) : p)), 900)
    try {
      const res = await api.post<{ ok: boolean; draft?: ImportLocationDraft; error?: string }>('/locations/import', { url: importUrl.trim() })
      if (res.ok && res.draft) {
        const d = res.draft
        setF((prev) => ({
          ...prev,
          title: d.title || prev.title,
          address: d.address || prev.address,
          websiteUrl: d.websiteUrl || prev.websiteUrl,
          mapsUrl: d.mapsUrl || prev.mapsUrl,
          beds: d.beds || prev.beds,
          singleBeds: d.singleBeds || prev.singleBeds,
          doubleBeds: d.doubleBeds || prev.doubleBeds,
          toilets: d.toilets || prev.toilets,
          price: d.price || prev.price,
          phone: d.phone || prev.phone,
          description: d.description || prev.description,
        }))
        if (d.amenities?.length) setAmenities(d.amenities)
        if (d.images?.length) setImages(d.images)
      } else {
        setImportError(res.error ?? t('recipes.importFailed'))
      }
    } catch (e: any) {
      setImportError(e?.message ?? t('recipes.importFailed'))
    } finally {
      clearInterval(timer)
      setProgress(100)
      setTimeout(() => setImporting(false), 500)
    }
  }

  function set<K extends keyof typeof f>(k: K, v: (typeof f)[K]) { setF((s) => ({ ...s, [k]: v })) }
  function toggleAmenity(a: string) {
    setAmenities((s) => (s.includes(a) ? s.filter((x) => x !== a) : [...s, a]))
  }
  function setImage(i: number, url: string) { setImages((s) => s.map((u, idx) => (idx === i ? url : u))) }

  async function save() {
    if (!f.title.trim()) return
    const body = { ...f, amenities, images: images.filter((u) => u.trim()) }
    if (initial) await api.patch(`/locations/${initial.id}`, body)
    else await api.post(`/events/${eventId}/locations`, body)
    onSaved()
  }

  const num = (k: 'beds' | 'singleBeds' | 'doubleBeds' | 'toilets', label: string) => (
    <div>
      <label className="label">{label}</label>
      <input className="input" type="number" min={0} value={f[k] || ''} onChange={(e) => set(k, +e.target.value)} />
    </div>
  )

  return (
    <Modal title={initial ? t('locations.edit') : t('locations.add')} onClose={onClose} wide>
      <div className="space-y-4">
        {aiEnabled && !initial && (
          <div className="rounded-lg border border-dashed border-border p-3">
            <label className="label flex items-center gap-1"><Sparkles size={14} className="text-brand" /> {t('recipes.importTitle')}</label>
            <div className="flex gap-2">
              <input className="input" type="url" placeholder="https://…" value={importUrl} onChange={(e) => setImportUrl(e.target.value)} disabled={importing} />
              <button type="button" className="btn-ghost whitespace-nowrap" onClick={importFromUrl} disabled={importing || !importUrl.trim()}>
                {importing ? <Loader2 size={15} className="animate-spin" /> : <Sparkles size={15} />} {t('recipes.import')}
              </button>
            </div>
            {importing && (
              <div className="mt-3">
                <div className="h-2 w-full overflow-hidden rounded-full bg-surface">
                  <div className="h-2 rounded-full bg-brand transition-all duration-500" style={{ width: `${Math.round(progress)}%` }} />
                </div>
                <p className="mt-1 text-xs text-muted">{t('recipes.importing')} {Math.round(progress)}%</p>
              </div>
            )}
            {importError && <p className="mt-2 text-xs text-danger">{importError}</p>}
          </div>
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label">{t('locations.intitule')}</label>
            <input className="input" value={f.title} onChange={(e) => set('title', e.target.value)} required />
          </div>
          <div>
            <label className="label">{t('locations.address')}</label>
            <input className="input" value={f.address} onChange={(e) => set('address', e.target.value)} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {num('beds', t('locations.beds'))}
          {num('singleBeds', t('locations.singleBeds'))}
          {num('doubleBeds', t('locations.doubleBeds'))}
          {num('toilets', t('locations.toilets'))}
        </div>
        <div className="sm:w-1/2">
          <label className="label">{t('locations.price')}</label>
          <input className="input" type="number" min={0} step="0.01" value={f.price || ''} onChange={(e) => set('price', +e.target.value)} />
          {f.price > 0 && participants > 0 && (
            <p className="mt-1 text-xs text-muted">= {Math.round((f.price / participants) * 100) / 100} €{t('locations.perPerson')} ({participants} {t('menu.persons')})</p>
          )}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label">{t('locations.website')}</label>
            <input className="input" value={f.websiteUrl} onChange={(e) => set('websiteUrl', e.target.value)} placeholder="https://…" />
          </div>
          <div>
            <label className="label">{t('locations.maps')} (URL)</label>
            <input className="input" value={f.mapsUrl} onChange={(e) => set('mapsUrl', e.target.value)} placeholder="https://maps…" />
          </div>
        </div>

        <div>
          <label className="label">{t('locations.amenities')}</label>
          <div className="flex flex-wrap gap-2">
            {[...AMENITIES, ...amenities.filter((a) => !AMENITIES.includes(a))].map((a) => (
              <label key={a} className={`chip cursor-pointer ${amenities.includes(a) ? 'bg-brand text-brand-fg' : ''}`}>
                <input type="checkbox" className="hidden" checked={amenities.includes(a)} onChange={() => toggleAmenity(a)} />
                {a}
              </label>
            ))}
          </div>
          <div className="mt-2 flex gap-2">
            <input className="input w-48" placeholder={t('locations.addCustomAmenity')} value={customAmenity} onChange={(e) => setCustomAmenity(e.target.value)} />
            <button type="button" className="btn-ghost" onClick={() => { if (customAmenity.trim()) { toggleAmenity(customAmenity.trim()); setCustomAmenity('') } }}>
              <Plus size={15} />
            </button>
          </div>
        </div>

        <div>
          <label className="label">{t('locations.images')}</label>
          <div className="space-y-2">
            {images.map((img, i) => (
              <div key={i} className="flex items-start gap-2">
                <div className="flex-1"><ImageUpload value={img} onChange={(u) => setImage(i, u)} /></div>
                <button type="button" className="btn-ghost" onClick={() => setImages((s) => (s.length > 1 ? s.filter((_, idx) => idx !== i) : ['']))}><Trash2 size={15} /></button>
              </div>
            ))}
          </div>
          <button type="button" className="btn-ghost mt-2" onClick={() => setImages((s) => [...s, ''])}><Plus size={15} /> {t('locations.addImage')}</button>
        </div>

        <div>
          <label className="label">{t('locations.description')}</label>
          <textarea className="input min-h-24" value={f.description} onChange={(e) => set('description', e.target.value)} />
        </div>

        <div>
          <label className="label">{t('locations.observation')}</label>
          <textarea className="input min-h-16" value={f.observation} onChange={(e) => set('observation', e.target.value)} placeholder={t('locations.observationPlaceholder')} />
        </div>

        {showVenueInfo && (
          <div className="grid gap-4 rounded-lg border border-border p-3 sm:grid-cols-2">
            <p className="text-xs font-semibold uppercase text-muted sm:col-span-2">{t('locations.venueInfo')}</p>
            <div>
              <label className="label">{t('locations.phone')}</label>
              <input className="input" value={f.phone} onChange={(e) => set('phone', e.target.value)} />
            </div>
            <div>
              <label className="label">{t('locations.usefulInfo')}</label>
              <input className="input" value={f.usefulInfo} onChange={(e) => set('usefulInfo', e.target.value)} />
            </div>
          </div>
        )}

        <div className="flex justify-end gap-2">
          <button className="btn-ghost" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={save}>{t('common.save')}</button>
        </div>
      </div>
    </Modal>
  )
}
