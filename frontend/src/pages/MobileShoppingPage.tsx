import { FormEvent, ReactNode, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarDays, ChevronRight, Eye, EyeOff, LogOut, ShoppingCart, SlidersHorizontal, Tent, Users } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import { useLive } from '../context/LiveContext'
import { displayName } from '../lib/types'
import type { Event, EventParticipant, ShoppingLine, User } from '../lib/types'
import Avatar from '../components/Avatar'
import UserInfoModal from '../components/UserInfoModal'
import IbanRequestsBell from '../components/IbanRequestsBell'

const STANDARD = ['Drive', 'Station']

// Focused mobile experience (PWA start page): sign in → pick an event → only
// the shopping list of that event.
export default function MobileShoppingPage() {
  const { user, loading, logout } = useAuth()
  const [eventId, setEventId] = useState<string | null>(null)

  if (loading) return <div className="grid min-h-screen place-items-center text-muted">…</div>
  if (!user) return <MobileLogin />
  if (!eventId) return <EventPicker onPick={setEventId} onLogout={logout} />
  return <MobileShopping eventId={eventId} onBack={() => setEventId(null)} onLogout={logout} />
}

function MobileLogin() {
  const { t } = useTranslation()
  const { login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(email, password)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erreur')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="card w-full max-w-sm p-8">
        <div className="mb-6 flex flex-col items-center gap-2">
          <Tent className="text-brand" size={36} />
          <h1 className="text-xl font-bold">CampMenu</h1>
          <p className="text-center text-sm text-muted">{t('shopping.title')}</p>
        </div>
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="label">{t('auth.email')}</label>
            <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
          </div>
          <div>
            <label className="label">{t('auth.password')}</label>
            <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
          <button className="btn-primary w-full" disabled={busy}>{t('auth.signIn')}</button>
        </form>
        <div className="mt-4 text-center">
          <Link to="/forgot" className="text-sm text-brand hover:underline">{t('auth.forgot')}</Link>
        </div>
      </div>
    </div>
  )
}

function MobileHeader({ title, onLogout, right }: { title: string; onLogout: () => void; right?: ReactNode }) {
  return (
    <header className="sticky top-0 z-20 flex items-center justify-between gap-2 border-b border-border bg-card/90 px-4 py-3 backdrop-blur" style={{ paddingTop: 'max(0.75rem, env(safe-area-inset-top))' }}>
      <div className="flex min-w-0 items-center gap-2 font-semibold text-brand">
        <ShoppingCart size={20} className="shrink-0" />
        <span className="truncate">{title}</span>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {right}
        <IbanRequestsBell />
        <button onClick={onLogout} className="rounded-lg px-2 py-1 text-muted hover:text-danger"><LogOut size={18} /></button>
      </div>
    </header>
  )
}

