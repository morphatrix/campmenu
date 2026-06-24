import { FormEvent, ReactNode, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, CalendarDays, ChefHat, ChevronRight, Eye, EyeOff, ListPlus, LogOut, Plus, Search, ShoppingCart, SlidersHorizontal, Store, Tent, Trash2, Users } from 'lucide-react'
import { api, ApiError, resolveAsset } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import { useLive } from '../context/LiveContext'
import { displayName } from '../lib/types'
import type { Event, EventParticipant, EventTab, Recipe, ShoppingLine, TabArticle, User } from '../lib/types'
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
  const [tab, setTab] = useState<'courses' | 'lists' | 'participants' | 'recipes'>('courses')
  const [showFilters, setShowFilters] = useState(false)
  const [hideBought, setHideBought] = useState(false)
  const [hiddenSections, setHiddenSections] = useState<Set<string>>(new Set())
  const [broughtBy, setBroughtBy] = useState('') // '' = all, 'none' = unassigned, else userId
  const [listFilter, setListFilter] = useState('') // '' = all, else a source list name
  const [byAisle, setByAisle] = useState(false)
  const [aiEnabled, setAiEnabled] = useState(false)
  useEffect(() => { api.get<{ aiEnabled?: boolean }>('/config').then((c) => setAiEnabled(!!c.aiEnabled)).catch(() => {}) }, [])

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
    setLines((ls) => ls.map((l) => {
      if (l !== line) return l
      const m = { ...l, ...patch }
      if (patch.boughtQuantity !== undefined) m.bought = l.quantity > 0 && patch.boughtQuantity >= l.quantity
      return m
    }))
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
    if (listFilter && !(l.lists || []).includes(listFilter)) return false
    return true
  })

  const allLists = useMemo(() => {
    const set = new Set<string>()
    for (const l of lines) for (const n of l.lists || []) set.add(n)
    return [...set].sort((a, b) => a.localeCompare(b))
  }, [lines])

  const groups = useMemo(() => {
    const map = new Map<string, ShoppingLine[]>()
    for (const l of filtered) {
      const key = byAisle ? (l.aisle || t('shopping.otherAisle')) : (l.section || '')
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(l)
    }
    return [...map.entries()].sort((a, b) => (a[0] === '' ? -1 : b[0] === '' ? 1 : a[0].localeCompare(b[0])))
  }, [filtered, byAisle, t])

  const boughtCount = lines.filter((l) => l.bought).length
  const activeFilters = (hideBought ? 1 : 0) + hiddenSections.size + (broughtBy ? 1 : 0) + (listFilter ? 1 : 0)

  function toggleSection(key: string) {
    setHiddenSections((s) => { const n = new Set(s); n.has(key) ? n.delete(key) : n.add(key); return n })
  }
  function resetFilters() { setHideBought(false); setHiddenSections(new Set()); setBroughtBy(''); setListFilter('') }

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
          {aiEnabled && (
            <button
              onClick={() => setByAisle((v) => !v)}
              className={`inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium ${byAisle ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted'}`}
            >
              <Store size={13} /> {t('shopping.byAisle')}
            </button>
          )}
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
            {allLists.length > 0 && (
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold uppercase text-muted">{t('mobile.byList')}</span>
                <select className="input h-8 flex-1 py-1 text-sm" value={listFilter} onChange={(e) => setListFilter(e.target.value)}>
                  <option value="">{t('mobile.all')}</option>
                  {allLists.map((n) => <option key={n} value={n}>{n}</option>)}
                </select>
              </div>
            )}
          </div>
        )}
        </>
        )}
      </header>

      <main className="mx-auto max-w-md px-3 py-4">
        {tab === 'recipes' ? (
          <MobileRecipes />
        ) : tab === 'lists' ? (
          <MobileAdhocLists eventId={eventId} onChanged={loadLines} />
        ) : tab === 'participants' ? (
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
        <button onClick={() => setTab('lists')}
          className={`flex flex-1 flex-col items-center gap-0.5 py-2 text-xs font-medium ${tab === 'lists' ? 'text-brand' : 'text-muted'}`}>
          <ListPlus size={20} /> {t('mobile.tabLists')}
        </button>
        <button onClick={() => setTab('participants')}
          className={`flex flex-1 flex-col items-center gap-0.5 py-2 text-xs font-medium ${tab === 'participants' ? 'text-brand' : 'text-muted'}`}>
          <Users size={20} /> {t('mobile.tabParticipants')}
        </button>
        <button onClick={() => setTab('recipes')}
          className={`flex flex-1 flex-col items-center gap-0.5 py-2 text-xs font-medium ${tab === 'recipes' ? 'text-brand' : 'text-muted'}`}>
          <ChefHat size={20} /> {t('mobile.tabRecipes')}
        </button>
      </nav>
    </div>
  )
}

