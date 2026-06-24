import { FormEvent, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { KeyRound } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import PasswordFields, { isStrongPassword } from '../components/PasswordFields'

export default function ResetPasswordPage() {
  const { token = '' } = useParams()
  const { t } = useTranslation()
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.post('/auth/reset-password', { token, password })
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('auth.resetInvalid'))
    }
  }

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="card w-full max-w-sm p-8">
        <div className="mb-6 flex flex-col items-center gap-2">
          <KeyRound className="text-brand" size={32} />
          <h1 className="text-xl font-bold">{t('auth.resetTitle')}</h1>
        </div>
        {done ? (
          <div className="text-center">
            <p className="mb-4 text-sm text-success">{t('auth.resetDone')}</p>
            <Link to="/login" className="btn-primary">{t('auth.signIn')}</Link>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <PasswordFields password={password} confirm={confirm} setPassword={setPassword} setConfirm={setConfirm} />
            {error && <p className="text-sm text-danger">{error}</p>}
            <button className="btn-primary w-full" disabled={!isStrongPassword(password) || password !== confirm}>{t('common.save')}</button>
          </form>
        )}
      </div>
    </div>
  )
}
