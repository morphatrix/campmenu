import { Fragment, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  DndContext, DragEndEvent, DragOverlay, DragStartEvent, PointerSensor, useDraggable, useDroppable, useSensor, useSensors,
} from '@dnd-kit/core'
import { Trash2, GripVertical } from 'lucide-react'
import { api } from '../../lib/api'
import { useLive } from '../../context/LiveContext'
import { isCocktail } from '../../lib/types'
import type { Event, Meal, MealType, Recipe } from '../../lib/types'

const MEAL_TYPES: MealType[] = ['BREAKFAST', 'LUNCH', 'DINNER', 'APERITIF', 'DESSERT']

function dayCount(start: string, end: string): number {
  return Math.max(1, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 86400000) + 1)
}

function RecipeChip({ recipe }: { recipe: Recipe }) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id: `recipe:${recipe.id}` })
  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={`flex cursor-grab touch-none items-center gap-1 rounded-lg border border-border bg-card px-2 py-1 text-xs ${isDragging ? 'opacity-30' : ''}`}
    >
      <GripVertical size={12} className="shrink-0 text-muted" />
      <span className="min-w-0 truncate">{recipe.name}</span>
    </div>
  )
}

function Cell({
  meal, effective, onChange,
}: {
  meal: Meal | undefined; effective: number; onChange: () => void
}) {
  const { setNodeRef, isOver } = useDroppable({ id: meal ? `meal:${meal.id}` : 'none', disabled: !meal })
  const { t } = useTranslation()
  if (!meal) return <div ref={setNodeRef} className="min-h-20 min-w-0 rounded-lg border border-dashed border-border" />

  async function removeRecipe(id: string) {
    await api.del(`/meal-recipes/${id}`)
    onChange()
  }
  async function setWeight(id: string, v: number) {
    await api.patch(`/meal-recipes/${id}`, { participantCount: v })
    onChange()
  }

  return (
    <div
      ref={setNodeRef}
      className={`min-h-20 min-w-0 space-y-1 rounded-lg border p-1.5 transition ${isOver ? 'border-brand bg-brand/10 ring-2 ring-brand/30' : 'border-border bg-card'}`}
    >
      {meal.recipes?.length ? (
        meal.recipes.map((mr) => (
          <div key={mr.id} className="flex items-center justify-between gap-1 rounded bg-surface px-1.5 py-1 text-xs">
            <span className="min-w-0 flex-1 truncate">{mr.recipe?.name}</span>
            <span className="flex shrink-0 items-center gap-1">
              <input
                className="w-10 rounded border border-border bg-card px-1 text-right"
                type="number"
                min={0}
                value={mr.participantCount || ''}
                placeholder={String(effective)}
                onChange={(e) => setWeight(mr.id, +e.target.value)}
                title={t('menu.persons')}
              />
              <button onClick={() => removeRecipe(mr.id)} className="text-danger"><Trash2 size={12} /></button>
            </span>
          </div>
        ))
      ) : (
        <p className="px-1 py-2 text-center text-[11px] text-muted">{t('menu.dropHere')}</p>
      )}
    </div>
  )
}

