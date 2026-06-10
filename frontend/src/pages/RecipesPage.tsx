import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChefHat, Pencil, Plus, Trash2, Users, Wine, X } from 'lucide-react'
import { api, resolveAsset } from '../lib/api'
import { useLive } from '../context/LiveContext'
import { useAuth } from '../context/AuthContext'
import IngredientInput from '../components/IngredientInput'
import ImageUpload from '../components/ImageUpload'
import Modal from '../components/Modal'
import { isCocktail, isStaff } from '../lib/types'
import type { Recipe } from '../lib/types'

interface DraftIngredient { name: string; quantity: number; unit: string }

const PREDEFINED_TAGS = ['apéro', 'entrée', 'plat', 'accompagnement', 'dessert', 'petit-déjeuner', 'boisson']

export default function RecipesPage({ cocktails = false }: { cocktails?: boolean }) {
  const { t } = useTranslation()
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Recipe | null>(null)
  const [selected, setSelected] = useState<Recipe | null>(null)
  const [tagFilter, setTagFilter] = useState('')

  async function load() {
    setRecipes(await api.get<Recipe[]>('/recipes'))
  }
  useEffect(() => { load() }, [])
  useLive(load)

  const visible = recipes.filter((r) => (cocktails ? isCocktail(r) : !isCocktail(r)))
  const tags = useMemo(() => [...new Set(visible.flatMap((r) => r.tags ?? []))].filter((tg) => tg !== 'cocktail').sort(), [visible])
  const displayed = visible.filter((r) => tagFilter === '' || (r.tags ?? []).includes(tagFilter))
  const showAside = !cocktails && tags.length > 0

  return (
    <div className={`grid gap-4 ${showAside ? 'md:grid-cols-[200px_1fr]' : ''}`}>
      {/* Tag filter (recipes only) */}
      {showAside && (
        <aside className="card h-fit p-3">
          <p className="mb-2 text-xs font-semibold uppercase text-muted">{t('recipes.tags')}</p>
          <ul className="space-y-1 text-sm">
            <li><button className={`w-full rounded-lg px-2 py-1 text-left ${tagFilter === '' ? 'bg-brand text-brand-fg' : 'hover:bg-surface'}`} onClick={() => setTagFilter('')}>{t('recipes.allTags')}</button></li>
            {tags.map((tg) => (
              <li key={tg}><button className={`w-full rounded-lg px-2 py-1 text-left capitalize ${tagFilter === tg ? 'bg-brand text-brand-fg' : 'hover:bg-surface'}`} onClick={() => setTagFilter(tg)}>{tg}</button></li>
            ))}
          </ul>
        </aside>
      )}

      <div>
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-bold">{cocktails ? t('nav.cocktails') : t('recipes.title')}</h1>
          <button className="btn-primary" onClick={() => { setEditing(null); setShowForm(true) }}>
            <Plus size={16} /> {cocktails ? t('recipes.createCocktail') : t('recipes.create')}
          </button>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {displayed.map((r) => (
            <button key={r.id} onClick={() => setSelected(r)} className="card relative block h-44 overflow-hidden p-0 text-left transition hover:border-brand">
              {r.photoUrl ? (
                <img src={resolveAsset(r.photoUrl)} alt={r.name} className="absolute inset-0 h-full w-full object-cover" />
              ) : (
                <div className="grid h-full place-items-center bg-surface">{isCocktail(r) ? <Wine className="text-muted" size={32} /> : <ChefHat className="text-muted" size={32} />}</div>
              )}
              <span className={`chip absolute right-2 top-2 ${r.approved ? 'text-success' : 'text-accent'} ${r.photoUrl ? 'bg-card/60 backdrop-blur-sm' : ''}`}>
                {r.approved ? t('recipes.approved') : t('recipes.pending')}
              </span>
              <div className={`absolute inset-x-0 bottom-0 p-3 ${r.photoUrl ? 'bg-card/55 backdrop-blur-md' : ''}`}>
                <h2 className="font-semibold">{r.name}</h2>
                <p className="flex flex-wrap items-center gap-2 text-xs text-muted">
                  <span className="flex items-center gap-1"><Users size={12} /> {r.basePersons}</span>
                  {(r.tags ?? []).filter((tg) => tg !== 'cocktail').map((tg) => <span key={tg} className="capitalize">· {tg}</span>)}
                  <span>· {r.ingredients?.length ?? 0} ingr.</span>
                </p>
              </div>
            </button>
          ))}
        </div>

        {selected && (
          <RecipeDetail recipe={selected} onClose={() => setSelected(null)}
            onEdit={() => { setEditing(selected); setSelected(null); setShowForm(true) }}
            onChanged={() => { setSelected(null); load() }} />
        )}
        {showForm && (
          <RecipeFormModal initial={editing} forceCocktail={cocktails}
            onClose={() => setShowForm(false)} onSaved={() => { setShowForm(false); load() }} />
        )}
      </div>
    </div>
  )
}

