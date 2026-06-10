import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { KeyRound } from 'lucide-react'
import { api } from '../lib/api'

export default function ForgotPasswordPage() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    await api.post('/auth/forgot-password', { email })
    setSent(true)
  }

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="card w-full max-w-sm p-8">
        <div className="mb-6 flex flex-col items-center gap-2">
          <KeyRound className="text-brand" size={32} />
          <h1 className="text-xl font-bold">{t('auth.forgotTitle')}</h1>
        </div>
        {sent ? (
          <p className="text-center text-sm text-success">{t('auth.forgotSent')}</p>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <div>
              <label className="label">{t('auth.email')}</label>
              <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            </div>
            <button className="btn-primary w-full">{t('auth.send')}</button>
          </form>
        )}
        <div className="mt-4 text-center">
          <Link to="/login" className="text-sm text-brand hover:underline">{t('auth.backToLogin')}</Link>
        </div>
      </div>
    </div>
  )
}
