import { FormEvent, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Tent } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../context/AuthContext'

export default function InvitePage() {
  const { code = '' } = useParams()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { refresh } = useAuth()
  const [valid, setValid] = useState<boolean | null>(null)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState<'confirm' | 'in' | null>(null)

  useEffect(() => {
    api
      .get<{ email: string; valid: boolean }>(`/invite/${code}`)
      .then((r) => {
        setValid(true)
        if (r.email) setEmail(r.email)
      })
      .catch(() => setValid(false))
  }, [code])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const res = await api.post<{ emailConfirmRequired?: boolean }>('/auth/register', {
        code, email, password, firstName, lastName,
      })
      if (res.emailConfirmRequired) {
        setDone('confirm')
      } else {
        await refresh()
        setDone('in')
        navigate('/')
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erreur')
    }
  }

  if (valid === false) {
    return <div className="grid min-h-screen place-items-center text-danger">{t('auth.inviteInvalid')}</div>
  }
  if (done === 'confirm') {
    return <div className="grid min-h-screen place-items-center px-4 text-center text-fg">{t('auth.confirmRequired')}</div>
  }

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <div className="card w-full max-w-sm p-8">
        <div className="mb-6 flex flex-col items-center gap-2">
          <Tent className="text-brand" size={36} />
          <h1 className="text-xl font-bold">{t('auth.register')}</h1>
        </div>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">{t('auth.firstName')}</label>
              <input className="input" value={firstName} onChange={(e) => setFirstName(e.target.value)} required />
            </div>
            <div>
              <label className="label">{t('auth.lastName')}</label>
              <input className="input" value={lastName} onChange={(e) => setLastName(e.target.value)} />
            </div>
          </div>
          <div>
            <label className="label">{t('auth.email')}</label>
            <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>
          <div>
            <label className="label">{t('auth.password')}</label>
            <input className="input" type="password" minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} required />
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
          <button className="btn-primary w-full">{t('auth.signUp')}</button>
        </form>
      </div>
    </div>
  )
}
