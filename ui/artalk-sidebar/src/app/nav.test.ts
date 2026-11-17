import { describe, expect, it } from 'vitest'
import {
  defaultPathFor,
  findNavItem,
  isPathAllowed,
  labelKeyFor,
  navItemsFor,
  primaryItemsFor,
  secondaryItemsFor,
} from './nav'

const ids = (items: { id: string }[]) => items.map((item) => item.id)

describe('navigation by role and view', () => {
  it('gives a visitor only their own pages', () => {
    expect(ids(navItemsFor(false, 'user'))).toEqual(['comments', 'account'])
    // A visitor cannot be moved into the admin view, but the helper must not
    // hand out management pages even if it is asked to.
    expect(ids(navItemsFor(false, 'admin'))).toEqual(['comments', 'account'])
  })

  it('gives an admin in visitor view the same two pages', () => {
    expect(ids(navItemsFor(true, 'user'))).toEqual(['comments', 'account'])
    expect(ids(primaryItemsFor(true, 'user'))).toEqual(['comments', 'account'])
    expect(secondaryItemsFor(true, 'user')).toEqual([])
  })

  it('gives an admin in admin view every page, split into tabs and sheet', () => {
    const all = ids(navItemsFor(true, 'admin'))
    expect(all).toContain('dashboard')
    expect(all).toContain('security')
    expect(ids(primaryItemsFor(true, 'admin'))).toEqual(['dashboard', 'comments', 'moderation', 'users'])
    expect(ids(secondaryItemsFor(true, 'admin'))).not.toContain('dashboard')
  })

  it('lands each role on a page it may see', () => {
    expect(defaultPathFor(true, 'admin')).toBe('/dashboard')
    expect(defaultPathFor(true, 'user')).toBe('/comments')
    expect(defaultPathFor(false)).toBe('/comments')
    // Both default destinations are reachable from the menu of that role.
    expect(ids(navItemsFor(true, 'admin'))).toContain('dashboard')
    expect(ids(navItemsFor(false, 'user'))).toContain('comments')
  })

  it('renames shared routes in visitor view', () => {
    const comments = findNavItem('comments')
    expect(comments).toBeDefined()
    expect(labelKeyFor(comments!, 'user')).toBe('myComments')
    expect(labelKeyFor(comments!, 'admin')).toBe('comments')
  })
})

describe('path guard', () => {
  it('keeps management routes off-limits to visitors', () => {
    for (const path of ['/dashboard', '/moderation', '/pages', '/users', '/sites', '/security', '/settings', '/transfer']) {
      expect(isPathAllowed(path, false)).toBe(false)
      expect(isPathAllowed(path, true)).toBe(true)
    }
  })

  it('lets everyone reach their own pages, and rejects unknown routes', () => {
    expect(isPathAllowed('/comments', false)).toBe(true)
    expect(isPathAllowed('/account', false)).toBe(true)
    expect(isPathAllowed('/login', false)).toBe(true)
    expect(isPathAllowed('/not-a-page', true)).toBe(false)
  })
})
