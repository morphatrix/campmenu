import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'

// UnitInput autocompletes an ingredient's unit: units already used for this
// exact ingredient first, falling back to every unit ever used anywhere only
// when none of the ingredient-specific ones match what's being typed.
export default function UnitInput({
  value,
  onChange,
  ingredientName,
  className,
}: {
  value: string
  onChange: (v: string) => void
  ingredientName: string
  className?: string
}) {
  const [forIngredient, setForIngredient] = useState<string[]>([])
  const [allKnown, setAllKnown] = useState<string[]>([])
  const [open, setOpen] = useState(false)
  const timer = useRef<number>()

  useEffect(() => {
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(async () => {
      try {
        const res = await api.get<{ forIngredient: string[]; allKnown: string[] }>(
          `/ingredients/units?name=${encodeURIComponent(ingredientName.trim())}`,
        )
        setForIngredient(res.forIngredient ?? [])
        setAllKnown(res.allKnown ?? [])
      } catch {
        setForIngredient([])
        setAllKnown([])
      }
    }, 200)
    return () => window.clearTimeout(timer.current)
  }, [ingredientName])

  const q = value.trim().toLowerCase()
  const specific = forIngredient.filter((u) => q === '' || u.toLowerCase().includes(q))
  const list = (specific.length > 0 ? specific : allKnown.filter((u) => q === '' || u.toLowerCase().includes(q)))
    .filter((u) => u.toLowerCase() !== q)
    .slice(0, 8)

  return (
    <div className="relative">
      <input
        className={className ?? 'input'}
        placeholder="unité"
        value={value}
        onChange={(e) => { onChange(e.target.value); setOpen(true) }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
      />
      {open && list.length > 0 && (
        <ul className="absolute z-30 mt-1 w-full overflow-hidden rounded-lg border border-border bg-card shadow-lg">
          {list.map((u) => (
            <li key={u}>
              <button
                type="button"
                className="block w-full px-3 py-1.5 text-left text-sm hover:bg-surface"
                onMouseDown={() => { onChange(u); setOpen(false) }}
              >
                {u}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
