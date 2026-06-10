import { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { api } from '../lib/api'
import type { User } from '../lib/types'

interface AuthState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  setUser: (u: User) => void
  refresh: () => Promise<void>
  impersonate: (id: string) => Promise<void>
  stopImpersonate: () => Promise<void>
}

const AuthContext = createContext<AuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  async function refresh() {
    try {
      const u = await api.get<User>('/me')
      setUser(u)
    } catch {
      setUser(null)
    }
  }

  useEffect(() => {
    refresh().finally(() => setLoading(false))
  }, [])

  async function login(email: string, password: string) {
    const res = await api.post<{ user: User }>('/auth/login', { email, password })
    setUser(res.user)
  }

  async function logout() {
    await api.post('/auth/logout')
    setUser(null)
  }

  async function impersonate(id: string) {
    const u = await api.post<User>(`/users/${id}/impersonate`)
    setUser(u)
  }

  async function stopImpersonate() {
    const u = await api.post<User>('/auth/stop-impersonate')
    setUser(u)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, setUser, refresh, impersonate, stopImpersonate }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
