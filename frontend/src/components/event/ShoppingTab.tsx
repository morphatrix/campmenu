import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Store } from 'lucide-react'
import { api } from '../../lib/api'
import { useLive } from '../../context/LiveContext'
import { displayName } from '../../lib/types'
import type { Event, EventParticipant, ShoppingLine, SiteConfig } from '../../lib/types'

const STANDARD = ['Drive', 'Station']

type ShoppingPatch = Partial<ShoppingLine> & { clearBroughtBy?: boolean }
type SortMode = 'category' | 'day' | 'alpha' | 'list'
type GroupItem = { line: ShoppingLine; qty: number }

export default function ShoppingTab({ event }: { event: Event }) {
  const { t, i18n } = useTranslation()
  const [lines, setLines] = useState<ShoppingLine[]>([])
  const [byAisle, setByAisle] = useState(false)
  const [sortMode, setSortMode] = useState<SortMode>('category')
  const [search, setSearch] = useState('')
  const [aiEnabled, setAiEnabled] = useState(false)
  const participants = (event.participants ?? []).filter((p) => p.user)

  useEffect(() => { api.get<SiteConfig>('/config').then((c) => setAiEnabled(!!c.aiEnabled)).catch(() => {}) }, [])

  async function load() {
    const res = await api.get<ShoppingLine[]>(`/events/${event.id}/shopping`)
    res.sort((a, b) => Number(a.bought) - Number(b.bought) || a.name.localeCompare(b.name))
    setLines(res)
  }
  useEffect(() => { load() }, [event.id])
  useLive(load)

  async function update(line: ShoppingLine, patch: ShoppingPatch) {
    setLines((ls) => ls.map((l) => {
      if (l !== line) return l
      const m = { ...l, ...patch }
      if (patch.boughtQuantity !== undefined) m.bought = l.quantity > 0 && patch.boughtQuantity >= l.quantity
      return m
    }))
    await api.patch(`/events/${event.id}/shopping`, {
      section: line.section, name: line.name, unit: line.unit, ingredientId: line.ingredientId ?? null, ...patch,
    })
  }

  function dayLabel(dayIndex: number): string {
    const d = new Date(event.startDate)
    d.setDate(d.getDate() + dayIndex)
    const weekday = d.toLocaleDateString(i18n.language, { weekday: 'long' })
    const date = d.toLocaleDateString(i18n.language, { day: 'numeric', month: 'short' })
    return `${weekday.charAt(0).toUpperCase()}${weekday.slice(1)} ${date}`
  }

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return q ? lines.filter((l) => l.name.toLowerCase().includes(q)) : lines
  }, [lines, search])

  // Grouping depends on the chosen sort mode: by section/aisle (default), by
  // event day, by source list, or a single flat alphabetical list. Lines that
  // span several days/lists appear in each relevant group (same object, so
  // edits stay in sync); lines with no day/list info fall into a catch-all.
  // In day mode, the displayed quantity is that day's slice (dayQuantities),
  // not the line's overall total — the checkbox/update still target the
  // full line since "bought" is a single decision for the whole ingredient.
  const groups = useMemo(() => {
    if (sortMode === 'alpha') {
      const sorted = [...filtered].sort((a, b) => a.name.localeCompare(b.name))
      return sorted.length > 0
        ? [['', sorted.map((l) => ({ line: l, qty: l.quantity }))] as [string, GroupItem[]]]
        : []
    }
    if (sortMode === 'day') {
      const map = new Map<string, GroupItem[]>()
      for (const l of filtered) {
        for (const key of l.days && l.days.length > 0 ? l.days.map(String) : ['other']) {
          if (!map.has(key)) map.set(key, [])
          map.get(key)!.push({ line: l, qty: key === 'other' ? l.quantity : (l.dayQuantities?.[key] ?? l.quantity) })
        }
      }
      return [...map.entries()]
        .sort((a, b) => (a[0] === 'other' ? 1 : b[0] === 'other' ? -1 : Number(a[0]) - Number(b[0])))
        .map(([key, items]) => [key === 'other' ? t('shopping.otherDay') : dayLabel(Number(key)), items] as [string, GroupItem[]])
    }
    if (sortMode === 'list') {
      const map = new Map<string, GroupItem[]>()
      for (const l of filtered) {
        for (const name of l.lists && l.lists.length > 0 ? l.lists : [t('shopping.general')]) {
          if (!map.has(name)) map.set(name, [])
          map.get(name)!.push({ line: l, qty: l.quantity })
        }
      }
      return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]))
    }
    const map = new Map<string, GroupItem[]>()
    for (const l of filtered) {
      const key = byAisle ? (l.aisle || t('shopping.otherAisle')) : (l.section || '')
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push({ line: l, qty: l.quantity })
    }
    return [...map.entries()].sort((a, b) => (a[0] === '' ? -1 : b[0] === '' ? 1 : a[0].localeCompare(b[0])))
  }, [filtered, sortMode, byAisle, t, i18n.language, event.startDate])

  if (lines.length === 0) return <p className="text-muted">{t('shopping.empty')}</p>

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="relative w-full max-w-xs">
          <Search size={15} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
          <input className="input h-9 pl-8" placeholder={t('shopping.search')} value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted">{t('shopping.sortBy')}</label>
          <select className="input h-9 py-1 text-sm" value={sortMode} onChange={(e) => setSortMode(e.target.value as SortMode)}>
            <option value="category">{t('shopping.sortCategory')}</option>
            <option value="day">{t('shopping.sortDay')}</option>
            <option value="alpha">{t('shopping.sortAlpha')}</option>
            <option value="list">{t('shopping.sortList')}</option>
          </select>
          {aiEnabled && sortMode === 'category' && (
            <button
              onClick={() => setByAisle((v) => !v)}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium ${byAisle ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-surface text-muted'}`}
            >
              <Store size={13} /> {t('shopping.byAisle')}
            </button>
          )}
        </div>
      </div>
      {groups.length === 0 && <p className="text-muted">{t('shopping.noResults')}</p>}
      {groups.map(([label, items]) => (
        <section key={label || '__general__'}>
          {sortMode !== 'alpha' && <h3 className="mb-2 font-semibold">{label || t('shopping.general')}</h3>}
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] border-collapse text-sm">
              <thead>
                <tr className="text-left text-muted">
                  <th className="p-2"></th>
                  <th className="p-2">{t('shopping.ingredient')}</th>
                  <th className="p-2 text-right">{t('shopping.quantity')}</th>
                  <th className="p-2" />
                  <th className="p-2">{t('shopping.supply')}</th>
                  <th className="p-2">{t('shopping.observation')}</th>
                </tr>
              </thead>
              <tbody>
                {items.map(({ line, qty }, i) => (
                  <tr key={`${label}|${line.name}|${line.unit}|${i}`} className={`border-t border-border ${line.bought ? 'opacity-50' : ''}`}>
                    <td className="p-2 text-center">
                      <input type="checkbox" checked={line.bought} onChange={(e) => update(line, { boughtQuantity: e.target.checked ? line.quantity : 0 })} title={t('shopping.bought')} />
                    </td>
                    <td className="p-2 font-medium">{line.name}</td>
                    <td className="p-2 text-right tabular-nums">{qty}</td>
                    <td className="p-2 pl-1 text-left text-muted">
                      {line.unit}
                      {sortMode !== 'day' && line.boughtQuantity > 0 && line.boughtQuantity < line.quantity && (
                        <span className="ml-1 text-xs text-accent">{t('shopping.remaining', { n: Math.round((line.quantity - line.boughtQuantity) * 100) / 100, unit: line.unit })}</span>
                      )}
                    </td>
                    <td className="p-2">
                      <SupplySelect line={line} participants={participants} onUpdate={(patch) => update(line, patch)} />
                    </td>
                    <td className="p-2">
                      <input className="input h-8 py-1" defaultValue={line.observation} onBlur={(e) => update(line, { observation: e.target.value })} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ))}
    </div>
  )
}

function SupplySelect({
  line, participants, onUpdate,
}: {
  line: ShoppingLine
  participants: EventParticipant[]
  onUpdate: (patch: ShoppingPatch) => void
}) {
  const { t } = useTranslation()

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

  return (
    <div className="flex items-center gap-1">
      <select className="input h-8 py-1" value={mode} onChange={(e) => onSelect(e.target.value)}>
        <option value="">—</option>
        {STANDARD.map((s) => <option key={s} value={s}>{s}</option>)}
        {participants.map((p) => (
          <option key={p.id} value={`user:${p.userId}`}>{t('shopping.broughtByName', { name: displayName(p.user) })}</option>
        ))}
        <option value="__other__">{t('shopping.other')}</option>
      </select>
      {mode === '__other__' && (
        <input className="input h-8 w-28 py-1" placeholder={t('shopping.otherPlaceholder')} defaultValue={otherText}
          onChange={(e) => setOtherText(e.target.value)} onBlur={(e) => onUpdate({ source: e.target.value, clearBroughtBy: true })} />
      )}
    </div>
  )
}