function RecipeDetail({ recipe, onClose, onEdit, onChanged }: { recipe: Recipe; onClose: () => void; onEdit: () => void; onChanged: () => void }) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const canEdit = isStaff(user) || recipe.createdBy === user?.id
  const steps = (recipe.instructions ?? '').split('\n').map((l) => l.trim()).filter(Boolean)

  async function approve() { await api.post(`/recipes/${recipe.id}/approve`); onChanged() }
  async function remove() {
    if (!confirm(t('common.confirmDelete', { name: recipe.name }))) return
    await api.del(`/recipes/${recipe.id}`)
    onChanged()
  }

  return (
    <Modal title={recipe.name} onClose={onClose} wide>
      {recipe.photoUrl && <img src={resolveAsset(recipe.photoUrl)} alt={recipe.name} className="mb-4 max-h-60 w-full rounded-lg object-cover" />}
      <div className="mb-4 flex flex-wrap gap-2 text-xs">
        <span className="chip"><Users size={12} /> {recipe.basePersons} {t('menu.persons')}</span>
        {(recipe.tags ?? []).map((tg) => <span key={tg} className="chip capitalize">{tg}</span>)}
        <span className={`chip ${recipe.approved ? 'text-success' : 'text-accent'}`}>{recipe.approved ? t('recipes.approved') : t('recipes.pending')}</span>
      </div>

      <h3 className="mb-2 font-semibold">{t('recipes.ingredients')}</h3>
      <ul className="mb-5 divide-y divide-border">
        {recipe.ingredients?.map((ri) => (
          <li key={ri.id} className="flex justify-between py-1.5 text-sm">
            <span>{ri.ingredient?.canonicalName}</span>
            <span className="text-muted">{ri.quantity} {ri.unit}</span>
          </li>
        ))}
      </ul>

      {steps.length > 0 && (
        <>
          <h3 className="mb-2 font-semibold">{t('recipes.instructions')}</h3>
          <ol className="list-decimal space-y-2 pl-5 text-sm leading-relaxed marker:font-semibold marker:text-brand">
            {steps.map((step, i) => <li key={i}>{step}</li>)}
          </ol>
        </>
      )}

      {canEdit && (
        <div className="mt-6 flex flex-wrap justify-end gap-2 border-t border-border pt-4">
          {isStaff(user) && !recipe.approved && (
            <button className="btn-ghost text-success" onClick={approve}><Check size={15} /> {t('recipes.approve')}</button>
          )}
          <button className="btn-ghost" onClick={onEdit}><Pencil size={15} /> {t('common.edit')}</button>
          <button className="btn-ghost text-danger" onClick={remove}><Trash2 size={15} /> {t('common.delete')}</button>
        </div>
      )}
    </Modal>
  )
}

function TagSelector({ tags, onChange }: { tags: string[]; onChange: (t: string[]) => void }) {
  const { t } = useTranslation()
  const [custom, setCustom] = useState('')
  function toggle(tag: string) { onChange(tags.includes(tag) ? tags.filter((x) => x !== tag) : [...tags, tag]) }
  const all = [...PREDEFINED_TAGS, ...tags.filter((x) => x !== 'cocktail' && !PREDEFINED_TAGS.includes(x))]
  return (
    <div>
      <div className="flex flex-wrap gap-2">
        {all.map((tag) => (
          <label key={tag} className={`chip cursor-pointer capitalize ${tags.includes(tag) ? 'bg-brand text-brand-fg' : ''}`}>
            <input type="checkbox" className="hidden" checked={tags.includes(tag)} onChange={() => toggle(tag)} /> {tag}
          </label>
        ))}
      </div>
      <div className="mt-2 flex gap-2">
        <input className="input w-40" placeholder={t('recipes.addTag')} value={custom} onChange={(e) => setCustom(e.target.value)} />
        <button type="button" className="btn-ghost" onClick={() => { if (custom.trim()) { toggle(custom.trim().toLowerCase()); setCustom('') } }}><Plus size={15} /></button>
      </div>
    </div>
  )
}

