import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Apple, Smartphone, ArrowLeft } from 'lucide-react'

export default function InstallPage() {
  const { t } = useTranslation()
  const appUrl = `${window.location.origin}/m`
  const qr = `https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=8&data=${encodeURIComponent(appUrl)}`

  return (
    <div className="mx-auto max-w-lg px-4 py-8">
      <Link to="/login" className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-fg"><ArrowLeft size={15} /> CampMenu</Link>
      <h1 className="text-2xl font-bold">{t('install.title')}</h1>
      <p className="mt-1 text-sm text-muted">{t('install.subtitle')}</p>

      <div className="my-6 flex flex-col items-center gap-2">
        <img src={qr} alt="QR" className="rounded-xl border border-border bg-white p-1" width={220} height={220} />
        <p className="text-xs text-muted">{t('install.scan')}</p>
        <a href={appUrl} className="text-sm text-brand hover:underline">{appUrl}</a>
      </div>

      <div className="space-y-4">
        <section className="card p-5">
          <h2 className="mb-2 flex items-center gap-2 font-semibold"><Apple size={18} /> {t('install.ios')}</h2>
          <p className="text-sm text-muted">{t('install.iosSteps')}</p>
        </section>
        <section className="card p-5">
          <h2 className="mb-2 flex items-center gap-2 font-semibold"><Smartphone size={18} /> {t('install.android')}</h2>
          <p className="text-sm text-muted">{t('install.androidSteps')}</p>
        </section>
      </div>
    </div>
  )
}
