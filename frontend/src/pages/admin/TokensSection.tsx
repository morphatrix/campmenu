import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, KeyRound, Plus, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import type { ApiToken, NewApiToken } from '../../lib/types'

export default function TokensSection() {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<ApiToken[]>([])
  const [label, setLabel] = useState('')
  const [created, setCreated] = useState<NewApiToken | null>(null)
  const [copied, setCopied] = useState(false)

  async function load() {
    setTokens(await api.get<ApiToken[]>('/api-tokens'))
  }
  useEffect(() => { load() }, [])

  async function create() {
    if (!label.trim()) return
    const res = await api.post<NewApiToken>('/api-tokens', { label: label.trim() })
    setCreated(res)
    setLabel('')
    setCopied(false)
    load()
  }

  async function revoke(id: string) {
    if (!confirm(t('admin.tokenRevokeConfirm'))) return
    await api.del(`/api-tokens/${id}`)
    if (created?.id === id) setCreated(null)
    load()
  }

  async function copyToken() {
    if (!created) return
    try { await navigator.clipboard.writeText(created.token) } catch { /* clipboard may be blocked */ }
    setCopied(true)
  }

  return (
    <section className="card space-y-4 p-6">
      <p className="text-sm text-muted">{t('admin.tokenHint')}</p>

      <div className="flex flex-wrap items-end gap-2">
        <div>
          <label className="label">{t('admin.tokenLabel')}</label>
          <input className="input w-56" value={label} onChange={(e) => setLabel(e.target.value)} placeholder={t('admin.tokenLabelPlaceholder')} />
        </div>
        <button className="btn-primary" onClick={create} disabled={!label.trim()}>
          <Plus size={15} /> {t('admin.tokenCreate')}
        </button>
      </div>

      {created && (
        <div className="rounded-lg border border-border bg-surface p-3">
          <p className="mb-2 text-sm font-medium text-accent">{t('admin.tokenShowOnce')}</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 overflow-x-auto whitespace-nowrap rounded bg-card px-2 py-1 text-xs">{created.token}</code>
            <button className="btn-ghost" onClick={copyToken}>
              <Copy size={15} /> {copied ? t('admin.tokenCopied') : t('admin.tokenCopy')}
            </button>
          </div>
        </div>
      )}

      <ul className="divide-y divide-border text-sm">
        {tokens.map((tk) => (
          <li key={tk.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
            <span className="flex items-center gap-2">
              <KeyRound size={15} className="text-muted" />
              <span className="font-medium">{tk.label}</span>
              <span className="text-xs text-muted">
                {t('admin.tokenCreatedAt', { date: new Date(tk.createdAt).toLocaleDateString() })}
                {tk.lastUsedAt ? ` · ${t('admin.tokenLastUsed', { date: new Date(tk.lastUsedAt).toLocaleDateString() })}` : ` · ${t('admin.tokenNeverUsed')}`}
              </span>
            </span>
            <button className="btn-ghost text-danger" onClick={() => revoke(tk.id)} title={t('admin.tokenRevoke')}>
              <Trash2 size={15} />
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}
