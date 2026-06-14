import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell, Check, X } from 'lucide-react'
import { api } from '../lib/api'
import { useLive } from '../context/LiveContext'
import { displayName } from '../lib/types'
import type { IbanRequest } from '../lib/types'
import Modal from './Modal'
import Avatar from './Avatar'

// IbanRequestsBell shows incoming IBAN access requests with a badge; tapping it
// opens a list to accept or deny each one.
export default function IbanRequestsBell() {
  const { t } = useTranslation()
  const [reqs, setReqs] = useState<IbanRequest[]>([])
  const [open, setOpen] = useState(false)

  async function load() {
    try {
      setReqs(await api.get<IbanRequest[]>('/iban-requests'))
    } catch { /* ignore (e.g. transient) */ }
  }
  useEffect(() => { load() }, [])
  useLive(load)

  async function accept(id: string) { await api.post(`/iban-requests/${id}/accept`); load() }
  async function deny(id: string) { await api.post(`/iban-requests/${id}/deny`); load() }

  const count = reqs.length

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="relative rounded-lg px-2 py-1 text-muted hover:text-fg"
        title={t('iban.requestsTitle')}
        aria-label={t('iban.requestsTitle')}
      >
        <Bell size={18} />
        {count > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-[10px] font-bold text-white">
            {count}
          </span>
        )}
      </button>
      {open && (
        <Modal title={t('iban.requestsTitle')} onClose={() => setOpen(false)}>
          {reqs.length === 0 ? (
            <p className="text-sm text-muted">{t('iban.noRequests')}</p>
          ) : (
            <ul className="space-y-2">
              {reqs.map((req) => (
                <li key={req.id} className="flex items-center gap-2 rounded-lg bg-surface px-2 py-2 text-sm">
                  <Avatar user={req.requester} size={28} />
                  <span className="min-w-0 flex-1 truncate">{t('iban.requestsFrom', { name: displayName(req.requester) })}</span>
                  <button onClick={() => accept(req.id)} className="btn-ghost text-success" title={t('iban.accept')}><Check size={16} /></button>
                  <button onClick={() => deny(req.id)} className="btn-ghost text-danger" title={t('iban.deny')}><X size={16} /></button>
                </li>
              ))}
            </ul>
          )}
        </Modal>
      )}
    </>
  )
}
