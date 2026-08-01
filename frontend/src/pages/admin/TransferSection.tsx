import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Download, Upload } from 'lucide-react'
import { api } from '../../lib/api'
import { isCocktail } from '../../lib/types'
import type {
  Event, ImportCommitResult, ImportPreview, PreviewItem, Recipe, TransferBundle, User,
} from '../../lib/types'

type Mode = 'export' | 'import'

function toggle(set: Set<string>, id: string): Set<string> {
  const next = new Set(set)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  return next
}

function CheckList({ items, selected, onToggle, onSelectAll, label }: {
  items: { id: string; label: string }[]
  selected: Set<string>
  onToggle: (id: string) => void
  onSelectAll: (all: boolean) => void
  label: string
}) {
  const { t } = useTranslation()
  const allSelected = items.length > 0 && items.every((it) => selected.has(it.id))
  return (
    <div className="card p-4">
      <div className="mb-2 flex items-center justify-between">
        <h4 className="font-semibold">{label}</h4>
        <label className="flex items-center gap-1 text-xs">
          <input type="checkbox" checked={allSelected} onChange={(e) => onSelectAll(e.target.checked)} /> {t('admin.transferSelectAll')}
        </label>
      </div>
      <ul className="max-h-48 divide-y divide-border overflow-y-auto text-sm">
        {items.map((it) => (
          <li key={it.id} className="flex items-center gap-2 py-1">
            <input type="checkbox" checked={selected.has(it.id)} onChange={() => onToggle(it.id)} />
            <span className="truncate">{it.label}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function ExportPanel() {
  const { t } = useTranslation()
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [events, setEvents] = useState<Event[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [selRecipeIds, setSelRecipeIds] = useState<Set<string>>(new Set())
  const [selCocktailIds, setSelCocktailIds] = useState<Set<string>>(new Set())
  const [selEventIds, setSelEventIds] = useState<Set<string>>(new Set())
  const [selUserIds, setSelUserIds] = useState<Set<string>>(new Set())
  const [exporting, setExporting] = useState(false)

  useEffect(() => {
    api.get<Recipe[]>('/recipes').then(setRecipes)
    api.get<Event[]>('/events').then(setEvents)
    api.get<User[]>('/users').then(setUsers)
  }, [])

  const plainRecipes = recipes.filter((r) => !isCocktail(r))
  const cocktails = recipes.filter(isCocktail)

  async function doExport() {
    setExporting(true)
    try {
      const recipeIds = [...selRecipeIds, ...selCocktailIds]
      const bundle = await api.post<TransferBundle>('/export', {
        recipeIds, eventIds: [...selEventIds], userIds: [...selUserIds],
      })
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `campmenu-export-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <CheckList
          label={t('admin.transferRecipes')} items={plainRecipes.map((r) => ({ id: r.id, label: r.name }))}
          selected={selRecipeIds} onToggle={(id) => setSelRecipeIds((s) => toggle(s, id))}
          onSelectAll={(all) => setSelRecipeIds(all ? new Set(plainRecipes.map((r) => r.id)) : new Set())}
        />
        <CheckList
          label={t('admin.transferCocktails')} items={cocktails.map((r) => ({ id: r.id, label: r.name }))}
          selected={selCocktailIds} onToggle={(id) => setSelCocktailIds((s) => toggle(s, id))}
          onSelectAll={(all) => setSelCocktailIds(all ? new Set(cocktails.map((r) => r.id)) : new Set())}
        />
        <CheckList
          label={t('admin.transferEvents')} items={events.map((e) => ({ id: e.id, label: e.name }))}
          selected={selEventIds} onToggle={(id) => setSelEventIds((s) => toggle(s, id))}
          onSelectAll={(all) => setSelEventIds(all ? new Set(events.map((e) => e.id)) : new Set())}
        />
        <CheckList
          label={t('admin.transferUsers')} items={users.map((u) => ({ id: u.id, label: `${u.firstName} ${u.lastName} <${u.email}>` }))}
          selected={selUserIds} onToggle={(id) => setSelUserIds((s) => toggle(s, id))}
          onSelectAll={(all) => setSelUserIds(all ? new Set(users.map((u) => u.id)) : new Set())}
        />
      </div>
      <button className="btn-primary" onClick={doExport} disabled={exporting}>
        <Download size={15} /> {t('admin.transferDoExport')}
      </button>
    </div>
  )
}

function PreviewList({ label, items, selected, onToggle, onSelectAllConflicts }: {
  label: string
  items: PreviewItem[]
  selected: Set<string>
  onToggle: (key: string) => void
  onSelectAllConflicts: () => void
}) {
  const { t } = useTranslation()
  if (items.length === 0) return null
  return (
    <div className="card p-4">
      <div className="mb-2 flex items-center justify-between">
        <h4 className="font-semibold">{label}</h4>
        <button className="btn-ghost text-xs" onClick={onSelectAllConflicts}>{t('admin.transferSelectAllConflicts')}</button>
      </div>
      <ul className="max-h-48 divide-y divide-border overflow-y-auto text-sm">
        {items.map((it) => (
          <li key={it.key} className="flex items-center gap-2 py-1">
            <input type="checkbox" checked={selected.has(it.key)} onChange={() => onToggle(it.key)} />
            <span className="truncate">{it.label}</span>
            {it.exists && <span className="chip text-accent">{t('admin.transferExisting')}</span>}
          </li>
        ))}
      </ul>
    </div>
  )
}

function ImportPanel() {
  const { t } = useTranslation()
  const [bundle, setBundle] = useState<TransferBundle | null>(null)
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [selRecipeKeys, setSelRecipeKeys] = useState<Set<string>>(new Set())
  const [selEventKeys, setSelEventKeys] = useState<Set<string>>(new Set())
  const [selUserKeys, setSelUserKeys] = useState<Set<string>>(new Set())
  const [committing, setCommitting] = useState(false)
  const [result, setResult] = useState<ImportCommitResult | null>(null)

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const parsed = JSON.parse(await file.text()) as TransferBundle
    setBundle(parsed)
    setResult(null)
    const p = await api.post<ImportPreview>('/import/preview', parsed)
    setPreview(p)
    setSelRecipeKeys(new Set(p.recipes.filter((i) => !i.exists).map((i) => i.key)))
    setSelEventKeys(new Set(p.events.filter((i) => !i.exists).map((i) => i.key)))
    setSelUserKeys(new Set(p.users.filter((i) => !i.exists).map((i) => i.key)))
  }

  async function doCommit() {
    if (!bundle) return
    setCommitting(true)
    try {
      const res = await api.post<ImportCommitResult>('/import/commit', {
        bundle,
        selections: { recipes: [...selRecipeKeys], events: [...selEventKeys], users: [...selUserKeys] },
      })
      setResult(res)
    } finally {
      setCommitting(false)
    }
  }

  return (
    <div className="space-y-4">
      <label className="btn-ghost inline-flex w-fit cursor-pointer items-center gap-2">
        <Upload size={15} /> {t('admin.transferChooseFile')}
        <input type="file" accept="application/json" className="hidden" onChange={handleFile} />
      </label>

      {preview && (
        <div className="grid gap-4 sm:grid-cols-2">
          <PreviewList
            label={t('admin.transferRecipes')} items={preview.recipes.filter((i) => !isCocktail(bundle?.recipes.find((r) => r.name === i.key) ?? {}))}
            selected={selRecipeKeys} onToggle={(k) => setSelRecipeKeys((s) => toggle(s, k))}
            onSelectAllConflicts={() => setSelRecipeKeys((s) => new Set([...s, ...preview.recipes.filter((i) => i.exists).map((i) => i.key)]))}
          />
          <PreviewList
            label={t('admin.transferCocktails')} items={preview.recipes.filter((i) => isCocktail(bundle?.recipes.find((r) => r.name === i.key) ?? {}))}
            selected={selRecipeKeys} onToggle={(k) => setSelRecipeKeys((s) => toggle(s, k))}
            onSelectAllConflicts={() => setSelRecipeKeys((s) => new Set([...s, ...preview.recipes.filter((i) => i.exists).map((i) => i.key)]))}
          />
          <PreviewList
            label={t('admin.transferEvents')} items={preview.events}
            selected={selEventKeys} onToggle={(k) => setSelEventKeys((s) => toggle(s, k))}
            onSelectAllConflicts={() => setSelEventKeys((s) => new Set([...s, ...preview.events.filter((i) => i.exists).map((i) => i.key)]))}
          />
          <PreviewList
            label={t('admin.transferUsers')} items={preview.users}
            selected={selUserKeys} onToggle={(k) => setSelUserKeys((s) => toggle(s, k))}
            onSelectAllConflicts={() => setSelUserKeys((s) => new Set([...s, ...preview.users.filter((i) => i.exists).map((i) => i.key)]))}
          />
        </div>
      )}

      {preview && (
        <button className="btn-primary" onClick={doCommit} disabled={committing}>
          <Upload size={15} /> {committing ? t('admin.transferImporting') : t('admin.transferDoImport')}
        </button>
      )}

      {result && (
        <div className="card p-4 text-sm">
          <p className="text-success">
            {t('admin.transferResult', { users: result.importedUsers, recipes: result.importedRecipes, events: result.importedEvents })}
          </p>
          {result.skipped.length > 0 && (
            <div className="mt-2">
              <p className="text-accent">{t('admin.transferSkipped')}</p>
              <ul className="list-inside list-disc text-muted">
                {result.skipped.map((s, i) => <li key={i}>{s}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function TransferSection() {
  const { t } = useTranslation()
  const [mode, setMode] = useState<Mode>('export')

  return (
    <section className="space-y-4">
      <div className="flex gap-2">
        <button
          onClick={() => setMode('export')}
          className={`rounded-lg px-3 py-1.5 text-sm font-medium ${mode === 'export' ? 'bg-brand text-brand-fg' : 'bg-card text-muted hover:text-fg'}`}
        >
          {t('admin.transferExport')}
        </button>
        <button
          onClick={() => setMode('import')}
          className={`rounded-lg px-3 py-1.5 text-sm font-medium ${mode === 'import' ? 'bg-brand text-brand-fg' : 'bg-card text-muted hover:text-fg'}`}
        >
          {t('admin.transferImport')}
        </button>
      </div>
      {mode === 'export' ? <ExportPanel /> : <ImportPanel />}
    </section>
  )
}