function EventPicker({ onPick, onLogout }: { onPick: (id: string) => void; onLogout: () => void }) {
  const { t } = useTranslation()
  const [events, setEvents] = useState<Event[]>([])
  useEffect(() => { api.get<Event[]>('/events').then(setEvents) }, [])

  return (
    <div className="min-h-screen">
      <MobileHeader title="CampMenu" onLogout={onLogout} />
      <main className="mx-auto max-w-md px-4 py-6">
        <h2 className="mb-4 text-lg font-bold">{t('mobile.chooseEvent')}</h2>
        {events.length === 0 ? (
          <p className="text-muted">{t('mobile.noEvents')}</p>
        ) : (
          <ul className="space-y-2">
            {events.map((ev) => (
              <li key={ev.id}>
                <button onClick={() => onPick(ev.id)} className="card flex w-full items-center justify-between gap-2 p-4 text-left transition active:scale-[.99]">
                  <span className="min-w-0">
                    <span className="block truncate font-semibold">{ev.name}</span>
                    <span className="flex items-center gap-1 text-xs text-muted"><CalendarDays size={12} /> {new Date(ev.startDate).toLocaleDateString()} → {new Date(ev.endDate).toLocaleDateString()}</span>
                  </span>
                  <ChevronRight size={20} className="shrink-0 text-muted" />
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="mt-6 text-center">
          <Link to="/" className="text-sm text-brand hover:underline">{t('mobile.fullSite')}</Link>
        </div>
      </main>
    </div>
  )
}

type ShoppingPatch = Partial<ShoppingLine> & { clearBroughtBy?: boolean }

function MobileShopping({ eventId, onBack, onLogout }: { eventId: string; onBack: () => void; onLogout: () => void }) {
  const { t } = useTranslation()
  const [event, setEvent] = useState<Event | null>(null)
  const [lines, setLines] = useState<ShoppingLine[]>([])
  const [tab, setTab] = useState<'courses' | 'participants'>('courses')
  const [showFilters, setShowFilters] = useState(false)
  const [hideBought, setHideBought] = useState(false)
  const [hiddenSections, setHiddenSections] = useState<Set<string>>(new Set())
  const [broughtBy, setBroughtBy] = useState('') // '' = all, 'none' = unassigned, else userId

  async function loadEvent() {
    const res = await api.get<{ event: Event }>(`/events/${eventId}`)
    setEvent(res.event)
  }
  async function loadLines() {
    const res = await api.get<ShoppingLine[]>(`/events/${eventId}/shopping`)
    res.sort((a, b) => Number(a.bought) - Number(b.bought) || a.name.localeCompare(b.name))
    setLines(res)
  }
  useEffect(() => { loadEvent(); loadLines() }, [eventId])
  useLive(loadLines)

  async function update(line: ShoppingLine, patch: ShoppingPatch) {
    setLines((ls) => ls.map((l) => (l === line ? { ...l, ...patch } : l)))
    await api.patch(`/events/${eventId}/shopping`, {
      section: line.section, name: line.name, unit: line.unit, ingredientId: line.ingredientId ?? null, ...patch,
    })
  }

  const participants = (event?.participants ?? []).filter((p) => p.user)
  const allSections = useMemo(
    () => [...new Set(lines.map((l) => l.section || ''))].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b))),
    [lines],
  )

  const filtered = lines.filter((l) => {
    if (hideBought && l.bought) return false
    if (hiddenSections.has(l.section || '')) return false
    if (broughtBy === 'none' && l.broughtBy) return false
    if (broughtBy && broughtBy !== 'none' && l.broughtBy !== broughtBy) return false
    return true
  })

  const groups = useMemo(() => {
    const map = new Map<string, ShoppingLine[]>()
    for (const l of filtered) {
      const key = l.section || ''
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(l)
    }
    return [...map.entries()].sort((a, b) => (a[0] === '' ? -1 : b[0] === '' ? 1 : a[0].localeCompare(b[0])))
  }, [filtered])

  const boughtCount = lines.filter((l) => l.bought).length
  const activeFilters = (hideBought ? 1 : 0) + hiddenSections.size + (broughtBy ? 1 : 0)

  function toggleSection(key: string) {
    setHiddenSections((s) => { const n = new Set(s); n.has(key) ? n.delete(key) : n.add(key); return n })
  }
  function resetFilters() { setHideBought(false); setHiddenSections(new Set()); setBroughtBy('') }

  return (
    <div className="min-h-screen pb-24">
      <header className="sticky top-0 z-20 border-b border-border bg-card/95 backdrop-blur" style={{ paddingTop: 'env(safe-area-inset-top)' }}>
        <div className="flex items-center justify-between gap-2 px-4 py-3">
          <div className="flex min-w-0 items-center gap-2 font-semibold text-brand">
            <ShoppingCart size={20} className="shrink-0" />
            <span className="truncate">{event?.name ?? t('shopping.title')}</span>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <button onClick={onBack} className="rounded-lg px-2 py-1 text-xs text-muted hover:text-fg">{t('mobile.back')}</button>
            <IbanRequestsBell />
            <button onClick={onLogout} className="rounded-lg px-2 py-1 text-muted hover:text-danger"><LogOut size={18} /></button>
          </div>
        </div>

        {tab === 'courses' && (
        <>
        {/* Quick filters + progress */}
        <div className="flex items-center gap-2 px-3 pb-2">
          <button
            onClick={() => setHideBought((v) => !v)}
            className={`inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium ${hideBought ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted'}`}
          >
            {hideBought ? <EyeOff size={13} /> : <Eye size={13} />} {t('mobile.hideBought')}
          </button>
          <button
            onClick={() => setShowFilters((v) => !v)}
            className={`inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium ${showFilters || activeFilters ? 'border-brand text-brand' : 'border-border bg-surface text-muted'}`}
          >
            <SlidersHorizontal size={13} /> {t('mobile.filters')}{activeFilters ? ` (${activeFilters})` : ''}
          </button>
          <span className="ml-auto text-xs text-muted">{t('mobile.bought', { n: boughtCount, total: lines.length })}</span>
        </div>
        {lines.length > 0 && (
          <div className="h-1 w-full bg-surface">
            <div className="h-1 bg-brand transition-all" style={{ width: `${Math.round((boughtCount / lines.length) * 100)}%` }} />
          </div>
        )}

        {/* Filter panel */}
        {showFilters && (
          <div className="space-y-3 border-t border-border px-3 py-3">
            {allSections.length > 1 && (
              <div>
                <p className="mb-1 text-xs font-semibold uppercase text-muted">{t('mobile.sections')}</p>
                <div className="flex flex-wrap gap-1.5">
                  {allSections.map((s) => {
                    const shown = !hiddenSections.has(s)
                    return (
                      <button key={s || '__g'} onClick={() => toggleSection(s)}
                        className={`rounded-full border px-2.5 py-0.5 text-xs ${shown ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted line-through'}`}>
                        {s || t('shopping.general')}
                      </button>
                    )
                  })}
                </div>
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold uppercase text-muted">{t('mobile.broughtBy')}</span>
              <select className="input h-8 flex-1 py-1 text-sm" value={broughtBy} onChange={(e) => setBroughtBy(e.target.value)}>
                <option value="">{t('mobile.all')}</option>
                <option value="none">{t('mobile.unassigned')}</option>
                {participants.map((p) => <option key={p.id} value={p.userId}>{displayName(p.user)}</option>)}
              </select>
              {activeFilters > 0 && (
                <button onClick={resetFilters} className="rounded-lg px-2 py-1 text-xs text-brand">{t('mobile.reset')}</button>
              )}
            </div>
          </div>
        )}
        </>
        )}
      </header>

      <main className="mx-auto max-w-md px-3 py-4">
        {tab === 'participants' ? (
          <MobileParticipants participants={participants} />
        ) : filtered.length === 0 ? (
          <p className="text-center text-muted">{lines.length === 0 ? t('shopping.empty') : '—'}</p>
        ) : (
          <div className="space-y-5">
            {groups.map(([section, items]) => (
              <section key={section || '__general__'}>
                <h3 className="mb-2 px-1 text-sm font-semibold text-muted">{section || t('shopping.general')}</h3>
                <div className="space-y-2">
                  {items.map((line, i) => (
                    <MobileItem key={`${section}|${line.name}|${line.unit}|${i}`} line={line} participants={participants} onUpdate={(p) => update(line, p)} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </main>

      {/* Bottom tab bar: Courses / Participants */}
      <nav className="fixed inset-x-0 bottom-0 z-20 flex border-t border-border bg-card/95 backdrop-blur" style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}>
        <button onClick={() => setTab('courses')}
          className={`flex flex-1 flex-col items-center gap-0.5 py-2 text-xs font-medium ${tab === 'courses' ? 'text-brand' : 'text-muted'}`}>
          <ShoppingCart size={20} /> {t('mobile.tabShopping')}
        </button>
        <button onClick={() => setTab('participants')}
          className={`flex flex-1 flex-col items-center gap-0.5 py-2 text-xs font-medium ${tab === 'participants' ? 'text-brand' : 'text-muted'}`}>
          <Users size={20} /> {t('mobile.tabParticipants')}
        </button>
      </nav>
    </div>
  )
}

function MobileParticipants({ participants }: { participants: EventParticipant[] }) {
  const { t } = useTranslation()
  const [info, setInfo] = useState<User | null>(null)
  if (participants.length === 0) return <p className="text-center text-muted">{t('mobile.noParticipants')}</p>
  return (
    <>
      <ul className="space-y-2">
        {participants.map((p) => {
          const u = p.user!
          const fullName = `${u.firstName ?? ''} ${u.lastName ?? ''}`.trim() || u.email
          return (
            <li key={p.id}>
              <button
                onClick={() => setInfo(u)}
                className="card flex w-full items-center gap-3 p-3 text-left transition active:scale-[.99]"
              >
                <Avatar user={u} size={44} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">{fullName}</span>
                  {u.nickname && <span className="block truncate text-xs text-muted">« {u.nickname} »</span>}
                </span>
                <ChevronRight size={18} className="shrink-0 text-muted" />
              </button>
            </li>
          )
        })}
      </ul>
      {info && <UserInfoModal user={info} onClose={() => setInfo(null)} />}
    </>
  )
}

function MobileItem({ line, participants, onUpdate }: { line: ShoppingLine; participants: EventParticipant[]; onUpdate: (p: ShoppingPatch) => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  function currentMode(): string {
    if (line.broughtBy) return `user:${line.broughtBy}`
    if (STANDARD.includes(line.source)) return line.source
    if (line.source) return '__other__'
    return ''
  }
  const [mode, setMode] = useState(currentMode())
  const [otherText, setOtherText] = useState(line.broughtBy || STANDARD.includes(line.source) ? '' : line.source)

  function onSelect(value: string) {
    setMode(value)
    if (value === '') onUpdate({ source: '', clearBroughtBy: true })
    else if (STANDARD.includes(value)) onUpdate({ source: value, clearBroughtBy: true })
    else if (value.startsWith('user:')) onUpdate({ broughtBy: value.slice(5), source: '' })
  }

  const supplyLabel = line.broughtBy
    ? t('shopping.broughtByName', { name: displayName(participants.find((p) => p.userId === line.broughtBy)?.user) })
    : line.source

  return (
    <div className={`card p-3 ${line.bought ? 'opacity-60' : ''}`}>
      <div className="flex items-center gap-3">
        <input type="checkbox" className="h-6 w-6 shrink-0 accent-brand" checked={line.bought} onChange={(e) => onUpdate({ bought: e.target.checked })} />
        <button className="min-w-0 flex-1 text-left" onClick={() => setOpen((v) => !v)}>
          <span className={`block font-medium ${line.bought ? 'line-through' : ''}`}>{line.name}</span>
          <span className="text-xs text-muted">{line.quantity} {line.unit}{supplyLabel ? ` · ${supplyLabel}` : ''}{line.observation ? ` · ${line.observation}` : ''}</span>
        </button>
        <ChevronRight size={18} className={`shrink-0 text-muted transition ${open ? 'rotate-90' : ''}`} onClick={() => setOpen((v) => !v)} />
      </div>
      {open && (
        <div className="mt-3 space-y-2 border-t border-border pt-3">
          <select className="input" value={mode} onChange={(e) => onSelect(e.target.value)}>
            <option value="">{t('shopping.supply')} —</option>
            {STANDARD.map((s) => <option key={s} value={s}>{s}</option>)}
            {participants.map((p) => <option key={p.id} value={`user:${p.userId}`}>{t('shopping.broughtByName', { name: displayName(p.user) })}</option>)}
            <option value="__other__">{t('shopping.other')}</option>
          </select>
          {mode === '__other__' && (
            <input className="input" placeholder={t('shopping.otherPlaceholder')} defaultValue={otherText}
              onChange={(e) => setOtherText(e.target.value)} onBlur={(e) => onUpdate({ source: e.target.value, clearBroughtBy: true })} />
          )}
          <input className="input" placeholder={t('shopping.observation')} defaultValue={line.observation} onBlur={(e) => onUpdate({ observation: e.target.value })} />
        </div>
      )}
    </div>
  )
}
