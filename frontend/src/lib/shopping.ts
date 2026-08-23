import type { ShoppingLine } from './types'

export type DaySegment = { key: string; qty: number }

// Breaks a line's total quantity into day-tagged slices (from dayQuantities)
// plus a catch-all "other" slice for whatever isn't tied to a specific day
// (tab-sourced quantities, or lines with no day info at all). Used to
// derive/toggle per-day bought status from the single boughtQuantity pool a
// line carries: segments are consumed in chronological order (FIFO), so
// checking a later day implicitly requires earlier days to be covered too,
// and unchecking a day also uncovers any later ones.
export function daySegments(line: ShoppingLine): DaySegment[] {
  const days = [...(line.days ?? [])].sort((a, b) => a - b)
  const dq = line.dayQuantities ?? {}
  const segs = days.map((d) => ({ key: String(d), qty: dq[String(d)] ?? 0 }))
  const sumDays = segs.reduce((s, x) => s + x.qty, 0)
  const other = Math.round((line.quantity - sumDays) * 100) / 100
  if (other > 0.0001) segs.push({ key: 'other', qty: other })
  return segs
}

function thresholds(line: ShoppingLine, key: string): { before: number; through: number } {
  let before = 0
  for (const s of daySegments(line)) {
    const through = before + s.qty
    if (s.key === key) return { before, through }
    before = through
  }
  return { before: line.quantity, through: line.quantity }
}

export function isDaySegmentBought(line: ShoppingLine, key: string): boolean {
  return line.boughtQuantity >= thresholds(line, key).through - 0.0001
}

// New boughtQuantity to PATCH when a single day's checkbox is toggled.
export function toggleDaySegment(line: ShoppingLine, key: string, checked: boolean): number {
  const { before, through } = thresholds(line, key)
  const v = checked ? Math.max(line.boughtQuantity, through) : Math.min(line.boughtQuantity, before)
  return Math.round(v * 100) / 100
}

// What's left to buy once partial purchases are deducted (used outside day
// mode, where a line is shown as a single row rather than per-day slices).
export function remainingQty(line: ShoppingLine): number {
  return Math.max(0, Math.round((line.quantity - line.boughtQuantity) * 100) / 100)
}
