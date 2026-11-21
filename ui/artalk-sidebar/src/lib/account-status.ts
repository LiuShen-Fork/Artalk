export interface AccountStatusInput {
  has_password?: boolean
  identity_providers?: string[]
  is_in_conf?: boolean
}

export interface AccountStatus {
  /** No password, no linked provider: a profile left behind by anonymous commenting. */
  isGuest: boolean
  hasPassword: boolean
  providers: string[]
}

/**
 * Artalk keeps no "email verified" flag: an email code is a one-off challenge.
 * The durable sign of a real account is a password or a linked OAuth identity;
 * everything else is a guest profile. Config-file admins are accounts even if
 * their password column were somehow empty.
 */
export function accountStatus(user: AccountStatusInput): AccountStatus {
  const hasPassword = user.has_password === true
  const providers = user.identity_providers ?? []
  return {
    isGuest: !hasPassword && providers.length === 0 && user.is_in_conf !== true,
    hasPassword,
    providers,
  }
}
