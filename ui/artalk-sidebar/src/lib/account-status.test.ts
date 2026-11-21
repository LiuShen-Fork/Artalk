import { expect, it } from 'vitest'
import { accountStatus } from './account-status'

it('treats a password as an account', () => {
  expect(accountStatus({ has_password: true }).isGuest).toBe(false)
})

it('treats a linked provider as an account', () => {
  const status = accountStatus({ identity_providers: ['github'] })
  expect(status.isGuest).toBe(false)
  expect(status.providers).toEqual(['github'])
})

it('treats a bare name and email profile as a guest', () => {
  expect(accountStatus({}).isGuest).toBe(true)
  expect(accountStatus({ has_password: false, identity_providers: [] }).isGuest).toBe(true)
})

it('never labels a config-file admin as a guest', () => {
  expect(accountStatus({ is_in_conf: true }).isGuest).toBe(false)
})
