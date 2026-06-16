import { createContext, ReactNode, useCallback, useContext, useState } from 'react'

// The "active event" is the one the user is currently working in. It persists
// (localStorage) so navigating to global sections (Recipes, Lists…) and back
// keeps the context; clearing it returns to the events list.
export type ActiveEvent = { id: string; name: string } | null

const STORAGE_KEY = 'campmenu.activeEvent'

const ActiveEventContext = createContext<{
  active: ActiveEvent
  setActive: (e: ActiveEvent) => void
}>({ active: null, setActive: () => {} })

export function ActiveEventProvider({ children }: { children: ReactNode }) {
  const [active, setActiveState] = useState<ActiveEvent>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw ? (JSON.parse(raw) as ActiveEvent) : null
    } catch {
      return null
    }
  })

  const setActive = useCallback((e: ActiveEvent) => {
    setActiveState(e)
    try {
      if (e) localStorage.setItem(STORAGE_KEY, JSON.stringify(e))
      else localStorage.removeItem(STORAGE_KEY)
    } catch { /* storage may be unavailable */ }
  }, [])

  return <ActiveEventContext.Provider value={{ active, setActive }}>{children}</ActiveEventContext.Provider>
}

export function useActiveEvent() {
  return useContext(ActiveEventContext)
}
