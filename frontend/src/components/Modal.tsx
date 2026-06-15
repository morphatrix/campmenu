import { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'

// Simple centered modal with a backdrop. Click outside or the X to close.
export default function Modal({
  title,
  onClose,
  children,
  wide,
}: {
  title?: string
  onClose: () => void
  children: ReactNode
  wide?: boolean
}) {
  // Scroll lives on the outer container; the inner flex uses min-h-full so the
  // modal is centered when short but, when taller than the viewport, grows and
  // stays fully scrollable from the very top (avoids the items-center clipping bug).
  // Render through a portal to <body>: a sticky/blurred header (backdrop-filter)
  // would otherwise become the containing block for this fixed overlay and clip
  // it to the header instead of the viewport.
  return createPortal(
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/50" onClick={onClose}>
      <div className="flex min-h-full items-center justify-center p-4">
        <div
          className={`card w-full ${wide ? 'max-w-2xl' : 'max-w-md'} p-6`}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="sticky -top-6 z-10 -mx-6 -mt-6 mb-4 flex items-center justify-between border-b border-border bg-card px-6 py-3">
            {title ? <h2 className="text-lg font-semibold">{title}</h2> : <span />}
            <button onClick={onClose} className="text-muted hover:text-fg" aria-label="close">
              <X size={20} />
            </button>
          </div>
          {children}
        </div>
      </div>
    </div>,
    document.body,
  )
}