// MobileAdhocLists manages event-private "top-up" lists: free lists any
// participant can create during the event to complete the shopping. Their items
// flow straight into the shopping list and are never subject to votes.
type AdhocList = EventTab & { articles?: TabArticle[] }

function MobileAdhocLists({ eventId, onChanged }: { eventId: string; onChanged: () => void }) {
  const { t } = useTranslation()
  const [lists, setLists] = useState<AdhocList[]>([])
  const [units, setUnits] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [newName, setNewName] = useState('')

  async function load() {
    const res = await api.get<AdhocList[]>(`/events/${eventId}/adhoc-lists`)
    setLists(res)
    setLoading(false)
  }
  useEffect(() => { load() }, [eventId])
  useEffect(() => { api.get<{ name: string }[]>('/units').then((u) => setUnits(u.map((x) => x.name))).catch(() => {}) }, [])

  async function createList(e: FormEvent) {
    e.preventDefault()
    const name = newName.trim()
    if (!name) return
    setNewName('')
    await api.post(`/events/${eventId}/adhoc-lists`, { name })
    await load()
  }
  async function deleteList(id: string) {
    if (!confirm(t('mobile.adhocDeleteList'))) return
    await api.del(`/adhoc-lists/${id}`)
    await load()
    onChanged()
  }

  if (loading) return <p className="text-center text-muted">…</p>
  return (
    <div className="space-y-4">
      <datalist id="adhoc-units">
        {units.map((u) => <option key={u} value={u} />)}
      </datalist>
      <p className="px-1 text-xs text-muted">{t('mobile.adhocIntro')}</p>
      <form onSubmit={createList} className="flex items-center gap-2">
        <input
          className="input h-10 flex-1"
          placeholder={t('mobile.adhocNewList')}
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
        />
        <button type="submit" disabled={!newName.trim()} className="btn-primary h-10 px-3 disabled:opacity-50">
          <Plus size={18} />
        </button>
      </form>
      {lists.length === 0 ? (
        <p className="text-center text-muted">{t('mobile.adhocEmpty')}</p>
      ) : (
        lists.map((l) => (
          <AdhocListCard key={l.id} list={l} onChanged={() => { load(); onChanged() }} onDelete={() => deleteList(l.id)} />
        ))
      )}
    </div>
  )
}

