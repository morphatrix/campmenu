import { ChangeEvent, ClipboardEvent, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Crop, Upload } from 'lucide-react'
import { resolveAsset, uploadImage } from '../lib/api'
import ImageCropper from './ImageCropper'

// ImageUpload lets the user paste an external URL, upload a file, or paste an
// image straight from the clipboard (Ctrl+V after clicking the drop zone).
// Any picked/pasted image goes through a crop/zoom step first — circular for
// avatars (`circle`), rectangular otherwise, matching the aspect ratio the
// photo is actually displayed at elsewhere in the app — which also re-encodes
// a small JPEG so even huge phone photos upload comfortably. Clicking an
// already-set image reopens the cropper on it.
export default function ImageUpload({
  value,
  onChange,
  circle,
}: {
  value: string
  onChange: (url: string) => void
  circle?: boolean
}) {
  const { t } = useTranslation()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [cropFile, setCropFile] = useState<File | null>(null)

  async function doUpload(blob: Blob) {
    setBusy(true)
    setError('')
    try {
      const file = blob instanceof File ? blob : new File([blob], 'image.jpg', { type: blob.type || 'image/jpeg' })
      onChange(await uploadImage(file))
    } catch (err: any) {
      setError(err?.message ?? 'upload impossible')
    } finally {
      setBusy(false)
    }
  }

  function onFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setCropFile(file)
  }

  function onPaste(e: ClipboardEvent<HTMLDivElement>) {
    const item = Array.from(e.clipboardData?.items ?? []).find((it) => it.type.startsWith('image/'))
    if (!item) return
    e.preventDefault()
    const file = item.getAsFile()
    if (file) setCropFile(file)
  }

  async function recrop() {
    if (!value) return
    setError('')
    try {
      const res = await fetch(resolveAsset(value))
      const blob = await res.blob()
      setCropFile(new File([blob], 'image.jpg', { type: blob.type || 'image/jpeg' }))
    } catch {
      setError(t('profile.recropFailed'))
    }
  }

  return (
    <div>
      {value && (
        <button type="button" onClick={recrop} className="group relative mb-2 block w-fit" title={t('profile.recrop')}>
          <img
            src={resolveAsset(value)}
            alt=""
            className={circle ? 'h-24 w-24 rounded-full object-cover' : 'aspect-[4/3] w-56 rounded-lg object-cover'}
          />
          <span className="absolute inset-0 hidden items-center justify-center gap-1 rounded-lg bg-black/50 text-xs font-medium text-white group-hover:flex">
            <Crop size={14} /> {t('profile.recrop')}
          </span>
        </button>
      )}
      <div
        className="rounded-lg focus:outline-none focus:ring-2 focus:ring-brand"
        tabIndex={0}
        onPaste={onPaste}
      >
        <div className="flex gap-2">
          <input className="input" placeholder="https://…" value={value} onChange={(e) => onChange(e.target.value)} />
          <label className="btn-ghost cursor-pointer whitespace-nowrap">
            <Upload size={15} /> {busy ? '…' : 'Upload'}
            <input type="file" accept="image/*" className="hidden" onChange={onFile} disabled={busy} />
          </label>
        </div>
        <p className="mt-1 text-xs text-muted">{t('profile.pasteHint')}</p>
      </div>
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}
      {cropFile && (
        <ImageCropper
          file={cropFile}
          shape={circle ? 'circle' : 'rect'}
          onCancel={() => setCropFile(null)}
          onCropped={(blob) => {
            setCropFile(null)
            doUpload(blob)
          }}
        />
      )}
    </div>
  )
}
