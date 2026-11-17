import { useSyncExternalStore } from 'react'

/**
 * Match a media query. Used where the layout has to change structurally rather
 * than just visually — a Drawer's edge, for example, cannot be switched by a
 * CSS class.
 */
export function useMediaQuery(query: string) {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query)
      mql.addEventListener('change', onChange)
      return () => mql.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}
