import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'

interface Suggestion {
  id: string
  name: string
  defaultUnit: string
  distance: number
}

// IngredientInput is a fuzzy autocomplete over the central ingredient
// referential to avoid duplicates (e.g. "beurre salé" -> "beurre demi-sel").
export default function IngredientInput({
  value,
  onChange,
  onPickUnit,
}: {
  value: string
  onChange: (v: string) => void
  onPickUnit?: (unit: string) => void
}) {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [open, setOpen] = useState(false)
  const timer = useRef<number>()

  useEffect(() => {
    window.clearTimeout(timer.current)
    if (!value.trim()) {
      setSuggestions([])
      return
    }
    timer.current = window.setTimeout(async () => {
      try {
        const res = await api.get<Suggestion[]>(`/ingredients/suggest?q=${encodeURIComponent(value)}`)
        setSuggestions(res.filter((s) => s.name.toLowerCase() !== value.trim().toLowerCase()))
      } catch {
        setSuggestions([])
      }
    }, 180)
    return () => window.clearTimeout(timer.current)
  }, [value])

  return (
    <div className="relative">
      <input
        className="input"
        value={value}
        onChange={(e) => { onChange(e.target.value); setOpen(true) }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        placeholder="ingrédient"
      />
      {open && suggestions.length > 0 && (
        <ul className="absolute z-30 mt-1 w-full overflow-hidden rounded-lg border border-border bg-card shadow-lg">
          {suggestions.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                className="flex w-full items-center justify-between px-3 py-1.5 text-left text-sm hover:bg-surface"
                onMouseDown={() => {
                  onChange(s.name)
                  if (s.defaultUnit && onPickUnit) onPickUnit(s.defaultUnit)
                  setOpen(false)
                }}
              >
                <span>{s.name}</span>
                <span className="text-xs text-muted">{s.defaultUnit}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
