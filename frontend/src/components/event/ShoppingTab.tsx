import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../../lib/api'
import { useLive } from '../../context/LiveContext'
import { displayName } from '../../lib/types'
import type { Event, EventParticipant, ShoppingLine } from '../../lib/types'

const STANDARD = ['Drive', 'Station']

type ShoppingPatch = Partial<ShoppingLine> & { clearBroughtBy?: boolean }

export default function ShoppingTab({ event }: { event: Event }) {
  const { t } = useTranslation()
  const [lines, setLines] = useState<ShoppingLine[]>([])
  const participants = (event.participants ?? []).filter((p) => p.user)

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

  // Group lines by section; the empty section ("Général") comes first.
  const groups = useMemo(() => {
    const map = new Map<string, ShoppingLine[]>()
    for (const l of lines) {
      const key = l.section || ''
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(l)
    }
    return [...map.entries()].sort((a, b) => (a[0] === '' ? -1 : b[0] === '' ? 1 : a[0].localeCompare(b[0])))
  }, [lines])

  if (lines.length === 0) return <p className="text-muted">{t('shopping.empty')}</p>

  return (
    <div className="space-y-6">
      {groups.map(([section, items]) => (
        <section key={section || '__general__'}>
          <h3 className="mb-2 font-semibold">{section || t('shopping.general')}</h3>
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
                {items.map((line, i) => (
                  <tr key={`${section}|${line.name}|${line.unit}|${i}`} className={`border-t border-border ${line.bought ? 'opacity-50' : ''}`}>
                    <td className="p-2 text-center">
                      <input type="checkbox" checked={line.bought} onChange={(e) => update(line, { boughtQuantity: e.target.checked ? line.quantity : 0 })} title={t('shopping.bought')} />
                    </td>
                    <td className="p-2 font-medium">{line.name}</td>
                    <td className="p-2 text-right tabular-nums">{line.quantity}</td>
                    <td className="p-2 pl-1 text-left text-muted">
                      {line.unit}
                      {line.boughtQuantity > 0 && line.boughtQuantity < line.quantity && (
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
