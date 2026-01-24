import { computed, ref, watch } from 'vue'

export type Theme = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'gosearch.theme'

const theme = ref<Theme>('system')
const loaded = ref(false)

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

const effectiveTheme = computed<'light' | 'dark'>(() => {
  if (theme.value === 'system') {
    return getSystemTheme()
  }
  return theme.value
})

function applyTheme(t: 'light' | 'dark') {
  const root = document.documentElement
  root.classList.remove('light', 'dark')
  root.classList.add(t)
}

function readStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored
    }
  } catch {
    // ignore storage errors
  }
  return 'system'
}

function persistTheme(t: Theme) {
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch {
    // ignore storage errors
  }
}

function setTheme(t: Theme) {
  theme.value = t
  persistTheme(t)
  applyTheme(effectiveTheme.value)
}

function initTheme() {
  if (loaded.value) return
  
  theme.value = readStoredTheme()
  applyTheme(effectiveTheme.value)
  
  // Listen for system theme changes
  if (typeof window !== 'undefined') {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', () => {
      if (theme.value === 'system') {
        applyTheme(effectiveTheme.value)
      }
    })
  }
  
  loaded.value = true
}

// Watch for theme changes and apply them
watch(effectiveTheme, (newTheme) => {
  applyTheme(newTheme)
})

export function useSettingsStore() {
  return {
    theme,
    effectiveTheme,
    setTheme,
    initTheme
  }
}
