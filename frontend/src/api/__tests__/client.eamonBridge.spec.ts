import { describe, expect, it } from 'vitest'
import { resolveAdminRequestPath } from '../client'

describe('eamon admin request bridge', () => {
  it('rewrites account-management requests on the eamon page', () => {
    expect(resolveAdminRequestPath('/admin/accounts?page=2', '/admin/eamon-accounts'))
      .toBe('/admin/integration/eamon/proxy/accounts?page=2')
    expect(resolveAdminRequestPath('/admin/accounts/42/test', '/admin/eamon-accounts'))
      .toBe('/admin/integration/eamon/proxy/accounts/42/test')
    expect(resolveAdminRequestPath('/admin/groups/all', '/admin/eamon-accounts'))
      .toBe('/admin/integration/eamon/proxy/groups/all')
  })

  it('does not rewrite unrelated admin areas', () => {
    expect(resolveAdminRequestPath('/admin/users', '/admin/eamon-accounts')).toBe('/admin/users')
    expect(resolveAdminRequestPath('/admin/system/update', '/admin/eamon-accounts')).toBe('/admin/system/update')
  })

  it('does not affect the local account page', () => {
    expect(resolveAdminRequestPath('/admin/accounts', '/admin/accounts')).toBe('/admin/accounts')
  })
})
