const rawBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api'
let base = rawBase.trim()
if (base === '/' || base === '') {
  base = ''
} else if (base.endsWith('/')) {
  base = base.slice(0, -1)
}

export function apiUrl(path: string) {
  if (!path) return base
  if (/^https?:\/\//i.test(path)) return path
  const suffix = path.startsWith('/') ? path : `/${path}`
  if (!base) return suffix
  return `${base}${suffix}`
}

export function apiFetch(path: string, init?: RequestInit) {
  return fetch(apiUrl(path), init)
}
