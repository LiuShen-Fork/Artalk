/**
 * The activity heatmap is a rectangle, never a shape with a bitten-off corner:
 * every column holds exactly seven days, the grid starts on a Sunday and ends
 * on the Saturday of the current week. Days that have not happened yet stay in
 * the grid as future cells, so the last column is as tall as all the others.
 */

export interface HeatDay {
  date: string
  count: number
}

export interface HeatCell {
  date: string
  count: number
  future: boolean
}

/** `2026-11-18` in local time — the format the API uses for `by_day`. */
export function isoDay(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

export function startOfToday(): Date {
  const date = new Date()
  date.setHours(0, 0, 0, 0)
  return date
}

/** Whole weeks ending with the week that contains `today`. */
export function buildWeeks(byDay: HeatDay[], weeks: number, today = startOfToday()): HeatCell[][] {
  const counts = new Map(byDay.map((item) => [item.date, item.count]))
  const first = new Date(today)
  first.setDate(first.getDate() - first.getDay() - (weeks - 1) * 7)

  const grid: HeatCell[][] = []
  const cursor = new Date(first)
  for (let week = 0; week < weeks; week += 1) {
    const cells: HeatCell[] = []
    for (let index = 0; index < 7; index += 1) {
      const date = isoDay(cursor)
      cells.push({ date, count: counts.get(date) ?? 0, future: cursor > today })
      cursor.setDate(cursor.getDate() + 1)
    }
    grid.push(cells)
  }
  return grid
}

/** Days the grid covers — a round number, since the weeks are whole. */
export function heatDayCount(weeks: number): number {
  return weeks * 7
}
