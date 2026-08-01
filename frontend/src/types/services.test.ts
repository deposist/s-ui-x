import { describe, expect, it } from 'vitest'
import { availableSrvTypes, defaultSrvType } from './services'

describe('runtime service capabilities', () => {
  it('filters unavailable service types while retaining supported runtime types', () => {
    expect(availableSrvTypes([
      { type: 'resolved', available: true },
      { type: 'derp', available: false },
      { type: 'unknown', available: true },
    ])).toEqual({ Resolved: 'resolved' })
  })

  it('uses the first runtime-available type as the creation default', () => {
    expect(defaultSrvType([
      { type: 'derp', available: false },
      { type: 'ssm-api', available: true },
      { type: 'resolved', available: true },
    ])).toBe('resolved')
  })

  it('keeps official untagged types when runtime data is not available', () => {
    expect(availableSrvTypes()).toEqual({ Resolved: 'resolved', SSMAPI: 'ssm-api' })
    expect(defaultSrvType()).toBe('resolved')
  })
})
