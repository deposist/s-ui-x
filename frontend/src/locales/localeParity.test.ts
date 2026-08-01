import { describe, expect, it } from 'vitest'

import en from './en'
import fa from './fa'
import ru from './ru'
import vi from './vi'
import zhcn from './zhcn'
import zhtw from './zhtw'

const locales = { en, fa, ru, vi, zhcn, zhtw } as const

// Every shipped locale must have the same leaf-key structure. Runtime fallback
// remains useful for dynamic misses, but it must not hide incomplete bundles.
const flatten = (obj: Record<string, unknown>, prefix = ''): string[] => {
  const out: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      out.push(...flatten(v as Record<string, unknown>, key))
    } else {
      out.push(key)
    }
  }
  return out
}

describe('locale key parity', () => {
  const referenceKeys = new Set(flatten(en as Record<string, unknown>))

  for (const [locale, messages] of Object.entries(locales)) {
    if (locale === 'en') continue
    const keys = new Set(flatten(messages as Record<string, unknown>))

    it(`${locale} defines every key en defines`, () => {
      const missing = [...referenceKeys].filter((key) => !keys.has(key)).sort()
      expect(missing, `keys in en but missing from ${locale}: ${missing.join(', ')}`).toEqual([])
    })

    it(`${locale} does not define keys absent from en`, () => {
      const extra = [...keys].filter((key) => !referenceKeys.has(key)).sort()
      expect(extra, `keys in ${locale} but missing from en: ${extra.join(', ')}`).toEqual([])
    })
  }
})
