import { describe, expect, it } from 'vitest'

import {
  authErrorMessage,
  forcedPasswordChangePayload,
  forcedPasswordResetUsername,
} from './loginFlow'

describe('forced password reset login flow', () => {
  it('detects reset metadata before normal success handling and uses server username', () => {
    expect(forcedPasswordResetUsername({
      success: false,
      msg: 'password reset required',
      obj: { forcePasswordReset: true, username: 'migrated-admin' },
    }, 'typed-admin')).toBe('migrated-admin')
  })

  it('falls back to the submitted username when reset metadata omits it', () => {
    expect(forcedPasswordResetUsername({
      success: false,
      msg: 'password reset required',
      obj: { forcePasswordReset: true },
    }, 'typed-admin')).toBe('typed-admin')
  })

  it('does not enter reset mode for ordinary failures or malformed metadata', () => {
    expect(forcedPasswordResetUsername({ success: false, msg: 'Invalid login', obj: null }, 'typed-admin')).toBeNull()
    expect(forcedPasswordResetUsername({ success: false, msg: 'reset', obj: { forcePasswordReset: 'true' } }, 'typed-admin')).toBeNull()
  })

  it('builds the exact reset payload and preserves reset failure text', () => {
    expect(forcedPasswordChangePayload('old-pass', 'new-admin', 'new-pass')).toEqual({
      oldPass: 'old-pass',
      newUsername: 'new-admin',
      newPass: 'new-pass',
    })
    expect(authErrorMessage({ success: false, msg: 'password policy failed', obj: null }, 'fallback')).toBe('password policy failed')
    expect(authErrorMessage({ success: false, msg: '', obj: null }, 'fallback')).toBe('fallback')
  })
})
