import { useEffect, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BadgeCheck, Ban, Copy, Database, Eye, KeyRound, Loader2, Mail, Pencil, Plus, ShieldCheck, Sparkles, Trash2, UserCog } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import Modal from '../components/Modal'
import Avatar from '../components/Avatar'
import { isAdmin as roleIsAdmin, isStaff } from '../lib/types'
import { PALETTES } from '../lib/appearance'
import type { Invite, Role, User } from '../lib/types'

type Section = 'invites' | 'users' | 'settings'

export default function AdminPage() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const admin = roleIsAdmin(user)
  const [section, setSection] = useState<Section>('invites')

  if (!isStaff(user)) return <Navigate to="/" replace />

  const tabs: { id: Section; label: string }[] = [
    { id: 'invites', label: t('admin.invites') },
    { id: 'users', label: t('admin.users') },
    ...(admin ? [{ id: 'settings' as Section, label: t('admin.settings') }] : []),
  ]

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('admin.title')}</h1>
      <div className="flex gap-2">
        {tabs.map((tb) => (
          <button
            key={tb.id}
            onClick={() => setSection(tb.id)}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium ${
              section === tb.id ? 'bg-brand text-brand-fg' : 'bg-card text-muted hover:text-fg'
            }`}
          >
            {tb.label}
          </button>
        ))}
      </div>

      {section === 'invites' && <InvitesSection admin={admin} />}
      {section === 'users' && <UsersSection admin={admin} />}
      {admin && section === 'settings' && <SettingsSection />}
    </div>
  )
}

function isExhausted(inv: Invite): boolean {
  if (inv.revoked) return true
  if (inv.maxUses > 0 && inv.useCount >= inv.maxUses) return true
  if (inv.expiresAt && new Date(inv.expiresAt).getTime() < Date.now()) return true
  return false
}

function InvitesSection({ admin }: { admin: boolean }) {
  const { t } = useTranslation()
  const [invites, setInvites] = useState<Invite[]>([])
  const [role, setRole] = useState<Role>('USER')
  const [maxUses, setMaxUses] = useState(0)
  const [expiresInDays, setExpiresInDays] = useState(0)
  const [lastLink, setLastLink] = useState('')

  async function load() {
    setInvites(await api.get<Invite[]>('/invites'))
  }
  useEffect(() => { load() }, [])

  async function createInvite() {
    const res = await api.post<{ link: string }>('/invites', { role, maxUses, expiresInDays })
    setLastLink(res.link)
    load()
  }
  async function revoke(inv: Invite) {
    await api.post(`/invites/${inv.id}/revoke`)
    load()
  }
  async function showAndCopy(inv: Invite) {
    const link = `${window.location.origin}/invite/${inv.code}`
    setLastLink(link)
    try { await navigator.clipboard.writeText(link) } catch { /* clipboard may be blocked */ }
  }

  return (
    <section className="card p-6">
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div>
          <label className="label">{t('admin.role')}</label>
          <select className="input w-40" value={role} onChange={(e) => setRole(e.target.value as Role)}>
            <option value="USER">{t('roles.USER')}</option>
            <option value="COLLABORATOR">{t('roles.COLLABORATOR')}</option>
            {admin && <option value="ADMIN">{t('roles.ADMIN')}</option>}
          </select>
        </div>
        <div>
          <label className="label">{t('admin.maxUses')}</label>
          <input className="input w-44" type="number" min={0} value={maxUses} onChange={(e) => setMaxUses(+e.target.value)} />
        </div>
        <div>
          <label className="label">{t('admin.expiresDays')}</label>
          <input className="input w-52" type="number" min={0} value={expiresInDays} onChange={(e) => setExpiresInDays(+e.target.value)} />
        </div>
        <button className="btn-primary" onClick={createInvite}>
          <Plus size={16} /> {t('admin.createInvite')}
        </button>
      </div>
      {lastLink && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-border bg-surface p-2 text-sm">
          <code className="flex-1 truncate">{lastLink}</code>
          <button className="btn-ghost" onClick={() => navigator.clipboard.writeText(lastLink)}>
            <Copy size={15} /> {t('admin.copyLink')}
          </button>
        </div>
      )}
      <div className="overflow-x-auto">
        <ul className="min-w-[640px] divide-y divide-border text-sm">
          {invites.map((inv) => (
            <li key={inv.id} className="grid grid-cols-[minmax(0,1fr)_120px_180px_84px_84px] items-center gap-3 py-2">
              <code className="truncate text-muted">{inv.code}</code>
              <span className="chip justify-self-start">{t(`roles.${inv.role}`)}</span>
              <span className="text-muted">
                {inv.useCount}{inv.maxUses > 0 ? `/${inv.maxUses}` : ''} {t('admin.uses')}
                {inv.expiresAt ? ` · ${new Date(inv.expiresAt).toLocaleDateString()}` : ''}
              </span>
              <span className={`font-medium ${isExhausted(inv) ? 'text-danger' : 'text-success'}`}>
                {inv.revoked ? t('admin.revoked') : isExhausted(inv) ? t('admin.exhausted') : t('admin.active')}
              </span>
              <span className="flex justify-end gap-1">
                {!isExhausted(inv) && (
                  <>
                    <button className="btn-ghost" onClick={() => showAndCopy(inv)} title={t('admin.copyLink')}><Copy size={15} /></button>
                    <button className="btn-ghost text-danger" onClick={() => revoke(inv)} title={t('admin.revoke')}><Ban size={15} /></button>
                  </>
                )}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

const ROLE_TONE: Record<string, string> = { ADMIN: 'text-danger', COLLABORATOR: 'text-brand', USER: 'text-muted' }

function UsersSection({ admin }: { admin: boolean }) {
  const { t } = useTranslation()
  const { user: me, impersonate } = useAuth()
  const navigate = useNavigate()
  const [users, setUsers] = useState<User[]>([])
  const [editing, setEditing] = useState<User | null>(null)
  const [resetting, setResetting] = useState<User | null>(null)
  const [viewing, setViewing] = useState<User | null>(null)

  async function load() {
    setUsers(await api.get<User[]>('/users'))
  }
  useEffect(() => { load() }, [])

  async function remove(u: User) {
    if (!confirm(t('admin.confirmDelete'))) return
    await api.del(`/users/${u.id}`)
    load()
  }
  async function resend(u: User) {
    await api.post(`/users/${u.id}/resend-confirmation`)
    alert(t('admin.resent'))
  }
  async function confirmAccount(u: User) {
    await api.post(`/users/${u.id}/confirm`)
    load()
  }
  async function promoteCollaborator(u: User) {
    await api.post(`/users/${u.id}/promote-collaborator`)
    load()
  }
  async function impersonateUser(u: User) {
    await impersonate(u.id)
    navigate('/')
  }

  return (
    <section className="card p-6">
      <ul className="divide-y divide-border text-sm">
        {users.map((u) => (
          <li key={u.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
            <span className="flex min-w-0 items-center gap-2">
              <Avatar user={u} size={28} />
              <span className="min-w-0">
                <span className="font-medium">{u.firstName} {u.lastName}</span>{' '}
                {u.nickname && <span className="text-muted">« {u.nickname} »</span>}{' '}
                <span className="text-muted">· {u.email}</span>
                {!u.emailConfirmed && <span className="chip ml-2 text-accent">non confirmé</span>}
              </span>
            </span>
            <div className="flex items-center gap-2">
              <span className={`chip ${ROLE_TONE[u.role] ?? ''}`}>{t(`roles.${u.role}`)}</span>
              {admin ? (
                <>
                  {u.id !== me?.id && (
                    <button className="btn-ghost" onClick={() => impersonateUser(u)} title={t('impersonate.as')}><UserCog size={15} /></button>
                  )}
                  {!u.emailConfirmed && (
                    <>
                      <button className="btn-ghost text-success" onClick={() => confirmAccount(u)} title={t('admin.confirmAccount')}><BadgeCheck size={15} /></button>
                      <button className="btn-ghost text-accent" onClick={() => resend(u)} title={t('admin.resendConfirmation')}><Mail size={15} /></button>
                    </>
                  )}
                  <button className="btn-ghost" onClick={() => setEditing(u)} title={t('common.edit')}><Pencil size={15} /></button>
                  <button className="btn-ghost" onClick={() => setResetting(u)} title={t('admin.resetPassword')}><KeyRound size={15} /></button>
                  <button className="btn-ghost text-danger" onClick={() => remove(u)} title={t('common.delete')}><Trash2 size={15} /></button>
                </>
              ) : (
                <>
                  <button className="btn-ghost" onClick={() => setViewing(u)} title={t('admin.viewProfile')}><Eye size={15} /></button>
                  {u.role === 'USER' && (
                    <button className="btn-ghost text-brand" onClick={() => promoteCollaborator(u)} title={t('admin.promoteCollaborator')}><ShieldCheck size={15} /></button>
                  )}
                </>
              )}
            </div>
          </li>
        ))}
      </ul>

      {editing && <EditUserModal user={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); load() }} />}
      {resetting && <ResetPasswordModal user={resetting} onClose={() => setResetting(null)} />}
      {viewing && <ViewUserModal user={viewing} onClose={() => setViewing(null)} />}
    </section>
  )
}

function ViewUserModal({ user, onClose }: { user: User; onClose: () => void }) {
  const { t } = useTranslation()
  const row = (label: string, value?: string | number | null) =>
    value ? <div className="flex justify-between gap-4 py-1"><span className="text-muted">{label}</span><span className="font-medium">{value}</span></div> : null
  return (
    <Modal title={`${user.firstName} ${user.lastName}`} onClose={onClose}>
      <div className="text-sm">
        {row(t('profile.nickname'), user.nickname)}
        {row(t('auth.email'), user.email)}
        {row(t('admin.role'), t(`roles.${user.role}`))}
        {row(t('profile.birthDate'), user.birthDate ? new Date(user.birthDate).toLocaleDateString() : null)}
        {row(t('profile.shoeSize'), user.shoeSize)}
        {row(t('profile.weight'), user.weight)}
        {row(t('profile.iban'), user.iban)}
      </div>
    </Modal>
  )
}

function EditUserModal({ user, onClose, onSaved }: { user: User; onClose: () => void; onSaved: () => void }) {
  const { t } = useTranslation()
  const [firstName, setFirstName] = useState(user.firstName)
  const [lastName, setLastName] = useState(user.lastName)
  const [nickname, setNickname] = useState(user.nickname)
  const [iban, setIban] = useState(user.iban)
  const [email, setEmail] = useState(user.email)
  const [role, setRole] = useState<Role>(user.role)
  const [error, setError] = useState('')

  async function save() {
    try {
      await api.patch(`/users/${user.id}`, { firstName, lastName, nickname, iban, email, role })
      onSaved()
    } catch (e: any) {
      setError(e?.message ?? 'Erreur')
    }
  }

  return (
    <Modal title={t('admin.editUser')} onClose={onClose}>
      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div><label className="label">{t('auth.firstName')}</label><input className="input" value={firstName} onChange={(e) => setFirstName(e.target.value)} /></div>
          <div><label className="label">{t('auth.lastName')}</label><input className="input" value={lastName} onChange={(e) => setLastName(e.target.value)} /></div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div><label className="label">{t('profile.nickname')}</label><input className="input" value={nickname} onChange={(e) => setNickname(e.target.value)} /></div>
          <div><label className="label">{t('profile.iban')}</label><input className="input" value={iban} onChange={(e) => setIban(e.target.value)} /></div>
        </div>
        <div><label className="label">{t('auth.email')}</label><input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} /></div>
        <div><label className="label">{t('admin.role')}</label>
          <select className="input" value={role} onChange={(e) => setRole(e.target.value as Role)}>
            <option value="USER">{t('roles.USER')}</option>
            <option value="COLLABORATOR">{t('roles.COLLABORATOR')}</option>
            <option value="ADMIN">{t('roles.ADMIN')}</option>
          </select>
        </div>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-2">
          <button className="btn-ghost" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={save}>{t('common.save')}</button>
        </div>
      </div>
    </Modal>
  )
}

function ResetPasswordModal({ user, onClose }: { user: User; onClose: () => void }) {
  const { t } = useTranslation()
  const [password, setPassword] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  async function save() {
    try {
      await api.post(`/users/${user.id}/reset-password`, { password })
      setDone(true)
    } catch (e: any) {
      setError(e?.message ?? 'Erreur')
    }
  }

  return (
    <Modal title={`${t('admin.resetPassword')} · ${user.firstName}`} onClose={onClose}>
      <div className="space-y-3">
        <div><label className="label">{t('admin.newPassword')}</label><input className="input" type="text" minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="min. 8 caractères" /></div>
        {error && <p className="text-sm text-danger">{error}</p>}
        {done ? (
          <p className="text-sm text-success">{t('admin.saved')}</p>
        ) : (
          <div className="flex justify-end gap-2 pt-2">
            <button className="btn-ghost" onClick={onClose}>{t('common.cancel')}</button>
            <button className="btn-primary" onClick={save} disabled={password.length < 8}>{t('common.save')}</button>
          </div>
        )}
      </div>
    </Modal>
  )
}

function SettingsSection() {
  const { t } = useTranslation()
  const [s, setS] = useState<Record<string, string>>({})
  const [saved, setSaved] = useState(false)
  const [testEmail, setTestEmail] = useState<{ state: 'idle' | 'sending' | 'sent' | 'error'; msg?: string }>({ state: 'idle' })
  const [aiPrompt, setAiPrompt] = useState('')
  const [aiTest, setAiTest] = useState<{ state: 'idle' | 'testing' | 'done' | 'error'; text?: string }>({ state: 'idle' })

  useEffect(() => {
    api.get<Record<string, string>>('/settings').then(setS)
  }, [])

  function set(key: string, value: string) {
    setS((cur) => ({ ...cur, [key]: value }))
    setSaved(false)
  }

  async function save() {
    const updated = await api.patch<Record<string, string>>('/settings', s)
    setS(updated)
    setSaved(true)
  }

  async function sendTestEmail() {
    setTestEmail({ state: 'sending' })
    try {
      const res = await api.post<{ ok: boolean; to?: string; error?: string }>('/settings/test-email')
      if (res.ok) {
        setTestEmail({ state: 'sent', msg: t('settings.testEmailSent', { to: res.to }) })
      } else {
        setTestEmail({ state: 'error', msg: res.error ?? t('settings.testEmailFailed') })
      }
    } catch (e: any) {
      setTestEmail({ state: 'error', msg: e?.message ?? t('settings.testEmailFailed') })
    }
  }

  async function testAI() {
    setAiTest({ state: 'testing' })
    try {
      const res = await api.post<{ ok: boolean; response?: string; error?: string }>('/settings/ai-test', { prompt: aiPrompt })
      if (res.ok) setAiTest({ state: 'done', text: res.response ?? '' })
      else setAiTest({ state: 'error', text: res.error ?? t('settings.aiTestFailed') })
    } catch (e: any) {
      setAiTest({ state: 'error', text: e?.message ?? t('settings.aiTestFailed') })
    }
  }

  const field = (key: string, label: string, type = 'text') => (
    <div>
      <label className="label">{label}</label>
      <input className="input" type={type} value={s[key] ?? ''} onChange={(e) => set(key, e.target.value)} />
    </div>
  )

  return (
    <div className="space-y-6">
      <section className="card p-6">
        <h3 className="mb-4 font-semibold">{t('settings.branding')}</h3>
        <div className="grid gap-4 sm:grid-cols-2">
          {field('SITE_NAME', t('settings.siteName'))}
          {field('LOGO_URL', t('settings.logoUrl'))}
          <div>
            <label className="label">{t('settings.defaultTheme')}</label>
            <select className="input" value={s['DEFAULT_THEME'] ?? 'auto'} onChange={(e) => set('DEFAULT_THEME', e.target.value)}>
              <option value="light">{t('themes.light')}</option>
              <option value="dark">{t('themes.dark')}</option>
              <option value="auto">{t('themes.auto')}</option>
            </select>
          </div>
          <div>
            <label className="label">{t('settings.defaultPalette')}</label>
            <select className="input" value={s['DEFAULT_PALETTE'] ?? 'default'} onChange={(e) => set('DEFAULT_PALETTE', e.target.value)}>
              {PALETTES.map((p) => <option key={p} value={p}>{t(`palettes.${p}`)}</option>)}
            </select>
          </div>
        </div>
      </section>

      <section className="card p-6">
        <h3 className="mb-4 font-semibold">{t('settings.access')}</h3>
        <div className="grid gap-4 sm:grid-cols-2">
          {field('APP_URL', t('settings.appUrl'))}
          {field('CORS_ORIGINS', t('settings.corsOrigins'))}
          <label className="flex items-center gap-2 self-end pb-2">
            <input type="checkbox" checked={s['EMAIL_CONFIRM_REQUIRED'] === 'true'} onChange={(e) => set('EMAIL_CONFIRM_REQUIRED', String(e.target.checked))} />
            <span className="text-sm">{t('settings.emailConfirm')}</span>
          </label>
        </div>
      </section>

      <section className="card p-6">
        <h3 className="mb-4 font-semibold">{t('settings.smtp')}</h3>
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <button className="btn-ghost" onClick={sendTestEmail} disabled={testEmail.state === 'sending'}>
            <Mail size={15} /> {testEmail.state === 'sending' ? t('settings.testEmailSending') : t('settings.testEmail')}
          </button>
          {testEmail.state === 'sent' && <span className="text-sm text-success">{testEmail.msg}</span>}
          {testEmail.state === 'error' && <span className="text-sm text-danger">{testEmail.msg}</span>}
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {field('SMTP_HOST', t('settings.smtpHost'))}
          {field('SMTP_PORT', t('settings.smtpPort'))}
          {field('SMTP_USER', t('settings.smtpUser'))}
          {field('SMTP_PASS', t('settings.smtpPass'), 'password')}
          {field('SMTP_FROM', t('settings.smtpFrom'))}
        </div>
      </section>

      <section className="card p-6">
        <h3 className="mb-1 font-semibold">{t('settings.aiTitle')}</h3>
        <p className="mb-4 text-xs text-muted">{t('settings.aiHint')}</p>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label">{t('settings.aiProvider')}</label>
            <select className="input" value={s['AI_PROVIDER'] ?? ''} onChange={(e) => set('AI_PROVIDER', e.target.value)}>
              <option value="">{t('settings.aiDisabled')}</option>
              <option value="ollama">Ollama</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Claude (Anthropic)</option>
            </select>
          </div>
          {field('AI_MODEL', t('settings.aiModel'))}
          {field('AI_BASE_URL', t('settings.aiBaseUrl'))}
          {field('AI_API_KEY', t('settings.aiApiKey'), 'password')}
        </div>

        <div className="mt-4 border-t border-border pt-4">
          <label className="label">{t('settings.aiTestPrompt')}</label>
          <textarea className="input min-h-16" value={aiPrompt} onChange={(e) => setAiPrompt(e.target.value)} placeholder={t('settings.aiTestPlaceholder')} />
          <div className="mt-2 flex items-center gap-3">
            <button className="btn-ghost" onClick={testAI} disabled={aiTest.state === 'testing'}>
              {aiTest.state === 'testing' ? <Loader2 size={15} className="animate-spin" /> : <Sparkles size={15} />} {t('settings.aiTest')}
            </button>
            <span className="text-xs text-muted">{t('settings.aiTestHint')}</span>
          </div>
          {aiTest.state === 'error' && <p className="mt-2 text-sm text-danger">{aiTest.text}</p>}
          {aiTest.state === 'done' && <pre className="mt-2 max-h-60 overflow-auto whitespace-pre-wrap rounded-lg bg-surface p-3 text-sm">{aiTest.text}</pre>}
        </div>
      </section>

      <div className="flex items-center gap-3">
        <button className="btn-primary" onClick={save}>{t('common.save')}</button>
        {saved && <span className="text-sm text-success">{t('admin.saved')}</span>}
      </div>

      <DatabaseSection />
    </div>
  )
}

// DatabaseSection configures an optional external database. The DSN is stored in
// the primary database and applied on the next restart (the connection is bound
// at boot), so saving here shows a "restart required" hint.
function DatabaseSection() {
  const { t } = useTranslation()
  const [dsn, setDsn] = useState('')
  const [usingExternal, setUsingExternal] = useState(false)
  const [status, setStatus] = useState<{ state: 'idle' | 'saving' | 'saved' | 'error'; msg?: string }>({ state: 'idle' })

  useEffect(() => {
    api.get<{ externalDsn: string; usingExternal: boolean }>('/settings/db').then((d) => {
      setDsn(d.externalDsn ?? '')
      setUsingExternal(d.usingExternal)
    })
  }, [])

  async function save() {
    setStatus({ state: 'saving' })
    try {
      await api.patch('/settings/db', { externalDsn: dsn })
      setStatus({ state: 'saved', msg: t('settings.dbRestartNote') })
    } catch (e: any) {
      setStatus({ state: 'error', msg: e?.message ?? 'Erreur' })
    }
  }

  return (
    <section className="card p-6">
      <h3 className="mb-1 flex items-center gap-2 font-semibold"><Database size={16} /> {t('settings.dbTitle')}</h3>
      <p className="mb-3 text-xs text-muted">{t('settings.dbHint')}</p>
      <div className="mb-3">
        {usingExternal
          ? <span className="chip text-success">{t('settings.dbUsingExternal')}</span>
          : <span className="chip text-muted">{t('settings.dbUsingPrimary')}</span>}
      </div>
      <label className="label">{t('settings.dbExternalDsn')}</label>
      <input
        className="input font-mono text-xs"
        value={dsn}
        onChange={(e) => setDsn(e.target.value)}
        placeholder="host=db.example.com port=5432 user=campmenu password=… dbname=campmenu sslmode=require"
      />
      <div className="mt-3 flex items-center gap-3">
        <button className="btn-primary" onClick={save} disabled={status.state === 'saving'}>{t('common.save')}</button>
        {status.state === 'saved' && <span className="text-sm text-success">{status.msg}</span>}
        {status.state === 'error' && <span className="text-sm text-danger">{status.msg}</span>}
      </div>
    </section>
  )
}
