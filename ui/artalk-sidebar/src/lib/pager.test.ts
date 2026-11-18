import { describe, expect, it } from 'vitest'
import { PAGE_SIZE_OPTIONS, pageWindow } from './pager'

describe('pageWindow', () => {
  it('lists every page while the list is short', () => {
    expect(pageWindow(1, 1)).toEqual([1])
    expect(pageWindow(3, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
  })

  it('keeps the first and last page with a window around the current one', () => {
    expect(pageWindow(10, 42)).toEqual([1, null, 9, 10, 11, null, 42])
  })

  it('fills the leading run near the start', () => {
    expect(pageWindow(2, 42)).toEqual([1, 2, 3, 4, 5, null, 42])
  })

  it('fills the trailing run near the end', () => {
    expect(pageWindow(41, 42)).toEqual([1, null, 38, 39, 40, 41, 42])
  })

  it('collapses everything between the first page and the tail run', () => {
    expect(pageWindow(20, 21)).toEqual([1, null, 17, 18, 19, 20, 21])
  })
})

describe('PAGE_SIZE_OPTIONS', () => {
  it('is ascending and starts small enough for a phone', () => {
    expect(PAGE_SIZE_OPTIONS[0]).toBeLessThanOrEqual(10)
    expect([...PAGE_SIZE_OPTIONS].sort((a, b) => a - b)).toEqual(PAGE_SIZE_OPTIONS)
  })
})
