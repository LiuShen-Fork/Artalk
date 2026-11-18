import { describe, expect, it } from 'vitest'
import { allowsGuestToken } from './fetch'

describe('allowsGuestToken', () => {
  it('lets the commenter capability reach its own endpoints', () => {
    expect(allowsGuestToken('/comments')).toBe(true)
    expect(allowsGuestToken('/comments/42')).toBe(true)
    expect(allowsGuestToken('/notifies')).toBe(true)
    expect(allowsGuestToken('/notifies/read')).toBe(true)
    expect(allowsGuestToken('/user')).toBe(true)
    expect(allowsGuestToken('/user/activity')).toBe(true)
  })

  it('keeps it away from everything else', () => {
    expect(allowsGuestToken('/comments/42/vote')).toBe(false)
    expect(allowsGuestToken('/user/status')).toBe(false)
    expect(allowsGuestToken('/user/access_token')).toBe(false)
    expect(allowsGuestToken('/pv')).toBe(false)
    expect(allowsGuestToken('/login')).toBe(false)
    expect(allowsGuestToken('/user/activity/extra')).toBe(false)
  })
})
