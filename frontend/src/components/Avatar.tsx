import { resolveAsset } from '../lib/api'
import { displayName } from '../lib/types'

type AvatarUser = {
  firstName?: string
  lastName?: string
  nickname?: string
  email?: string
  photoUrl?: string
} | null | undefined

// Avatar shows a user's round profile photo, falling back to their initials.
export default function Avatar({ user, size = 28 }: { user: AvatarUser; size?: number }) {
  const name = displayName(user)
  const initials = name
    .split(/\s+/)
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()
  const dim = { width: `${size}px`, height: `${size}px` }

  if (user?.photoUrl) {
    return (
      <img
        src={resolveAsset(user.photoUrl)}
        alt=""
        style={dim}
        className="shrink-0 rounded-full object-cover"
      />
    )
  }
  return (
    <span
      style={{ ...dim, fontSize: `${Math.round(size * 0.4)}px` }}
      className="flex shrink-0 items-center justify-center rounded-full bg-brand/15 font-semibold text-brand"
      aria-hidden
    >
      {initials || '?'}
    </span>
  )
}
