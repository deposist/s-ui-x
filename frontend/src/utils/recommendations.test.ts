import { describe, expect, it } from 'vitest'
import { applyRecommendation, isEmptyRecommendationValue, resolveRecommendations } from './recommendations'

const spec = {
  id: 'resolver',
  labelKey: 'guidance.title',
  path: 'domain_resolver.server',
  value: 'local',
  mode: 'create' as const,
  onlyIfEmpty: true,
}

describe('recommendation mechanics', () => {
  it('treats absent, blank, zero, empty arrays, and empty objects as empty', () => {
    expect([undefined, null, '', 0, [], {}].every(isEmptyRecommendationValue)).toBe(true)
    expect([false, 'value', 443, ['value'], { value: true }].some(isEmptyRecommendationValue)).toBe(false)
  })

  it('creates missing parents only after an explicit applicable action', () => {
    const model: Record<string, unknown> = {}
    const context = { mode: 'create' as const, model }
    expect(resolveRecommendations([spec], context)[0].applicable).toBe(true)
    expect(model).toEqual({})
    expect(applyRecommendation(spec, context)).toBe(true)
    expect(model).toEqual({ domain_resolver: { server: 'local' } })
  })

  it('does not overwrite operator input or apply create presets while editing', () => {
    const populated = { domain_resolver: { server: 'remote' } }
    expect(applyRecommendation(spec, { mode: 'create', model: populated })).toBe(false)
    expect(populated.domain_resolver.server).toBe('remote')

    const editing: Record<string, unknown> = {}
    expect(applyRecommendation(spec, { mode: 'edit', model: editing })).toBe(false)
    expect(editing).toEqual({})
  })
})
