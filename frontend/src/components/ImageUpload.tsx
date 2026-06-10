import { ChangeEvent, useState } from 'react'
import { Upload } from 'lucide-react'
import { resolveAsset, uploadImage } from '../lib/api'

// ImageUpload lets the user either paste an external URL or upload a file
// (stored by the backend, which returns a /api/images/{id} URL).
export default function ImageUpload({
  value,
  onChange,
}: {
  value: string
  onChange: (url: string) => void
}) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function onFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setBusy(true)
    setError('')
    try {
      const url = await uploadImage(file)
      onChange(url)
    } catch (err: any) {
      setError(err?.message ?? 'upload impossible')
    } finally {
      setBusy(false)
      e.target.value = ''
    }
  }

  return (
    <div>
      {value && (
        <img src={resolveAsset(value)} alt="" className="mb-2 max-h-40 w-full rounded-lg object-cover" />
      )}
      <div className="flex gap-2">
        <input
          className="input"
          placeholder="https://…"
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
        <label className="btn-ghost cursor-pointer whitespace-nowrap" title="Importer une image">
          <Upload size={15} /> {busy ? '…' : 'Upload'}
          <input type="file" accept="image/*" className="hidden" onChange={onFile} disabled={busy} />
        </label>
      </div>
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}
    </div>
  )
}
