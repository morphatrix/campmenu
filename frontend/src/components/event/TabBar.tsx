import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { DndContext, DragEndEvent, PointerSensor, useSensor, useSensors } from '@dnd-kit/core'
import { SortableContext, arrayMove, horizontalListSortingStrategy, useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Plus, X, GripVertical } from 'lucide-react'
import { api } from '../../lib/api'
import type { EventTab, ProductList } from '../../lib/types'

function SortableTab({
  tab, active, isAdmin, onSelect, onDelete,
}: {
  tab: EventTab; active: boolean; isAdmin: boolean; onSelect: () => void; onDelete: () => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: tab.id, disabled: !isAdmin })
  const style = { transform: CSS.Transform.toString(transform), transition }
  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-1 rounded-lg border px-3 py-1.5 text-sm font-medium ${
        active ? 'border-brand bg-brand text-brand-fg' : 'border-border bg-card text-muted'
      }`}
    >
      {isAdmin && (
        <button {...attributes} {...listeners} className="cursor-grab opacity-60" aria-label="reorder">
          <GripVertical size={14} />
        </button>
      )}
      <button onClick={onSelect}>{tab.name}</button>
      {isAdmin && tab.removable && (
        <button onClick={onDelete} className="opacity-70 hover:text-danger" aria-label="delete">
          <X size={14} />
        </button>
      )}
    </div>
  )
}

export default function TabBar({
  tabs, active, isAdmin, onSelect, eventId, onChange,
}: {
  tabs: EventTab[]; active: string; isAdmin: boolean; onSelect: (id: string) => void; eventId: string; onChange: () => void
}) {
  const { t } = useTranslation()
  const [adding, setAdding] = useState(false)
  const [lists, setLists] = useState<ProductList[]>([])
  const [choice, setChoice] = useState('')
  const [newListName, setNewListName] = useState('')
  const [newListVoted, setNewListVoted] = useState(true)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))

  useEffect(() => {
    if (adding) api.get<ProductList[]>('/product-lists').then(setLists)
  }, [adding])
  const sorted = [...tabs].sort((a, b) => a.position - b.position)

  async function onDragEnd(e: DragEndEvent) {
    const { active: a, over } = e
    if (!over || a.id === over.id) return
    const ids = sorted.map((x) => x.id)
    const next = arrayMove(ids, ids.indexOf(String(a.id)), ids.indexOf(String(over.id)))
    await api.put(`/events/${eventId}/tabs/order`, { order: next })
    onChange()
  }

  async function addTab() {
    if (choice.startsWith('kind:')) {
      await api.post(`/events/${eventId}/tabs`, { kind: choice.slice(5) })
    } else {
      let listId = choice
      if (choice === '__new__') {
        if (!newListName.trim()) return
        const created = await api.post<ProductList>('/product-lists', { name: newListName, voted: newListVoted })
        listId = created.id
      }
      if (!listId) return
      await api.post(`/events/${eventId}/tabs`, { listId, icon: 'list', withRecipes: false })
    }
    setChoice('')
    setNewListName('')
    setAdding(false)
    onChange()
  }

  async function deleteTab(tab: EventTab) {
    if (!confirm(t('common.confirmDelete', { name: tab.name }))) return
    await api.del(`/tabs/${tab.id}`)
    onChange()
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <DndContext sensors={sensors} onDragEnd={onDragEnd}>
        <SortableContext items={sorted.map((x) => x.id)} strategy={horizontalListSortingStrategy}>
          <div className="flex flex-wrap gap-2">
            {sorted.map((tab) => (
              <SortableTab
                key={tab.id}
                tab={tab}
                active={tab.id === active}
                isAdmin={isAdmin}
                onSelect={() => onSelect(tab.id)}
                onDelete={() => deleteTab(tab)}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {isAdmin &&
        (adding ? (
          <div className="flex items-center gap-1">
            <select className="input w-52" value={choice} onChange={(e) => setChoice(e.target.value)} autoFocus>
              <option value="">{t('lists.chooseList')}</option>
              {!tabs.some((x) => x.kind === 'MENUS') && <option value="kind:MENUS">{t('events.includeMenus')}</option>}
              {!tabs.some((x) => x.kind === 'LOCATIONS') && <option value="kind:LOCATIONS">{t('events.includeLocations')}</option>}
              {lists.map((l) => <option key={l.id} value={l.id}>{l.name}{l.voted ? '' : t('lists.notVotedSuffix')}</option>)}
              <option value="__new__">{t('lists.newList')}</option>
            </select>
            {choice === '__new__' && (
              <>
                <input className="input w-36" value={newListName} placeholder={t('lists.name')} onChange={(e) => setNewListName(e.target.value)} />
                <label className="flex items-center gap-1 text-xs whitespace-nowrap">
                  <input type="checkbox" checked={newListVoted} onChange={(e) => setNewListVoted(e.target.checked)} /> {t('lists.voted')}
                </label>
              </>
            )}
            <button className="btn-primary" onClick={addTab}>{t('common.add')}</button>
            <button className="btn-ghost" onClick={() => setAdding(false)}>{t('common.cancel')}</button>
          </div>
        ) : (
          <button className="btn-ghost" onClick={() => setAdding(true)}>
            <Plus size={15} />
          </button>
        ))}
    </div>
  )
}
