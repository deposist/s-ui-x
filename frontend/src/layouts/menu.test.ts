import { describe, expect, it } from 'vitest'

import { appMenu, appMenuGroups, singBoxSettingsPaths } from './menu'

const expectedPaths = [
  '/',
  '/inbounds',
  '/clients',
  '/outbounds',
  '/endpoints',
  '/services',
  '/tls',
  '/rules',
  '/dns',
  '/telegram',
  '/paid-subscriptions',
  '/admins',
  '/audit',
  '/settings',
  '/donations',
]

describe('shared app menu', () => {
  it('derives one flat route list from the grouped source', () => {
    expect(appMenu).toEqual(appMenuGroups.flatMap(group => group.items))
    expect(appMenu.map(item => item.path)).toEqual(expectedPaths)
  })

  it('keeps every path unique', () => {
    const paths = appMenu.map(item => item.path)

    expect(new Set(paths).size).toBe(paths.length)
  })

  it('marks only sing-box editor surfaces', () => {
    expect(singBoxSettingsPaths).toEqual([
      '/inbounds',
      '/outbounds',
      '/endpoints',
      '/services',
      '/tls',
      '/rules',
      '/dns',
    ])
  })

  it('retains shell-specific icon metadata and Nexus badges', () => {
    expect(appMenu.every(item => item.classicIcon && item.nexusIcon)).toBe(true)
    expect(appMenu.find(item => item.path === '/clients')?.countKey).toBe('clients')
    expect(appMenu.find(item => item.path === '/paid-subscriptions')?.countKey).toBeUndefined()
  })
})
