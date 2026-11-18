import { useCallback, useState } from 'react'

const PREFIX = 'ATK_SIDEBAR_PAGE_SIZE_'

function read(key: string, fallback: number, options: number[]): number {
  try {
    const raw = Number(localStorage.getItem(PREFIX + key))
    return options.includes(raw) ? raw : fallback
  } catch {
    return fallback
  }
}

/**
 * How many rows a list shows, remembered per list so the reader sets it once.
 * Kept on the device rather than the server: it is a reading preference, not a
 * site-wide policy.
 */
export function usePageSize(
  key: string,
  fallback: number,
  options: number[] = [10, 20, 30, 50, 100],
): [number, (next: number) => void] {
  const [size, setSize] = useState(() => read(key, fallback, options))

  const update = useCallback(
    (next: number) => {
      setSize(next)
      try {
        localStorage.setItem(PREFIX + key, String(next))
      } catch {
        // Private mode without storage: the choice just does not persist.
      }
    },
    [key],
  )

  return [size, update]
}
