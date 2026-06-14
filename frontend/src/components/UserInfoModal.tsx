import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy, Lock } from 'lucide-react'
import { api } from '../lib/api'
import Modal from './Modal'
import Avatar from './Avatar'

type InfoUser = {
  id?: string
  firstName?: string
  lastName?: string
  nickname?: string
  email?: string
  photoUrl?: string
  iban?: string
  ibanHidden?: boolean
}

// UserInfoModal is a lightweight public profile card (name, nickname, photo,
// IBAN). The IBAN respects the owner's visibility: when withheld it offers to
// request access. Private fields (weight, birth date) are never shown.
export default function UserInfoModal({ user, onClose }: { user: InfoUser; onClose: () => void }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const [requested, setRequested] = useState(false)
  const fullName = `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim() || user.email || ''

  async function copyIban() {
    if (!user.iban) return
    try {
      await navigator.clipboard.writeText(user.iban)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch { /* clipboard may be blocked */ }
  }

  async function requestAccess() {
    if (!user.id) return
    try {
      await api.post(`/users/${user.id}/iban-request`)
      setRequested(true)
    } catch { /* ignore */ }
  }

  return (
    <Modal title={fullName} onClose={onClose}>
      <div className="flex flex-col items-center gap-3 text-center">
        <Avatar user={user} size={96} />
        <div>
          <p className="text-lg font-semibold">{fullName}</p>
          {user.nickname && <p className="text-sm text-muted">« {user.nickname} »</p>}
        </div>
        {user.iban ? (
          <button
            onClick={copyIban}
            className="flex max-w-full items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-surface"
            title={t('mobile.copyIban')}
          >
            {copied ? <Check size={15} className="shrink-0 text-success" /> : <Copy size={15} className="shrink-0" />}
            <span className="truncate font-mono">{copied ? t('mobile.ibanCopied') : user.iban}</span>
          </button>
        ) : user.ibanHidden ? (
          requested ? (
            <p className="text-sm text-success">{t('iban.requestSent')}</p>
          ) : (
            <button
              onClick={requestAccess}
              className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-surface"
            >
              <Lock size={14} /> {t('iban.requestAccess')}
            </button>
          )
        ) : null}
      </div>
    </Modal>
  )
}