function AdhocListCard({ list, onChanged, onDelete }: { list: AdhocList; onChanged: () => void; onDelete: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [qty, setQty] = useState('')
  const [unit, setUnit] = useState('')
  const articles = list.articles ?? []

  async function addItem(e: FormEvent) {
    e.preventDefault()
    const n = name.trim()
    if (!n) return
    setName(''); setQty(''); setUnit('')
    await api.post(`/adhoc-lists/${list.id}/items`, { name: n, unit: unit.trim(), quantity: parseFloat(qty) || 0 })
    onChanged()
  }
  async function removeItem(id: string) {
    await api.del(`/adhoc-items/${id}`)
    onChanged()
  }

  return (
    <section className="card p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <h3 className="min-w-0 truncate font-semibold">{list.name}</h3>
        <button onClick={onDelete} className="shrink-0 rounded-lg p-1 text-muted hover:text-danger"><Trash2 size={16} /></button>
      </div>
      {articles.length > 0 && (
        <ul className="mb-2 space-y-1">
          {articles.map((a) => (
            <AdhocItemRow key={a.id} article={a} onChanged={onChanged} onRemove={() => removeItem(a.id)} />
          ))}
        </ul>
      )}
      <form onSubmit={addItem} className="flex items-center gap-1.5">
        <input className="input h-9 flex-1 text-sm" placeholder={t('mobile.adhocItemName')} value={name} onChange={(e) => setName(e.target.value)} />
        <input className="input h-9 w-14 text-sm" placeholder={t('mobile.adhocQty')} inputMode="decimal" value={qty} onChange={(e) => setQty(e.target.value)} />
        <input list="adhoc-units" className="input h-9 w-14 text-sm" placeholder={t('mobile.adhocUnit')} value={unit} onChange={(e) => setUnit(e.target.value)} />
        <button type="submit" disabled={!name.trim()} className="btn-primary h-9 px-2 disabled:opacity-50"><Plus size={16} /></button>
      </form>
    </section>
  )
}

// AdhocItemRow shows one top-up item with inline-editable name, quantity and
// unit. Edits save on blur (PATCH) and refresh the shopping list.
function AdhocItemRow({ article, onChanged, onRemove }: { article: TabArticle; onChanged: () => void; onRemove: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState(article.name)
  const [qty, setQty] = useState(article.quantity ? String(article.quantity) : '')
  const [unit, setUnit] = useState(article.unit)

  useEffect(() => {
    setName(article.name)
    setQty(article.quantity ? String(article.quantity) : '')
    setUnit(article.unit)
  }, [article.id, article.name, article.quantity, article.unit])

  async function save() {
    const n = name.trim()
    if (!n) { setName(article.name); return } // name is required; revert empties
    if (n === article.name && (parseFloat(qty) || 0) === article.quantity && unit.trim() === article.unit) return
    await api.patch(`/adhoc-items/${article.id}`, { name: n, unit: unit.trim(), quantity: parseFloat(qty) || 0 })
    onChanged()
  }

  return (
    <li className="flex items-center gap-1.5 rounded-lg bg-surface px-2 py-1.5 text-sm">
      <input
        className="input h-8 min-w-0 flex-1 border-transparent bg-transparent px-1 text-sm focus:border-border focus:bg-card"
        value={name} onChange={(e) => setName(e.target.value)} onBlur={save}
      />
      <input
        className="input h-8 w-12 border-transparent bg-transparent px-1 text-center text-sm focus:border-border focus:bg-card"
        inputMode="decimal" placeholder={t('mobile.adhocQty')} value={qty} onChange={(e) => setQty(e.target.value)} onBlur={save}
      />
      <input
        list="adhoc-units"
        className="input h-8 w-12 border-transparent bg-transparent px-1 text-center text-sm focus:border-border focus:bg-card"
        placeholder={t('mobile.adhocUnit')} value={unit} onChange={(e) => setUnit(e.target.value)} onBlur={save}
      />
      <button onClick={onRemove} className="shrink-0 text-muted hover:text-danger"><Trash2 size={14} /></button>
    </li>
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

// MobileRecipes is a read-only recipe browser for cooking: search a list, tap to
// read the full recipe (ingredients + steps) in a large, legible layout.
function MobileRecipes() {
  const { t } = useTranslation()
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [q, setQ] = useState('')
  const [tag, setTag] = useState('')
  const [selected, setSelected] = useState<Recipe | null>(null)

  useEffect(() => { api.get<Recipe[]>('/recipes').then(setRecipes).catch(() => {}) }, [])

  if (selected) return <MobileRecipeView recipe={selected} onBack={() => setSelected(null)} />

  const tags = [...new Set(recipes.flatMap((r) => r.tags ?? []))].sort()
  const list = recipes
    .filter((r) => r.name.toLowerCase().includes(q.trim().toLowerCase()) && (tag === '' || (r.tags ?? []).includes(tag)))
    .sort((a, b) => a.name.localeCompare(b.name))

  return (
    <div>
      <div className="relative mb-2">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
        <input className="input pl-9" placeholder={t('mobile.searchRecipe')} value={q} onChange={(e) => setQ(e.target.value)} />
      </div>
      {tags.length > 0 && (
        <div className="-mx-3 mb-3 flex gap-1.5 overflow-x-auto px-3 pb-1">
          <button onClick={() => setTag('')}
            className={`shrink-0 rounded-full border px-3 py-1 text-xs font-medium ${tag === '' ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted'}`}>
            {t('recipes.allTags')}
          </button>
          {tags.map((tg) => (
            <button key={tg} onClick={() => setTag(tg)}
              className={`shrink-0 rounded-full border px-3 py-1 text-xs font-medium capitalize ${tag === tg ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted'}`}>
              {tg}
            </button>
          ))}
        </div>
      )}
      {list.length === 0 ? (
        <p className="text-center text-muted">{t('mobile.noRecipes')}</p>
      ) : (
        <ul className="space-y-2">
          {list.map((r) => (
            <li key={r.id}>
              <button onClick={() => setSelected(r)} className="card flex w-full items-center gap-3 p-3 text-left transition active:scale-[.99]">
                {r.photoUrl
                  ? <img src={resolveAsset(r.photoUrl)} alt="" className="h-12 w-12 shrink-0 rounded-lg object-cover" />
                  : <span className="grid h-12 w-12 shrink-0 place-items-center rounded-lg bg-surface"><ChefHat size={20} className="text-muted" /></span>}
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">{r.name}</span>
                  <span className="text-xs text-muted">{r.basePersons} {t('menu.persons')} · {r.ingredients?.length ?? 0} ingr.</span>
                </span>
                <ChevronRight size={18} className="shrink-0 text-muted" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function MobileRecipeView({ recipe, onBack }: { recipe: Recipe; onBack: () => void }) {
  const { t } = useTranslation()
  const steps = (recipe.instructions ?? '').split('\n').map((s) => s.trim()).filter(Boolean)
  return (
    <div className="space-y-4">
      <button onClick={onBack} className="flex items-center gap-1 text-sm text-brand"><ArrowLeft size={16} /> {t('mobile.recipeBack')}</button>
      {recipe.photoUrl && <img src={resolveAsset(recipe.photoUrl)} alt="" className="max-h-52 w-full rounded-lg object-cover" />}
      <div>
        <h2 className="text-xl font-bold">{recipe.name}</h2>
        <p className="text-sm text-muted">{recipe.basePersons} {t('menu.persons')}</p>
      </div>
      <div>
        <h3 className="mb-2 font-semibold">{t('recipes.ingredients')}</h3>
        <ul className="divide-y divide-border">
          {recipe.ingredients?.map((ri) => (
            <li key={ri.id} className="flex justify-between gap-3 py-2">
              <span>{ri.ingredient?.canonicalName}</span>
              <span className="shrink-0 text-muted tabular-nums">{ri.quantity} {ri.unit}</span>
            </li>
          ))}
        </ul>
      </div>
      {steps.length > 0 && (
        <div>
          <h3 className="mb-2 font-semibold">{t('recipes.instructions')}</h3>
          <ol className="list-decimal space-y-3 pl-5 leading-relaxed marker:font-semibold marker:text-brand">
            {steps.map((s, i) => <li key={i}>{s}</li>)}
          </ol>
        </div>
      )}
    </div>
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
        <input type="checkbox" className="h-6 w-6 shrink-0 accent-brand" checked={line.bought} onChange={(e) => onUpdate({ boughtQuantity: e.target.checked ? line.quantity : 0 })} />
        <button className="min-w-0 flex-1 text-left" onClick={() => setOpen((v) => !v)}>
          <span className={`block font-medium ${line.bought ? 'line-through' : ''}`}>{line.name}</span>
          <span className="text-xs text-muted">{line.quantity} {line.unit}{supplyLabel ? ` · ${supplyLabel}` : ''}{line.observation ? ` · ${line.observation}` : ''}</span>
          {line.boughtQuantity > 0 && line.boughtQuantity < line.quantity && (
            <span className="block text-xs text-accent">{t('shopping.remaining', { n: Math.round((line.quantity - line.boughtQuantity) * 100) / 100, unit: line.unit })}</span>
          )}
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
