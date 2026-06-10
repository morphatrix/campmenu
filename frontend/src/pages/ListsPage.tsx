import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ListPlus, Pencil, Plus, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useLive } from '../context/LiveContext'
import type { ProductList } from '../lib/types'

export default function ListsPage() {
  const { t } = useTranslation()
  const [lists, setLists] = useState<ProductList[]>([])
  const [selectedId, setSelectedId] = useState<string>('')
  const [newName, setNewName] = useState('')

  async function load(preferred?: string) {
    const data = await api.get<ProductList[]>('/product-lists')
    setLists(data)
    setSelectedId((cur) => {
      const ids = new Set(data.map((l) => l.id))
      if (preferred && ids.has(preferred)) return preferred
      if (cur && ids.has(cur)) return cur
      return data[0]?.id ?? ''
    })
  }
  useEffect(() => { load() }, [])
  useLive(load)

  const selected = lists.find((l) => l.id === selectedId)

  async function createList() {
    if (!newName.trim()) return
    const list = await api.post<ProductList>('/product-lists', { name: newName })
    setNewName('')
    await load(list.id)
  }
  async function renameList(l: ProductList) {
    const name = prompt(t('lists.rename'), l.name)
    if (!name) return
    await api.patch(`/product-lists/${l.id}`, { name })
    load(l.id)
  }
  async function deleteList(l: ProductList) {
    if (!confirm(`${t('common.delete')} « ${l.name} » ?`)) return
    await api.del(`/product-lists/${l.id}`)
    load()
  }

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">{t('lists.title')}</h1>
      <div className="grid gap-4 md:grid-cols-[260px_1fr]">
        {/* Sub-lists */}
        <aside className="card h-fit p-4">
          <div className="mb-3 flex gap-2">
            <input className="input" placeholder={t('lists.name')} value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && createList()} />
            <button className="btn-primary shrink-0" onClick={createList}><Plus size={16} /></button>
          </div>
          {lists.length === 0 ? (
            <p className="text-sm text-muted">{t('lists.empty')}</p>
          ) : (
            <ul className="space-y-1">
              {lists.map((l) => (
                <li key={l.id}>
                  <div className={`flex items-center justify-between rounded-lg px-2 py-1.5 ${l.id === selectedId ? 'bg-brand text-brand-fg' : 'hover:bg-surface'}`}>
                    <button className="flex-1 text-left text-sm font-medium" onClick={() => setSelectedId(l.id)}>
                      {l.name} <span className="opacity-70">({l.items?.length ?? 0})</span>
                    </button>
                    <button className="opacity-80 hover:opacity-100" onClick={() => renameList(l)}><Pencil size={13} /></button>
                    <button className="ml-1 opacity-80 hover:text-danger" onClick={() => deleteList(l)}><Trash2 size={13} /></button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </aside>

        {/* Items of selected list */}
        <section className="card p-5">
          {!selected ? (
            <p className="text-muted">{t('lists.selectPrompt')}</p>
          ) : (
            <ListItemsEditor list={selected} onChange={() => load(selected.id)} />
          )}
        </section>
      </div>
    </div>
  )
}

function ListItemsEditor({ list, onChange }: { list: ProductList; onChange: () => void }) {
  const { t } = useTranslation()
  const voted = list.voted
  const sections = list.sections ?? []
  const [name, setName] = useState('')
  const [unit, setUnit] = useState('pièce')
  const [section, setSection] = useState('')
  const [qty, setQty] = useState(1)
  const [q1, setQ1] = useState(1)
  const [q2, setQ2] = useState(2)
  const [q3, setQ3] = useState(3)
  const [newSection, setNewSection] = useState('')

  async function add() {
    if (!name.trim()) return
    const body = voted
      ? { name, unit, section, qtyPerLevel: { '1': q1, '2': q2, '3': q3 } }
      : { name, unit, section, quantity: qty }
    await api.post(`/product-lists/${list.id}/items`, body)
    setName('')
    onChange()
  }
  async function remove(itemId: string) { await api.del(`/product-list-items/${itemId}`); onChange() }
  async function toggleVoted() { await api.patch(`/product-lists/${list.id}`, { voted: !voted }); onChange() }
  async function addSection() {
    if (!newSection.trim()) return
    await api.patch(`/product-lists/${list.id}`, { sections: [...sections, newSection.trim()] })
    setNewSection('')
    onChange()
  }
  async function removeSection(s: string) {
    await api.patch(`/product-lists/${list.id}`, { sections: sections.filter((x) => x !== s) })
    onChange()
  }

  const items = list.items ?? []
  const groups = [...sections, ...new Set(items.map((i) => i.section).filter((s) => s && !sections.includes(s))), '']

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <ListPlus size={18} className="text-brand" />
        <h2 className="text-lg font-semibold">{list.name}</h2>
        <span className="chip">{items.length} {t('lists.products').toLowerCase()}</span>
        <label className="ml-auto flex items-center gap-1 text-sm">
          <input type="checkbox" checked={voted} onChange={toggleVoted} /> {t('lists.voted')}
        </label>
      </div>

      {/* Sections management */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        {sections.map((s) => (
          <span key={s} className="chip">{s}<button className="ml-1 hover:text-danger" onClick={() => removeSection(s)}>×</button></span>
        ))}
        <input className="input h-8 w-40 py-1" placeholder={t('lists.addSection')} value={newSection}
          onChange={(e) => setNewSection(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addSection()} />
        <button className="btn-ghost" onClick={addSection}><Plus size={14} /></button>
      </div>

      {/* Items grouped by section */}
      {items.length === 0 ? (
        <p className="mb-5 text-sm text-muted">{t('lists.noProducts')}</p>
      ) : (
        <div className="mb-5 space-y-4">
          {groups.map((sec) => {
            const its = items.filter((i) => (i.section || '') === sec)
            if (its.length === 0) return null
            return (
              <div key={sec || '__none__'}>
                {sec && <p className="mb-1 text-xs font-semibold uppercase text-muted">{sec}</p>}
                <ul className="divide-y divide-border">
                  {its.map((it) => (
                    <li key={it.id} className="flex items-center justify-between py-1.5 text-sm">
                      <span className="font-medium">{it.name} <span className="text-muted">{it.unit}</span></span>
                      <span className="flex items-center gap-3">
                        <span className="text-xs text-muted">
                          {voted ? `niv. ${Object.entries(it.qtyPerLevel ?? {}).map(([k, v]) => `${k}:${v}`).join(' ')}` : `${it.quantity} ${it.unit}`}
                        </span>
                        <button className="text-danger" onClick={() => remove(it.id)}><Trash2 size={14} /></button>
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            )
          })}
        </div>
      )}

      <div className="flex flex-wrap items-end gap-2 border-t border-border pt-4">
        <div>
          <label className="label">{t('lists.product')}</label>
          <input className="input w-40" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && add()} />
        </div>
        <div>
          <label className="label">unité</label>
          <input className="input w-20" value={unit} onChange={(e) => setUnit(e.target.value)} />
        </div>
        {sections.length > 0 && (
          <div>
            <label className="label">section</label>
            <select className="input w-32" value={section} onChange={(e) => setSection(e.target.value)}>
              <option value="">—</option>
              {sections.map((s) => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
        )}
        {voted ? (
          <div>
            <label className="label">qté niv.1/2/3</label>
            <div className="flex gap-1">
              <input className="input w-14" type="number" step="0.1" value={q1} onChange={(e) => setQ1(+e.target.value)} />
              <input className="input w-14" type="number" step="0.1" value={q2} onChange={(e) => setQ2(+e.target.value)} />
              <input className="input w-14" type="number" step="0.1" value={q3} onChange={(e) => setQ3(+e.target.value)} />
            </div>
          </div>
        ) : (
          <div>
            <label className="label">{t('lists.total')}</label>
            <input className="input w-20" type="number" step="0.1" value={qty} onChange={(e) => setQty(+e.target.value)} />
          </div>
        )}
        <button className="btn-primary" onClick={add}><Plus size={15} /> {t('lists.addProduct')}</button>
      </div>
    </div>
  )
}
