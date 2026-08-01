import { describe, expect, it } from 'vitest'
import { isPreloadError } from './preload-error'

describe('isPreloadError', () => {
  it.each([
    'Failed to fetch dynamically imported module: http://example.test/app/assets/chunk.js',
    'Importing a module script failed',
    'Failed to load module script',
    'error loading dynamically imported module: http://example.test/app/assets/chunk.js',
  ])('recognizes browser preload failure %j', (message) => {
    expect(isPreloadError(new TypeError(message))).toBe(true)
  })

  it('recognizes bundler chunk errors by name', () => {
    expect(isPreloadError({ name: 'ChunkLoadError' })).toBe(true)
  })

  it('ignores unrelated errors', () => {
    expect(isPreloadError(new Error('login failed'))).toBe(false)
  })
})