export default function MenuGrid({ event, effectiveParticipants }: { event: Event; effectiveParticipants: number }) {
  const { t, i18n } = useTranslation()
  const [meals, setMeals] = useState<Meal[]>([])
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [search, setSearch] = useState('')
  const [tagFilter, setTagFilter] = useState('')
  const [activeRecipe, setActiveRecipe] = useState<Recipe | null>(null)
  const days = dayCount(event.startDate, event.endDate)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))

  async function loadMeals() {
    setMeals(await api.get<Meal[]>(`/events/${event.id}/meals`))
  }
  useEffect(() => {
    loadMeals()
    // Cocktails are managed in the Apéro tab, not dropped on the menu grid.
    api.get<Recipe[]>('/recipes').then((r) => setRecipes(r.filter((x) => x.approved && !isCocktail(x))))
  }, [event.id])

  const tags = useMemo(() => [...new Set(recipes.flatMap((r) => r.tags ?? []))].sort(), [recipes])

  // Live refresh of the planned meals (other admins dropping recipes).
  useLive(loadMeals)

  const mealAt = useMemo(() => {
    const map = new Map<string, Meal>()
    meals.forEach((m) => map.set(`${m.dayIndex}:${m.type}`, m))
    return map
  }, [meals])

  // "Lundi 9 févr." computed from the start date + day offset, localized.
  function dayLabel(dayIndex: number): { weekday: string; date: string } {
    const d = new Date(event.startDate)
    d.setDate(d.getDate() + dayIndex)
    const weekday = d.toLocaleDateString(i18n.language, { weekday: 'long' })
    return {
      weekday: weekday.charAt(0).toUpperCase() + weekday.slice(1),
      date: d.toLocaleDateString(i18n.language, { day: 'numeric', month: 'short' }),
    }
  }

  function onDragStart(e: DragStartEvent) {
    const id = String(e.active.id).replace('recipe:', '')
    setActiveRecipe(recipes.find((r) => r.id === id) ?? null)
  }

  async function onDragEnd(e: DragEndEvent) {
    setActiveRecipe(null)
    const recipeId = String(e.active.id).replace('recipe:', '')
    const overId = e.over ? String(e.over.id) : ''
    if (!overId.startsWith('meal:')) return
    const mealId = overId.replace('meal:', '')
    try {
      await api.post(`/meals/${mealId}/recipes`, { recipeId, participantCount: 0 })
      loadMeals()
    } catch {
      /* max 3 reached */
    }
  }

  const filtered = recipes.filter((r) =>
    r.name.toLowerCase().includes(search.toLowerCase()) &&
    (tagFilter === '' || (r.tags ?? []).includes(tagFilter)),
  )

  return (
    <DndContext sensors={sensors} onDragStart={onDragStart} onDragEnd={onDragEnd} onDragCancel={() => setActiveRecipe(null)}>
      <div className="grid gap-4 lg:grid-cols-[220px_1fr]">
        <aside className="card h-fit p-3">
          <input className="input mb-2" placeholder="🔍" value={search} onChange={(e) => setSearch(e.target.value)} />
          {tags.length > 0 && (
            <select className="input mb-2 capitalize" value={tagFilter} onChange={(e) => setTagFilter(e.target.value)}>
              <option value="">{t('recipes.allTags')}</option>
              {tags.map((tg) => <option key={tg} value={tg}>{tg}</option>)}
            </select>
          )}
          <div className="flex max-h-[60vh] flex-col gap-1.5 overflow-y-auto">
            {filtered.map((r) => <RecipeChip key={r.id} recipe={r} />)}
          </div>
        </aside>

        <div className="overflow-x-auto">
          {/* One single grid so every column shares the same width (aligned). */}
          <div className="grid min-w-[680px] grid-cols-[110px_repeat(5,minmax(0,1fr))] gap-1">
            <div />
            {MEAL_TYPES.map((mt) => <div key={mt} className="py-1 text-center text-xs font-semibold text-muted">{t(`meals.${mt}`)}</div>)}
            {Array.from({ length: days }).map((_, day) => {
              const lbl = dayLabel(day)
              return (
                <Fragment key={day}>
                  <div className="flex flex-col items-center justify-center rounded-lg bg-surface px-1 py-2 text-center">
                    <span className="text-sm font-semibold leading-tight">{lbl.weekday}</span>
                    <span className="text-[11px] text-muted">{lbl.date}</span>
                  </div>
                  {MEAL_TYPES.map((mt) => (
                    <Cell key={mt} meal={mealAt.get(`${day}:${mt}`)} effective={effectiveParticipants} onChange={loadMeals} />
                  ))}
                </Fragment>
              )
            })}
          </div>
        </div>
      </div>

      {/* Floating clone that tracks the cursor precisely. */}
      <DragOverlay dropAnimation={null}>
        {activeRecipe ? (
          <div className="flex items-center gap-1 rounded-lg border border-brand bg-card px-2 py-1 text-xs shadow-lg">
            <GripVertical size={12} className="text-muted" />
            {activeRecipe.name}
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
