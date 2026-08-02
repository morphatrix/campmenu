import { ChangeEvent, ClipboardEvent, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Upload } from 'lucide-react'
import { resolveAsset, uploadImage } from '../lib/api'
import ImageCropper from './ImageCropper'

// ImageUpload lets the user paste an external URL, upload a file, or paste an
// image straight from the clipboard (Ctrl+V after clicking the drop zone).
// Any picked/pasted image goes through a crop/zoom step first — circular for
// avatars (`circle`), rectangular otherwise — which also re-encodes a small
// JPEG so even huge phone photos upload comfortably.
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

  return (
    <div>
      {value && (
        <img
          src={resolveAsset(value)}
          alt=""
          className={circle ? 'mb-2 h-24 w-24 rounded-full object-cover' : 'mb-2 max-h-40 w-full rounded-lg object-cover'}
        />
      )}
      <div
        className="flex gap-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand"
        tabIndex={0}
        onPaste={onPaste}
        title={t('profile.pasteHint')}
      >
        <input className="input" placeholder="https://…" value={value} onChange={(e) => onChange(e.target.value)} />
        <label className="btn-ghost cursor-pointer whitespace-nowrap" title={t('profile.pasteHint')}>
          <Upload size={15} /> {busy ? '…' : 'Upload'}
          <input type="file" accept="image/*" className="hidden" onChange={onFile} disabled={busy} />
        </label>
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
