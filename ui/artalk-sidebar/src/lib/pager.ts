/**
 * Shared pager vocabulary. Kept out of the component file so the module only
 * exports React components (react-refresh requirement).
 */
export const PAGE_SIZE_OPTIONS = [10, 20, 30, 50, 100]

/**
 * Page numbers with gaps, as `null` markers: `[1, null, 7, 8, 9, null, 42]`.
 * Small lists show every page; long ones keep the first, the last and a window
 * around the current page.
 */
export function pageWindow(page: number, pages: number): (number | null)[] {
  if (pages <= 7) return Array.from({ length: pages }, (_, index) => index + 1)

  const marks = new Set<number>([1, pages, page - 1, page, page + 1])
  if (page <= 4) for (const n of [2, 3, 4, 5]) marks.add(n)
  if (page >= pages - 3) for (const n of [pages - 4, pages - 3, pages - 2, pages - 1]) marks.add(n)

  const sorted = [...marks].filter((n) => n >= 1 && n <= pages).sort((a, b) => a - b)
  const out: (number | null)[] = []
  let previous = 0
  for (const n of sorted) {
    if (previous && n - previous > 1) out.push(null)
    out.push(n)
    previous = n
  }
  return out
}
