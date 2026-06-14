import { ChangeEvent, useState } from 'react'
import { Upload } from 'lucide-react'
import { resolveAsset, uploadImage } from '../lib/api'
import ImageCropper from './ImageCropper'

// ImageUpload lets the user paste an external URL or upload a file (stored by the
// backend, which returns a /api/images/{id} URL). With `circle`, picking a file
// opens a circular crop/zoom step — handy for avatars and, since it re-encodes a
// small square JPEG, it also avoids "Failed to fetch" on huge phone photos.
export default function ImageUpload({
  value,
  onChange,
  circle,
}: {
  value: string
  onChange: (url: string) => void
  circle?: boolean
}) {
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
    if (circle) setCropFile(file)
    else doUpload(file)
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
      <div className="flex gap-2">
        <input className="input" placeholder="https://…" value={value} onChange={(e) => onChange(e.target.value)} />
        <label className="btn-ghost cursor-pointer whitespace-nowrap" title="Importer une image">
          <Upload size={15} /> {busy ? '…' : 'Upload'}
          <input type="file" accept="image/*" className="hidden" onChange={onFile} disabled={busy} />
        </label>
      </div>
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}
      {cropFile && (
        <ImageCropper
          file={cropFile}
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
