import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, FlaskConical, Rocket } from 'lucide-react'
import { api } from '../lib/api'
import Modal from './Modal'
import type { UpgradeResult, UpgradeStatus } from '../lib/types'

// Shown only to admins (the /upgrade endpoints are admin-only). A sticky
// banner persists as long as migrations are pending; the wizard modal opens
// automatically once per browser session so it isn't missed even if the
// admin only logs in occasionally, and can be postponed without losing the
// banner reminder.
const DISMISS_KEY = 'campmenu-upgrade-wizard-dismissed'

export default function UpgradeNotice() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<UpgradeStatus | null>(null)
  const [wizardOpen, setWizardOpen] = useState(false)
  const [dryRun, setDryRun] = useState<{ state: 'idle' | 'running' | 'ok' | 'error'; error?: string }>({ state: 'idle' })
  const [apply, setApply] = useState<{ state: 'idle' | 'running' | 'ok' | 'error'; error?: string }>({ state: 'idle' })

  async function load() {
    const s = await api.get<UpgradeStatus>('/upgrade')
    setStatus(s)
    return s
  }

  useEffect(() => {
    load().then((s) => {
      if (s.pending.length > 0 && sessionStorage.getItem(DISMISS_KEY) !== 'true') {
        setWizardOpen(true)
      }
    }).catch(() => {})
  }, [])

  const next = status?.pending[0]

  async function doDryRun() {
    if (!next) return
    setDryRun({ state: 'running' })
    setApply({ state: 'idle' })
    const res = await api.post<UpgradeResult>('/upgrade/dry-run', { version: next.version })
    setDryRun(res.ok ? { state: 'ok' } : { state: 'error', error: res.error })
  }

  async function doApply() {
    if (!next) return
    setApply({ state: 'running' })
    const res = await api.post<UpgradeResult>('/upgrade/apply', { version: next.version })
    if (res.ok) {
      setApply({ state: 'ok' })
      setDryRun({ state: 'idle' })
      await load()
    } else {
      setApply({ state: 'error', error: res.error })
    }
  }

  function dismiss() {
    sessionStorage.setItem(DISMISS_KEY, 'true')
    setWizardOpen(false)
  }

  if (!status || status.pending.length === 0) return null

  return (
    <>
      <div
        className="sticky top-0 z-30 flex items-center justify-center gap-3 bg-accent px-4 py-1.5 text-sm font-medium text-white"
        style={{ paddingTop: 'max(0.375rem, env(safe-area-inset-top))' }}
      >
        <AlertTriangle size={15} />
        <span>{t('admin.upgradeBanner', { count: status.pending.length })}</span>
        <button onClick={() => setWizardOpen(true)} className="rounded-md bg-white/20 px-2 py-0.5 hover:bg-white/30">
          {t('admin.upgradeBannerAction')}
        </button>
      </div>

      {wizardOpen && next && (
        <Modal title={t('admin.upgradeWizardTitle')} onClose={dismiss}>
          <div className="space-y-3 text-sm">
            <p>{t('admin.upgradeNextPending', { version: next.version, description: next.description })}</p>
            {next.appVersion && (
              <p className="rounded-lg border border-border bg-surface p-2 text-xs text-muted">
                {t('admin.upgradeRepoRef', { appVersion: next.appVersion })}
              </p>
            )}
            {status.pending.length > 1 && (
              <p className="text-xs text-muted">{t('admin.upgradeOtherPending', { count: status.pending.length - 1 })}</p>
            )}
            <div className="flex flex-wrap items-center gap-3">
              <button className="btn-ghost" onClick={doDryRun} disabled={dryRun.state === 'running'}>
                <FlaskConical size={15} /> {dryRun.state === 'running' ? t('admin.upgradeTesting') : t('admin.upgradeDryRun')}
              </button>
              <button className="btn-primary" onClick={doApply} disabled={dryRun.state !== 'ok' || apply.state === 'running'}>
                <Rocket size={15} /> {apply.state === 'running' ? t('admin.upgradeApplying') : t('admin.upgradeApply')}
              </button>
            </div>
            {dryRun.state === 'ok' && <p className="text-success">{t('admin.upgradeDryRunOk')}</p>}
            {dryRun.state === 'error' && <p className="text-danger">{t('admin.upgradeFailed', { error: dryRun.error })}</p>}
            {apply.state === 'ok' && <p className="text-success">{t('admin.upgradeApplyOk', { version: next.version })}</p>}
            {apply.state === 'error' && <p className="text-danger">{t('admin.upgradeFailed', { error: apply.error })}</p>}
            <div className="flex justify-end gap-2 pt-2">
              <button className="btn-ghost" onClick={dismiss}>{t('admin.upgradeLater')}</button>
            </div>
          </div>
        </Modal>
      )}
    </>
  )
}
