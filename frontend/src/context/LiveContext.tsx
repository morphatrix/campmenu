import { createContext, useCallback, useContext, useEffect, useRef, ReactNode } from 'react'
import { openStream } from '../lib/api'
import { useAuth } from './AuthContext'

type Listener = () => void

const LiveContext = createContext<{ subscribe: (fn: Listener) => () => void }>({
  subscribe: () => () => {},
})

const TYPING_TAGS = ['INPUT', 'TEXTAREA', 'SELECT']
function isTyping(): boolean {
  const el = document.activeElement
  return !!el && TYPING_TAGS.includes(el.tagName)
}

// LiveProvider keeps a single shared SSE connection to /api/stream and notifies
// every registered listener (a view's load function) on each change broadcast,
// so updates are instant. Refreshes are deferred while a field is focused so an
// in-progress edit is never clobbered. Window focus + a slow interval act as a
// safety net if the SSE connection is ever interrupted.
export function LiveProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const listeners = useRef<Set<Listener>>(new Set())
  const pending = useRef(false)

  const dispatch = useCallback(() => {
    listeners.current.forEach((fn) => fn())
  }, [])

  const refreshIfIdle = useCallback(() => {
    if (isTyping()) {
      pending.current = true
      return
    }
    pending.current = false
    dispatch()
  }, [dispatch])

  const subscribe = useCallback((fn: Listener) => {
    listeners.current.add(fn)
    return () => {
      listeners.current.delete(fn)
    }
  }, [])

  useEffect(() => {
    if (!user) return
    const es = openStream('/api/stream')
    es.onmessage = () => refreshIfIdle()

    // Flush a deferred refresh shortly after the user stops typing.
    const flush = window.setInterval(() => {
      if (pending.current && !isTyping()) {
        pending.current = false
        dispatch()
      }
    }, 1500)
    // Safety net: refetch on focus and every 30s in case SSE dropped silently.
    const fallback = window.setInterval(() => { if (!isTyping()) dispatch() }, 30000)
    const onFocus = () => { if (!isTyping()) dispatch() }
    window.addEventListener('focus', onFocus)

    return () => {
      es.close()
      window.clearInterval(flush)
      window.clearInterval(fallback)
      window.removeEventListener('focus', onFocus)
    }
  }, [user, refreshIfIdle, dispatch])

  return <LiveContext.Provider value={{ subscribe }}>{children}</LiveContext.Provider>
}

// useLive registers a refetch callback fired whenever live data changes.
export function useLive(fn: Listener) {
  const { subscribe } = useContext(LiveContext)
  const saved = useRef(fn)
  saved.current = fn
  useEffect(() => subscribe(() => saved.current()), [subscribe])
}