function RecipeFormModal({ initial, forceCocktail, onClose, onSaved }: { initial: Recipe | null; forceCocktail: boolean; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState(initial?.name ?? '')
  const [basePersons, setBasePersons] = useState(initial?.basePersons ?? (forceCocktail ? 1 : 6))
  const [photoUrl, setPhotoUrl] = useState(initial?.photoUrl ?? '')
  const [tags, setTags] = useState<string[]>(initial?.tags ?? (forceCocktail ? ['cocktail'] : ['plat']))
  const [steps, setSteps] = useState<string[]>(() => {
    const list = (initial?.instructions ?? '').split('\n').map((l) => l.trim()).filter(Boolean)
    return list.length ? list : ['']
  })
  const [ingredients, setIngredients] = useState<DraftIngredient[]>(
    initial?.ingredients?.length
      ? initial.ingredients.map((ri) => ({ name: ri.ingredient?.canonicalName ?? '', quantity: ri.quantity, unit: ri.unit }))
      : [{ name: '', quantity: 0, unit: '' }],
  )

  function setIng(i: number, patch: Partial<DraftIngredient>) {
    setIngredients((list) => list.map((it, idx) => (idx === i ? { ...it, ...patch } : it)))
  }
  function setStep(i: number, val: string) { setSteps((list) => list.map((s, idx) => (idx === i ? val : s))) }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const finalTags = forceCocktail && !tags.includes('cocktail') ? ['cocktail', ...tags] : tags
    const body = {
      name, basePersons, coefficient: 1, photoUrl, tags: finalTags,
      instructions: steps.map((s) => s.trim()).filter(Boolean).join('\n'),
      ingredients: ingredients.filter((i) => i.name.trim()),
    }
    if (initial) await api.patch(`/recipes/${initial.id}`, body)
    else await api.post('/recipes', body)
    onSaved()
  }

  return (
    <Modal title={initial ? t('common.edit') : forceCocktail ? t('recipes.createCocktail') : t('recipes.create')} onClose={onClose} wide>
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="sm:col-span-2">
            <label className="label">{t('recipes.name')}</label>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div>
            <label className="label">{t('recipes.basePersons')}</label>
            <input className="input" type="number" min={1} value={basePersons} onChange={(e) => setBasePersons(+e.target.value)} />
          </div>
        </div>

        {!forceCocktail && (
          <div>
            <label className="label">{t('recipes.tags')}</label>
            <TagSelector tags={tags} onChange={setTags} />
          </div>
        )}

        <div>
          <label className="label">{t('profile.photo')}</label>
          <ImageUpload value={photoUrl} onChange={setPhotoUrl} />
        </div>

        <div>
          <label className="label">{t('recipes.ingredients')}</label>
          <div className="space-y-2">
            {ingredients.map((ing, i) => (
              <div key={i} className="grid grid-cols-[1fr_80px_100px_auto] gap-2">
                <IngredientInput value={ing.name} onChange={(v) => setIng(i, { name: v })} onPickUnit={(u) => setIng(i, { unit: u })} />
                <input className="input" type="number" step="0.1" placeholder="qté" value={ing.quantity || ''} onChange={(e) => setIng(i, { quantity: +e.target.value })} />
                <input className="input" placeholder="unité" value={ing.unit} onChange={(e) => setIng(i, { unit: e.target.value })} />
                <button type="button" className="btn-ghost" onClick={() => setIngredients((l) => l.filter((_, idx) => idx !== i))}><Trash2 size={15} /></button>
              </div>
            ))}
          </div>
          <button type="button" className="btn-ghost mt-2" onClick={() => setIngredients((l) => [...l, { name: '', quantity: 0, unit: '' }])}>
            <Plus size={15} /> {t('recipes.addIngredient')}
          </button>
        </div>

        <div>
          <label className="label">{t('recipes.instructions')} <span className="text-xs font-normal text-muted">({t('recipes.oneStepPerLine')})</span></label>
          <div className="space-y-2">
            {steps.map((step, i) => (
              <div key={i} className="flex items-center gap-2">
                <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-brand text-xs font-semibold text-brand-fg">{i + 1}</span>
                <input className="input" value={step} onChange={(e) => setStep(i, e.target.value)} placeholder={`${t('recipes.step')} ${i + 1}`} />
                <button type="button" className="btn-ghost" onClick={() => setSteps((l) => (l.length > 1 ? l.filter((_, idx) => idx !== i) : ['']))}><X size={15} /></button>
              </div>
            ))}
          </div>
          <button type="button" className="btn-ghost mt-2" onClick={() => setSteps((l) => [...l, ''])}><Plus size={15} /> {t('recipes.addStep')}</button>
        </div>

        <div className="flex justify-end gap-2">
          <button type="button" className="btn-ghost" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary">{initial ? t('common.save') : t('recipes.submitForApproval')}</button>
        </div>
      </form>
    </Modal>
  )
}
