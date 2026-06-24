import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, X, Wine, Save } from 'lucide-react'
import { api } from '../../lib/api'
import { useLive } from '../../context/LiveContext'
import { useAuth } from '../../context/AuthContext'
import { displayName, isCocktail } from '../../lib/types'
import type { Event, EventTab, ProductList, Recipe, TabArticle, TabConsumption } from '../../lib/types'

function dayCount(start: string, end: string): number {
  return Math.max(1, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 86400000) + 1)
}

interface Props {
  tab: EventTab
  event: Event
  isAdmin: boolean
  effectiveParticipants: number
  onChange: () => void
}

export default function MatrixTab(props: Props) {
  return (
    <div className="space-y-3">
      <SaveListToCatalog tab={props.tab} event={props.event} isAdmin={props.isAdmin} onChange={props.onChange} />
      {props.tab.voted ? <VotedMatrix {...props} /> : <NonVotedMatrix {...props} />}
    </div>
  )
}

// SaveListToCatalog lets an organizer promote this tab's event-private source
// list into the shared catalog. It renders nothing for global lists or non-admins.
function SaveListToCatalog({
  tab, event, isAdmin, onChange,
}: {
  tab: EventTab; event: Event; isAdmin: boolean; onChange: () => void
}) {
  const { t } = useTranslation()
  const [list, setList] = useState<ProductList | null>(null)
  useEffect(() => {
    if (!tab.listId) { setList(null); return }
    api.get<ProductList[]>(`/product-lists?eventId=${event.id}`)
      .then((ls) => setList(ls.find((l) => l.id === tab.listId) ?? null))
  }, [tab.listId, event.id])

  if (!isAdmin || !list || !list.eventId) return null

  async function save() {
    await api.post(`/product-lists/${list!.id}/save`, {})
    onChange()
  }
  return (
    <div className="flex items-center justify-end gap-2">
      <span className="chip bg-surface text-xs text-muted">{t('lists.eventOnlyBadge')}</span>
      <button className="btn-ghost text-xs" onClick={save} title={t('lists.saveToCatalogHint')}>
        <Save size={14} /> {t('lists.saveToCatalog')}
      </button>
    </div>
  )
}

// ─── Voted: participants pick a per-day quantity, total = Σ × days ──────────

function levelLabel(art: TabArticle, level: string): string {
  if (level === '0') return '0'
  const qty = art.qtyPerLevel?.[level]
  if (qty == null) return level
  return `${qty} ${art.unit}/j`.trim()
}

