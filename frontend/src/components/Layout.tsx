import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarDays, BookOpen, Martini, ListChecks, User as UserIcon, Shield, LogOut, Tent } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { displayName, isAdmin, isStaff } from '../lib/types'

export default function Layout() {
  const { t } = useTranslation()
  const { user, logout, stopImpersonate } = useAuth()
  const navigate = useNavigate()

  async function exitImpersonation() {
    await stopImpersonate()
    navigate('/')
  }

  const items = [
    { to: '/', label: t('nav.events'), icon: CalendarDays, end: true },
    { to: '/recipes', label: t('nav.recipes'), icon: BookOpen },
    { to: '/cocktails', label: t('nav.cocktails'), icon: Martini },
    { to: '/lists', label: t('nav.lists'), icon: ListChecks },
    { to: '/profile', label: t('nav.profile'), icon: UserIcon },
  ]
  if (isStaff(user)) items.push({ to: '/admin', label: isAdmin(user) ? t('nav.admin') : t('nav.manage'), icon: Shield })

  async function handleLogout() {
    await logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen">
      {user?.impersonating && (
        <div className="sticky top-0 z-30 flex items-center justify-center gap-3 bg-accent px-4 py-1.5 text-sm font-medium text-white">
          <span>{t('impersonate.banner', { name: displayName(user) })}</span>
          <button onClick={exitImpersonation} className="rounded-md bg-white/20 px-2 py-0.5 hover:bg-white/30">
            {t('impersonate.exit')}
          </button>
        </div>
      )}
      <header className="sticky top-0 z-20 border-b border-border bg-card/80 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2 font-semibold text-brand">
            <Tent size={22} /> CampMenu
            {user && <span className="chip ml-1 hidden font-normal text-muted sm:inline-flex">{t(`roles.${user.role}`)}</span>}
          </div>
          <nav className="flex items-center gap-1">
            {items.map((it) => (
              <NavLink
                key={it.to}
                to={it.to}
                end={it.end}
                className={({ isActive }) =>
                  `flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                    isActive ? 'bg-brand text-brand-fg' : 'text-muted hover:bg-surface hover:text-fg'
                  }`
                }
              >
                <it.icon size={16} />
                <span className="hidden sm:inline">{it.label}</span>
              </NavLink>
            ))}
            <button onClick={handleLogout} className="ml-1 rounded-lg px-3 py-1.5 text-sm text-muted hover:text-danger" title={t('nav.logout')}>
              <LogOut size={16} />
            </button>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
