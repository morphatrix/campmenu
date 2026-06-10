import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '../lib/api'

export default function ConfirmPage() {
  const { token = '' } = useParams()
  const { t } = useTranslation()
  const [status, setStatus] = useState<'pending' | 'ok' | 'error'>('pending')

  useEffect(() => {
    api.get(`/auth/confirm/${token}`).then(() => setStatus('ok')).catch(() => setStatus('error'))
  }, [token])

  return (
    <div className="grid min-h-screen place-items-center px-4 text-center">
      <div className="card max-w-sm p-8">
        {status === 'pending' && <p className="text-muted">{t('auth.confirming')}</p>}
        {status === 'ok' && (
          <>
            <p className="mb-4 text-success">{t('auth.confirmed')}</p>
            <Link to="/login" className="btn-primary">{t('auth.signIn')}</Link>
          </>
        )}
        {status === 'error' && <p className="text-danger">{t('auth.inviteInvalid')}</p>}
      </div>
    </div>
  )
}
