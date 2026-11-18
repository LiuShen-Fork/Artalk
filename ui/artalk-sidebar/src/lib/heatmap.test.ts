import { describe, expect, it } from 'vitest'
import { buildWeeks, heatDayCount, isoDay } from './heatmap'

// 2026-11-18 is a Wednesday.
const TODAY = new Date(2026, 10, 18)

describe('buildWeeks', () => {
  it('draws a complete rectangle of whole weeks', () => {
    const weeks = buildWeeks([], 26, TODAY)

    expect(weeks).toHaveLength(26)
    for (const week of weeks) expect(week).toHaveLength(7)
  })

  it('starts on a Sunday and ends on the Saturday of the current week', () => {
    const weeks = buildWeeks([], 26, TODAY)
    const first = weeks[0][0].date
    const last = weeks[weeks.length - 1][6].date

    expect(new Date(`${first}T00:00:00`).getDay()).toBe(0)
    expect(new Date(`${last}T00:00:00`).getDay()).toBe(6)
    // Six future days in the last column: today is Wednesday.
    expect(weeks[weeks.length - 1].filter((cell) => cell.future)).toHaveLength(3)
  })

  it('marks today as real and the days after it as future', () => {
    const weeks = buildWeeks([], 26, TODAY)
    const cells = weeks.flat()
    const today = cells.find((cell) => cell.date === isoDay(TODAY))
    const tomorrow = cells.find((cell) => cell.date === '2026-11-19')

    expect(today?.future).toBe(false)
    expect(tomorrow?.future).toBe(true)
  })

  it('carries the counts from the API onto the right days', () => {
    const weeks = buildWeeks(
      [
        { date: '2026-11-18', count: 4 },
        { date: '2026-05-24', count: 2 },
      ],
      26,
      TODAY,
    )
    const cells = weeks.flat()

    expect(cells.find((cell) => cell.date === '2026-11-18')?.count).toBe(4)
    expect(cells.find((cell) => cell.date === weeks[0][0].date)?.count).toBe(2)
    // Anything the API left out is a quiet day, not a hole.
    expect(cells.find((cell) => cell.date === '2026-11-17')?.count).toBe(0)
  })
})

describe('heatDayCount', () => {
  it('reports a round number of days', () => {
    expect(heatDayCount(26)).toBe(182)
  })
})
