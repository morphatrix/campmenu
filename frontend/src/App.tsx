import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from './context/AuthContext'
import { ActiveEventProvider } from './context/ActiveEventContext'
import { applyAppearance } from './lib/appearance'
import Layout from './components/Layout'
import LoginPage from './pages/LoginPage'
import InvitePage from './pages/InvitePage'
import ConfirmPage from './pages/ConfirmPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import MobileShoppingPage from './pages/MobileShoppingPage'
import InstallPage from './pages/InstallPage'
import EventsPage from './pages/EventsPage'
import EventDetailPage from './pages/EventDetailPage'
import RecipesPage from './pages/RecipesPage'
import ListsPage from './pages/ListsPage'
import ProfilePage from './pages/ProfilePage'
import AdminPage from './pages/AdminPage'

function Protected({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="grid h-screen place-items-center text-muted">…</div>
  if (!user) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  const { user } = useAuth()
  const { i18n } = useTranslation()

  // Reflect the signed-in user's appearance + language preferences.
  useEffect(() => {
    applyAppearance(user)
    if (user?.language && user.language !== i18n.language) {
      i18n.changeLanguage(user.language)
    }
  }, [user, i18n])

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/forgot" element={<ForgotPasswordPage />} />
      <Route path="/reset/:token" element={<ResetPasswordPage />} />
      <Route path="/install" element={<InstallPage />} />
      <Route path="/m" element={<MobileShoppingPage />} />
      <Route path="/invite/:code" element={<InvitePage />} />
      <Route path="/confirm/:token" element={<ConfirmPage />} />
      <Route
        element={
          <Protected>
            <ActiveEventProvider>
              <Layout />
            </ActiveEventProvider>
          </Protected>
        }
      >
        <Route path="/" element={<EventsPage />} />
        <Route path="/events/:id" element={<EventDetailPage />} />
        <Route path="/recipes" element={<RecipesPage />} />
        <Route path="/cocktails" element={<RecipesPage cocktails />} />
        <Route path="/lists" element={<ListsPage />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/admin" element={<AdminPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
