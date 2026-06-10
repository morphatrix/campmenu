import { FormEvent, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Smartphone, Tent } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { ApiError } from '../lib/api'

export default function LoginPage() {
  const { t } = useTranslation()
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (user) navigate('/')
  }, [user, navigate])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(email, password)
      navigate('/')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erreur')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="card w-full max-w-sm p-8">
        <div className="mb-6 flex flex-col items-center gap-2">
          <Tent className="text-brand" size={36} />
          <h1 className="text-xl font-bold">CampMenu</h1>
          <p className="text-center text-sm text-muted">{t('app.tagline')}</p>
        </div>
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <label className="label">{t('auth.email')}</label>
            <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>
          <div>
            <label className="label">{t('auth.password')}</label>
            <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
          <button className="btn-primary w-full" disabled={busy}>{t('auth.signIn')}</button>
        </form>
        <div className="mt-4 text-center">
          <Link to="/forgot" className="text-sm text-brand hover:underline">{t('auth.forgot')}</Link>
        </div>
        <p className="mt-3 text-center text-xs text-muted">{t('auth.noAccount')}</p>
        <div className="mt-4 border-t border-border pt-3 text-center">
          <Link to="/install" className="inline-flex items-center gap-1 text-sm text-brand hover:underline">
            <Smartphone size={15} /> {t('install.link')}
          </Link>
        </div>
      </div>
    </div>
  )
}
