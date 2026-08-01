import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, FlaskConical, Rocket, XCircle } from 'lucide-react'
import { api } from '../../lib/api'
import type { UpgradeResult, UpgradeStatus } from '../../lib/types'

export default function UpgradeSection() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<UpgradeStatus | null>(null)
  const [dryRun, setDryRun] = useState<{ state: 'idle' | 'running' | 'ok' | 'error'; error?: string }>({ state: 'idle' })
  const [apply, setApply] = useState<{ state: 'idle' | 'running' | 'ok' | 'error'; error?: string }>({ state: 'idle' })

  async function load() {
    setStatus(await api.get<UpgradeStatus>('/upgrade'))
  }
  useEffect(() => { load() }, [])

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

  if (!status) return null

  return (
    <section className="card space-y-4 p-6">
      <p className="text-sm text-muted">{t('admin.upgradeCurrentVersion', { version: status.currentVersion })}</p>

      {!next ? (
        <p className="text-sm text-success">{t('admin.upgradeUpToDate')}</p>
      ) : (
        <div className="space-y-3">
          <p className="text-sm">{t('admin.upgradeNextPending', { version: next.version, description: next.description })}</p>
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
          {dryRun.state === 'ok' && (
            <p className="flex items-center gap-1 text-sm text-success"><CheckCircle2 size={15} /> {t('admin.upgradeDryRunOk')}</p>
          )}
          {dryRun.state === 'error' && (
            <p className="flex items-center gap-1 text-sm text-danger"><XCircle size={15} /> {t('admin.upgradeFailed', { error: dryRun.error })}</p>
          )}
          {apply.state === 'ok' && (
            <p className="flex items-center gap-1 text-sm text-success"><CheckCircle2 size={15} /> {t('admin.upgradeApplyOk', { version: next.version })}</p>
          )}
          {apply.state === 'error' && (
            <p className="flex items-center gap-1 text-sm text-danger"><XCircle size={15} /> {t('admin.upgradeFailed', { error: apply.error })}</p>
          )}
        </div>
      )}
    </section>
  )
}
