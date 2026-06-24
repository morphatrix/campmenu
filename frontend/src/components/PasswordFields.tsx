import { useTranslation } from 'react-i18next'

// isStrongPassword enforces: ≥8 chars, a lowercase, an uppercase, a digit and a
// special character. Mirrors the server-side check.
export function isStrongPassword(pw: string): boolean {
  return pw.length >= 8 && /[a-z]/.test(pw) && /[A-Z]/.test(pw) && /\d/.test(pw) && /[^A-Za-z0-9]/.test(pw)
}

// PasswordFields renders two masked inputs (password + confirmation) with the
// complexity hint and a mismatch warning. The parent owns the state and gates
// submission with isStrongPassword(password) && password === confirm.
export default function PasswordFields({
  password,
  confirm,
  setPassword,
  setConfirm,
}: {
  password: string
  confirm: string
  setPassword: (v: string) => void
  setConfirm: (v: string) => void
}) {
  const { t } = useTranslation()
  const weak = password.length > 0 && !isStrongPassword(password)
  const mismatch = confirm.length > 0 && confirm !== password
  return (
    <>
      <div>
        <label className="label">{t('auth.password')}</label>
        <input className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
      </div>
      <div>
        <label className="label">{t('auth.confirmPassword')}</label>
        <input className="input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
      </div>
      <p className={`text-xs ${weak ? 'text-danger' : 'text-muted'}`}>{t('auth.passwordRules')}</p>
      {mismatch && <p className="text-xs text-danger">{t('auth.passwordMismatch')}</p>}
    </>
  )
}
