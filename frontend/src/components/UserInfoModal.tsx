import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import Modal from './Modal'
import Avatar from './Avatar'

type InfoUser = {
  firstName?: string
  lastName?: string
  nickname?: string
  email?: string
  photoUrl?: string
  iban?: string
}

// UserInfoModal is a lightweight public profile card (name, nickname, photo,
// IBAN). It deliberately omits private fields (weight, birth date).
export default function UserInfoModal({ user, onClose }: { user: InfoUser; onClose: () => void }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const fullName = `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim() || user.email || ''

  async function copyIban() {
    if (!user.iban) return
    try {
      await navigator.clipboard.writeText(user.iban)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch { /* clipboard may be blocked */ }
  }

  return (
    <Modal title={fullName} onClose={onClose}>
      <div className="flex flex-col items-center gap-3 text-center">
        <Avatar user={user} size={96} />
        <div>
          <p className="text-lg font-semibold">{fullName}</p>
          {user.nickname && <p className="text-sm text-muted">« {user.nickname} »</p>}
        </div>
        {user.iban && (
          <button
            onClick={copyIban}
            className="flex max-w-full items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-surface"
            title={t('mobile.copyIban')}
          >
            {copied ? <Check size={15} className="shrink-0 text-success" /> : <Copy size={15} className="shrink-0" />}
            <span className="truncate font-mono">{copied ? t('mobile.ibanCopied') : user.iban}</span>
          </button>
        )}
      </div>
    </Modal>
  )
}
