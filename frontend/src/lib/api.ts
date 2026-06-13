// Resolve the API base URL at runtime (k8s ConfigMap injects index.html) and
// fall back to the Vite build-time variable for local dev.
function apiBase(): string {
  const injected = document.querySelector('meta[name="campmenu-api"]')?.getAttribute('content') ?? undefined
  if (injected && injected !== '__API_URL__') return injected.replace(/\/$/, '')
  const env = import.meta.env.VITE_API_URL as string | undefined
  return (env ?? '').replace(/\/$/, '')
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${apiBase()}/api${path}`, {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  // Tolerate non-JSON bodies (e.g. an HTML error page injected by a proxy on a
  // 5xx): never let JSON.parse throw a cryptic "Unexpected token '<'".
  let data: any = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = null
  }
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? `Erreur ${res.status}`)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: <T>(path: string) => request<T>('DELETE', path),
}

// resolveAsset turns a stored photo value into a usable <img> src. External
// URLs pass through; relative ones (e.g. /api/images/{id}) are prefixed with
// the API origin so they work both same-origin and cross-origin (Compose).
export function resolveAsset(url?: string | null): string {
  if (!url) return ''
  if (/^https?:\/\//i.test(url) || url.startsWith('data:')) return url
  return apiBase() + url
}

// openStream opens the Server-Sent Events connection for live updates.
export function openStream(path: string): EventSource {
  return new EventSource(`${apiBase()}${path}`, { withCredentials: true })
}

// uploadImage POSTs a file to /api/images and returns its relative URL.
export async function uploadImage(file: File): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  const res = await fetch(`${apiBase()}/api/images`, {
    method: 'POST',
    credentials: 'include',
    body: fd,
  })
  const data = await res.json().catch(() => null)
  if (!res.ok) throw new ApiError(res.status, data?.error ?? 'upload impossible')
  return data.url as string
}