function VotedMatrix({ tab, event, isAdmin, onChange }: Props) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [cons, setCons] = useState<TabConsumption[]>([])
  const articles = tab.articles ?? []
  const days = dayCount(event.startDate, event.endDate)
  const participants = (event.participants ?? []).filter((p) => p.user)
  const levels = Object.keys(tab.consumptionLabels ?? { '0': '', '1': '', '2': '', '3': '' }).sort()

  async function loadCons() {
    setCons(await api.get<TabConsumption[]>(`/tabs/${tab.id}/consumption`))
  }
  useEffect(() => { loadCons() }, [tab.id])
  useLive(loadCons)

  const levelOf = useMemo(() => {
    const map = new Map<string, number>()
    cons.forEach((c) => map.set(`${c.articleId}:${c.userId}`, c.level))
    return map
  }, [cons])

  async function setLevel(articleId: string, level: number) {
    await api.put(`/tabs/${tab.id}/articles/${articleId}/consumption`, { level })
    loadCons()
  }

  function total(art: TabArticle): number {
    let sum = 0
    participants.forEach((p) => {
      const lvl = levelOf.get(`${art.id}:${p.userId}`) ?? 0
      if (lvl > 0) sum += art.qtyPerLevel?.[String(lvl)] ?? 0
    })
    return Math.round(sum * days * 100) / 100
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] border-collapse text-sm">
          <thead>
            <tr className="text-left text-muted">
              <th className="p-2">{t('matrix.article')}</th>
              {participants.map((p) => (
                <th key={p.id} className="p-2 text-center font-medium">{displayName(p.user)}</th>
              ))}
              <th className="p-2 text-right">{t('matrix.total')}</th>
              <th className="p-2" />
              {isAdmin && <th />}
            </tr>
          </thead>
          <tbody>
            {articles.map((art) => (
              <tr key={art.id} className="border-t border-border">
                <td className="p-2 font-medium">{art.name} <span className="text-xs text-muted">{art.unit}</span></td>
                {participants.map((p) => {
                  const mine = p.userId === user?.id
                  const lvl = levelOf.get(`${art.id}:${p.userId}`) ?? 0
                  return (
                    <td key={p.id} className="p-1 text-center">
                      <select
                        className={`rounded border border-border px-1 py-0.5 text-xs ${mine ? 'bg-card' : 'bg-surface text-muted'}`}
                        value={lvl} disabled={!mine} onChange={(e) => setLevel(art.id, +e.target.value)}
                      >
                        {levels.map((l) => <option key={l} value={l}>{levelLabel(art, l)}</option>)}
                      </select>
                    </td>
                  )
                })}
                <td className="p-2 text-right font-semibold tabular-nums">{total(art)}</td>
                <td className="p-2 pl-1 text-left text-xs text-muted">{art.unit}</td>
                {isAdmin && (
                  <td className="p-1">
                    <button className="text-danger" onClick={async () => { await api.del(`/articles/${art.id}`); onChange() }}><Trash2 size={14} /></button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-muted">
        Quantité choisie <strong>par personne et par jour</strong> · Total = somme des choix × {days} jour{days > 1 ? 's' : ''}
      </p>
      {isAdmin && <AddVotedArticle tab={tab} event={event} existing={articles} onAdded={onChange} />}
    </div>
  )
}

function AddVotedArticle({ tab, event, existing, onAdded }: { tab: EventTab; event: Event; existing: TabArticle[]; onAdded: () => void }) {
  const { t } = useTranslation()
  const [list, setList] = useState<ProductList | null>(null)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [name, setName] = useState('')
  const [unit, setUnit] = useState('pièce')
  const [q1, setQ1] = useState(1)
  const [q2, setQ2] = useState(2)
  const [q3, setQ3] = useState(3)

  async function loadList() {
    if (!tab.listId) return
    const lists = await api.get<ProductList[]>(`/product-lists?eventId=${event.id}`)
    setList(lists.find((l) => l.id === tab.listId) ?? null)
  }
  useEffect(() => { loadList() }, [tab.listId, event.id])

  const existingNames = new Set(existing.map((a) => a.name.toLowerCase()))
  const available = (list?.items ?? []).filter((it) => !existingNames.has(it.name.toLowerCase()))

  async function addSelection() {
    for (const it of available.filter((i) => selected.has(i.id))) {
      await api.post(`/tabs/${tab.id}/articles`, { name: it.name, unit: it.unit, qtyPerLevel: it.qtyPerLevel })
    }
    setSelected(new Set())
    onAdded()
  }
  async function addManual() {
    if (!name.trim()) return
    const qtyPerLevel = { '1': q1, '2': q2, '3': q3 }
    await api.post(`/tabs/${tab.id}/articles`, { name, unit, qtyPerLevel })
    if (tab.listId) await api.post(`/product-lists/${tab.listId}/items`, { name, unit, qtyPerLevel })
    setName('')
    onAdded()
  }

  return (
    <div className="space-y-4 border-t border-border pt-4">
      {tab.listId && available.length > 0 && (
        <div>
          <p className="mb-2 text-sm font-medium">{t('matrix.fromList')} · {list?.name}</p>
          <div className="flex flex-wrap gap-2">
            {available.map((it) => (
              <label key={it.id} className={`chip cursor-pointer ${selected.has(it.id) ? 'bg-brand text-brand-fg' : ''}`}>
                <input type="checkbox" className="hidden" checked={selected.has(it.id)}
                  onChange={() => setSelected((s) => { const n = new Set(s); n.has(it.id) ? n.delete(it.id) : n.add(it.id); return n })} />
                {it.name}
              </label>
            ))}
          </div>
          <button className="btn-primary mt-3" disabled={selected.size === 0} onClick={addSelection}>
            <Plus size={15} /> {t('matrix.addSelection')} ({selected.size})
          </button>
        </div>
      )}
      <div className="flex flex-wrap items-end gap-2">
        <div><label className="label">{t('matrix.newArticle')}</label><input className="input w-44" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addManual()} /></div>
        <div><label className="label">unité</label><input className="input w-24" value={unit} onChange={(e) => setUnit(e.target.value)} /></div>
        <div>
          <label className="label">qté niv.1/2/3</label>
          <div className="flex gap-1">
            <input className="input w-16" type="number" step="0.1" value={q1} onChange={(e) => setQ1(+e.target.value)} />
            <input className="input w-16" type="number" step="0.1" value={q2} onChange={(e) => setQ2(+e.target.value)} />
            <input className="input w-16" type="number" step="0.1" value={q3} onChange={(e) => setQ3(+e.target.value)} />
          </div>
        </div>
        <button className="btn-primary" onClick={addManual}><Plus size={15} /> {t('matrix.addArticle')}</button>
      </div>
    </div>
  )
}

// ─── Non-voted: organizers set total quantities, grouped by section ─────────

function NonVotedMatrix({ tab, isAdmin, effectiveParticipants, onChange }: Props) {
  const { t } = useTranslation()
  const articles = tab.articles ?? []
  const recipes = tab.recipes ?? []
  const [cocktails, setCocktails] = useState<Recipe[]>([])
  const [units, setUnits] = useState<string[]>([])
  const [newSection, setNewSection] = useState('')

  useEffect(() => {
    api.get<Recipe[]>('/recipes').then((r) => setCocktails(r.filter((x) => isCocktail(x) && x.approved)))
    api.get<{ name: string }[]>('/units').then((u) => setUnits(u.map((x) => x.name))).catch(() => {})
  }, [])

  // Ordered groups: tab.sections first, then any leftover sections, then unsectioned.
  const groups = useMemo(() => {
    const ordered = [...(tab.sections ?? [])]
    const extra = new Set<string>()
    articles.forEach((a) => { if (a.section && !ordered.includes(a.section)) extra.add(a.section) })
    recipes.forEach((r) => { if (r.section && !ordered.includes(r.section)) extra.add(r.section) })
    return [...ordered, ...extra, ''] // '' = sans section
  }, [tab.sections, articles, recipes])

  async function setQuantity(art: TabArticle, quantity: number) {
    await api.patch(`/articles/${art.id}`, { name: art.name, unit: art.unit, section: art.section, quantity })
    onChange()
  }
  async function setArticleUnit(art: TabArticle, unit: string) {
    if (unit === art.unit) return
    await api.patch(`/articles/${art.id}`, { name: art.name, unit, section: art.section, quantity: art.quantity })
    onChange()
  }
  async function removeArticle(id: string) { await api.del(`/articles/${id}`); onChange() }
  async function removeRecipe(id: string) { await api.del(`/tab-recipes/${id}`); onChange() }
  async function setRecipeCount(id: string, count: number) { await api.patch(`/tab-recipes/${id}`, { participantCount: count }); onChange() }

  async function addSection() {
    if (!newSection.trim()) return
    await api.patch(`/tabs/${tab.id}`, { sections: [...(tab.sections ?? []), newSection.trim()] })
    setNewSection('')
    onChange()
  }
  async function removeSection(name: string) {
    await api.patch(`/tabs/${tab.id}`, { sections: (tab.sections ?? []).filter((s) => s !== name) })
    onChange()
  }

  return (
    <div className="space-y-6">
      <datalist id="matrix-units">
        {units.map((u) => <option key={u} value={u} />)}
      </datalist>
      {groups.map((section) => {
        const arts = articles.filter((a) => (a.section || '') === section)
        const recs = recipes.filter((r) => (r.section || '') === section)
        if (section === '' && arts.length === 0 && recs.length === 0) return null
        return (
          <section key={section || '__none__'} className="card p-4">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="font-semibold">{section || t('shopping.other')}</h3>
              {isAdmin && section && (tab.sections ?? []).includes(section) && (
                <button className="text-muted hover:text-danger" title={t('common.delete')} onClick={() => removeSection(section)}><X size={15} /></button>
              )}
            </div>

            <ul className="divide-y divide-border">
              {arts.map((art) => (
                <li key={art.id} className="flex items-center justify-between gap-2 py-1.5 text-sm">
                  <span className="font-medium">{art.name}</span>
                  <span className="flex items-center gap-2">
                    {isAdmin ? (
                      <input className="input h-8 w-20 py-1 text-right" type="number" step="0.1" defaultValue={art.quantity || ''}
                        onBlur={(e) => setQuantity(art, +e.target.value)} />
                    ) : (
                      <span className="inline-block w-20 text-right font-semibold tabular-nums">{art.quantity}</span>
                    )}
                    {isAdmin ? (
                      <input list="matrix-units" className="input h-8 w-20 py-1" defaultValue={art.unit}
                        onBlur={(e) => setArticleUnit(art, e.target.value.trim())} />
                    ) : (
                      <span className="text-muted">{art.unit}</span>
                    )}
                    {isAdmin && <button className="text-danger" onClick={() => removeArticle(art.id)}><Trash2 size={14} /></button>}
                  </span>
                </li>
              ))}
              {recs.map((tr) => (
                <li key={tr.id} className="flex items-center justify-between gap-2 py-1.5 text-sm">
                  <span className="flex items-center gap-1 font-medium"><Wine size={14} className="text-accent" /> {tr.recipe?.name}</span>
                  <span className="flex items-center gap-2">
                    {isAdmin ? (
                      <input className="input h-8 w-20 py-1 text-right" type="number" min={0} defaultValue={tr.participantCount || ''}
                        placeholder={String(effectiveParticipants)} onBlur={(e) => setRecipeCount(tr.id, +e.target.value)} />
                    ) : (
                      <span className="inline-block w-20 text-right font-semibold tabular-nums">{tr.participantCount || effectiveParticipants}</span>
                    )}
                    <span className="text-muted">{t('menu.persons')}</span>
                    {isAdmin && <button className="text-danger" onClick={() => removeRecipe(tr.id)}><Trash2 size={14} /></button>}
                  </span>
                </li>
              ))}
            </ul>

            {isAdmin && <AddToSection tab={tab} section={section} cocktails={cocktails} onAdded={onChange} />}
          </section>
        )
      })}

      {isAdmin && (
        <div className="flex items-center gap-2">
          <input className="input w-48" placeholder={t('lists.addSection')} value={newSection}
            onChange={(e) => setNewSection(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addSection()} />
          <button className="btn-ghost" onClick={addSection}><Plus size={15} /> {t('lists.addSection')}</button>
        </div>
      )}
    </div>
  )
}

function AddToSection({ tab, section, cocktails, onAdded }: { tab: EventTab; section: string; cocktails: Recipe[]; onAdded: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [unit, setUnit] = useState('pièce')
  const [qty, setQty] = useState(1)
  const [cocktailId, setCocktailId] = useState('')

  async function addArticle() {
    if (!name.trim()) return
    await api.post(`/tabs/${tab.id}/articles`, { name, unit, section, quantity: qty })
    setName('')
    onAdded()
  }
  async function addCocktail() {
    if (!cocktailId) return
    await api.post(`/tabs/${tab.id}/recipes`, { recipeId: cocktailId, section, participantCount: 0 })
    setCocktailId('')
    onAdded()
  }

  return (
    <div className="mt-3 flex flex-wrap items-end gap-2 border-t border-border pt-3">
      <input className="input w-40" placeholder={t('matrix.article')} value={name} onChange={(e) => setName(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addArticle()} />
      <input list="matrix-units" className="input w-20" placeholder="unité" value={unit} onChange={(e) => setUnit(e.target.value)} />
      <input className="input w-20" type="number" step="0.1" placeholder="qté" value={qty || ''} onChange={(e) => setQty(+e.target.value)} />
      <button className="btn-ghost" onClick={addArticle}><Plus size={15} /> {t('matrix.addArticle')}</button>
      {cocktails.length > 0 && section.toLowerCase().includes('cocktail') && (
        <span className="ml-auto flex items-center gap-1">
          <select className="input w-40" value={cocktailId} onChange={(e) => setCocktailId(e.target.value)}>
            <option value="">{t('lists.addCocktail')}…</option>
            {cocktails.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          <button className="btn-ghost" disabled={!cocktailId} onClick={addCocktail}><Wine size={15} /></button>
        </span>
      )}
    </div>
  )
}
