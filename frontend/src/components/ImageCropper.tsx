import { PointerEvent as ReactPointerEvent, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Modal from './Modal'

const VIEW = 288 // on-screen circular viewport (px)
const OUTPUT = 512 // exported square image size (px)

type Offset = { x: number; y: number }

// ImageCropper lets the user zoom/pan an image inside a circular viewport, then
// exports a small square JPEG (so even huge phone photos upload comfortably).
export default function ImageCropper({
  file,
  onCancel,
  onCropped,
}: {
  file: File
  onCancel: () => void
  onCropped: (blob: Blob) => void
}) {
  const { t } = useTranslation()
  const [img, setImg] = useState<HTMLImageElement | null>(null)
  const [scale, setScale] = useState(1)
  const [minScale, setMinScale] = useState(1)
  const [offset, setOffset] = useState<Offset>({ x: 0, y: 0 })
  const drag = useRef<{ x: number; y: number; ox: number; oy: number } | null>(null)

  useEffect(() => {
    // Read as a data: URL (allowed by the site CSP, unlike blob:) so both the
    // preview <img> and the canvas source load.
    const reader = new FileReader()
    reader.onload = () => {
      const im = new Image()
      im.onload = () => {
        const cover = Math.max(VIEW / im.naturalWidth, VIEW / im.naturalHeight)
        setImg(im)
        setScale(cover)
        setMinScale(cover)
        setOffset({ x: (VIEW - im.naturalWidth * cover) / 2, y: (VIEW - im.naturalHeight * cover) / 2 })
      }
      im.src = reader.result as string
    }
    reader.readAsDataURL(file)
  }, [file])

  function clamp(o: Offset, s: number): Offset {
    if (!img) return o
    const w = img.naturalWidth * s
    const h = img.naturalHeight * s
    return {
      x: Math.min(0, Math.max(VIEW - w, o.x)),
      y: Math.min(0, Math.max(VIEW - h, o.y)),
    }
  }

  function onPointerDown(e: ReactPointerEvent<HTMLDivElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { x: e.clientX, y: e.clientY, ox: offset.x, oy: offset.y }
  }
  function onPointerMove(e: ReactPointerEvent<HTMLDivElement>) {
    if (!drag.current) return
    setOffset(
      clamp({ x: drag.current.ox + (e.clientX - drag.current.x), y: drag.current.oy + (e.clientY - drag.current.y) }, scale),
    )
  }
  function onPointerUp() {
    drag.current = null
  }

  function onZoom(s: number) {
    // Keep the viewport centre fixed while zooming.
    const c = VIEW / 2
    const k = s / scale
    setOffset(clamp({ x: c - (c - offset.x) * k, y: c - (c - offset.y) * k }, s))
    setScale(s)
  }

  function confirm() {
    if (!img) return
    const canvas = document.createElement('canvas')
    canvas.width = OUTPUT
    canvas.height = OUTPUT
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const ratio = OUTPUT / VIEW
    ctx.drawImage(
      img,
      offset.x * ratio,
      offset.y * ratio,
      img.naturalWidth * scale * ratio,
      img.naturalHeight * scale * ratio,
    )
    canvas.toBlob((blob) => blob && onCropped(blob), 'image/jpeg', 0.9)
  }

  return (
    <Modal title={t('profile.cropTitle')} onClose={onCancel}>
      <div className="flex flex-col items-center gap-4">
        <div
          className="relative touch-none select-none overflow-hidden rounded-full bg-surface"
          style={{ width: VIEW, height: VIEW }}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
        >
          {img && (
            <img
              src={img.src}
              alt=""
              draggable={false}
              style={{
                position: 'absolute',
                left: 0,
                top: 0,
                width: img.naturalWidth * scale,
                height: img.naturalHeight * scale,
                maxWidth: 'none',
                transform: `translate(${offset.x}px, ${offset.y}px)`,
              }}
            />
          )}
          <div className="pointer-events-none absolute inset-0 rounded-full ring-2 ring-white/70" />
        </div>
        <input
          type="range"
          min={minScale}
          max={minScale * 4}
          step={0.01}
          value={scale}
          onChange={(e) => onZoom(+e.target.value)}
          className="w-full"
          aria-label={t('profile.cropZoom')}
        />
        <div className="flex justify-end gap-2 self-stretch">
          <button className="btn-ghost" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={confirm}>{t('common.save')}</button>
        </div>
      </div>
    </Modal>
  )
}
