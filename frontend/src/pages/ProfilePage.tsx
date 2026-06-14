import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Smartphone } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import { PALETTES, THEMES } from '../lib/appearance'
import ImageUpload from '../components/ImageUpload'
import type { User } from '../lib/types'

export default function ProfilePage() {
  const { t } = useTranslation()
  const { user, setUser } = useAuth()
  const [form, setForm] = useState(() => ({ ...user! }))
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  function set<K extends keyof User>(key: K, value: User[K]) {
    setForm((f) => ({ ...f, [key]: value }))
    setSaved(false)
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const updated = await api.patch<User>('/me', {
        firstName: form.firstName,
        lastName: form.lastName,
        nickname: form.nickname,
        iban: form.iban,
        // Send ISO (the backend expects RFC3339); the date input is YYYY-MM-DD.
        birthDate: form.birthDate ? new Date(`${form.birthDate.slice(0, 10)}T00:00:00Z`).toISOString() : null,
        shoeSize: form.shoeSize ?? null,
        weight: form.weight ?? null,
        photoUrl: form.photoUrl,
        theme: form.theme,
        colorPalette: form.colorPalette,
        colorblindMode: form.colorblindMode,
        language: form.language,
      })
      setUser(updated)
      setSaved(true)
    } catch (err: any) {
      setError(err?.message ?? 'Erreur')
    }
  }

  return (
    <form onSubmit={onSubmit} className="mx-auto max-w-2xl">
      <div className="mb-6 flex items-center gap-3">
        <h1 className="text-2xl font-bold">{t('profile.title')}</h1>
        {user && <span className="chip text-brand">{t(`roles.${user.role}`)}</span>}
        <Link to="/install" className="ml-auto inline-flex items-center gap-1 text-sm text-brand hover:underline">
          <Smartphone size={15} /> {t('install.link')}
        </Link>
      </div>
      <div className="card grid gap-4 p-6 sm:grid-cols-2">
        <div>
          <label className="label">{t('auth.firstName')}</label>
          <input className="input" value={form.firstName} onChange={(e) => set('firstName', e.target.value)} />
        </div>
        <div>
          <label className="label">{t('auth.lastName')}</label>
          <input className="input" value={form.lastName} onChange={(e) => set('lastName', e.target.value)} />
        </div>
        <div>
          <label className="label">{t('profile.nickname')}</label>
          <input className="input" value={form.nickname} onChange={(e) => set('nickname', e.target.value)} placeholder={`${form.firstName} ${form.lastName}`.trim()} />
        </div>
        <div>
          <label className="label">{t('profile.iban')}</label>
          <input className="input" value={form.iban} onChange={(e) => set('iban', e.target.value)} placeholder="FR76…" />
        </div>
        <div>
          <label className="label">{t('profile.birthDate')}</label>
          <input className="input" type="date" value={form.birthDate?.slice(0, 10) ?? ''} onChange={(e) => set('birthDate', e.target.value)} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">{t('profile.shoeSize')}</label>
            <input className="input" type="number" step="0.5" value={form.shoeSize ?? ''} onChange={(e) => set('shoeSize', e.target.value ? +e.target.value : null)} />
          </div>
          <div>
            <label className="label">{t('profile.weight')}</label>
            <input className="input" type="number" step="0.5" value={form.weight ?? ''} onChange={(e) => set('weight', e.target.value ? +e.target.value : null)} />
          </div>
        </div>
        <div className="sm:col-span-2">
          <label className="label">{t('profile.photo')}</label>
          <ImageUpload value={form.photoUrl} onChange={(url) => set('photoUrl', url)} circle />
        </div>

        <div>
          <label className="label">{t('profile.theme')}</label>
          <select className="input" value={form.theme} onChange={(e) => set('theme', e.target.value)}>
            {THEMES.map((th) => <option key={th} value={th}>{t(`themes.${th}`)}</option>)}
          </select>
        </div>
        <div>
          <label className="label">{t('profile.palette')}</label>
          <select className="input" value={form.colorPalette} onChange={(e) => set('colorPalette', e.target.value)}>
            {PALETTES.map((p) => <option key={p} value={p}>{t(`palettes.${p}`)}</option>)}
          </select>
        </div>
        <div>
          <label className="label">{t('profile.language')}</label>
          <select className="input" value={form.language} onChange={(e) => set('language', e.target.value)}>
            <option value="fr">Français</option>
            <option value="en">English</option>
          </select>
        </div>
        <label className="flex items-center gap-2 self-end pb-2">
          <input type="checkbox" checked={form.colorblindMode} onChange={(e) => set('colorblindMode', e.target.checked)} />
          <span className="text-sm">{t('profile.colorblind')}</span>
        </label>

        <div className="flex items-center gap-3 sm:col-span-2">
          <button className="btn-primary">{t('profile.save')}</button>
          {saved && <span className="text-sm text-success">{t('profile.saved')}</span>}
          {error && <span className="text-sm text-danger">{error}</span>}
        </div>
      </div>
    </form>
  )
}
