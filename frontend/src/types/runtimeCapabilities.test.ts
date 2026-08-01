import { describe, expect, it } from 'vitest'
import {
  availableInboundEditorTypes,
  availableOutboundEditorTypes,
  canSaveRuntimeType,
  defaultInboundType,
  defaultOutboundType,
  isRuntimeCapabilities,
  runtimeRecommendationTypeAvailable,
  typeOptions,
} from './runtimeCapabilities'

const capability = (type: string, available = true) => ({ type, available })

const runtimeCapabilities = {
  buildTags: { with_quic: false },
  inbounds: [capability('direct'), capability('vless'), capability('naive', false)],
  outbounds: [capability('direct'), capability('vless'), capability('naive', false), capability('block')],
  groups: [capability('selector'), capability('urltest', false), capability('failover')],
  endpoints: [capability('wireguard', false)],
  services: [capability('resolved')],
}

describe('runtime capability contract', () => {
  it('requires every backend category and concrete availability flags', () => {
    expect(isRuntimeCapabilities(runtimeCapabilities)).toBe(true)
    expect(isRuntimeCapabilities({ ...runtimeCapabilities, groups: undefined })).toBe(false)
    expect(isRuntimeCapabilities({ ...runtimeCapabilities, outbounds: [{ type: 'direct' }] })).toBe(false)
    expect(isRuntimeCapabilities({ ...runtimeCapabilities, buildTags: { with_quic: 'false' } })).toBe(false)
  })

  it('rejects malformed backend payloads without falling back to compile-time inventory', () => {
    expect(isRuntimeCapabilities(null)).toBe(false)
    expect(isRuntimeCapabilities({ ...runtimeCapabilities, inbounds: 'direct' })).toBe(false)
    expect(isRuntimeCapabilities({ ...runtimeCapabilities, services: [null] })).toBe(false)
    expect(availableInboundEditorTypes(undefined)).toEqual([])
    expect(availableOutboundEditorTypes(null)).toEqual([])
    expect(defaultInboundType(undefined)).toBeUndefined()
    expect(defaultOutboundType(null)).toBeUndefined()
  })

  it('treats a present empty category as an authoritative empty set', () => {
    const empty = {
      ...runtimeCapabilities,
      inbounds: [],
      outbounds: [],
      groups: [],
    }
    expect(isRuntimeCapabilities(empty)).toBe(true)
    expect(availableInboundEditorTypes(empty)).toEqual([])
    expect(availableOutboundEditorTypes(empty)).toEqual([])
    expect(defaultInboundType(empty)).toBeUndefined()
    expect(defaultOutboundType(empty)).toBeUndefined()
  })

  it('intersects runtime availability with editor registries and combines outbound groups', () => {
    expect(availableInboundEditorTypes(runtimeCapabilities)).toEqual(['direct', 'vless'])
    expect(availableOutboundEditorTypes(runtimeCapabilities)).toEqual(['direct', 'vless', 'selector', 'failover'])
  })

  it('uses group availability for outbound recommendations only', () => {
    expect(runtimeRecommendationTypeAvailable(runtimeCapabilities, 'outbounds', 'selector')).toBe(true)
    expect(runtimeRecommendationTypeAvailable(runtimeCapabilities, 'outbounds', 'urltest')).toBe(false)
    expect(runtimeRecommendationTypeAvailable(runtimeCapabilities, 'inbounds', 'selector')).toBe(false)
  })

  it('keeps an unavailable historical type selectable without making it a creation default', () => {
    expect(typeOptions({ Direct: 'direct', Naive: 'naive', VLESS: 'vless' }, ['direct', 'vless'], 'naive'))
      .toEqual([
        { title: 'Direct', value: 'direct' },
        { title: 'Naive', value: 'naive' },
        { title: 'VLESS', value: 'vless' },
      ])
    expect(defaultInboundType(runtimeCapabilities)).toBe('direct')
    expect(defaultOutboundType(runtimeCapabilities)).toBe('direct')
  })

  it('preserves an unknown historical type without exposing it to new entities', () => {
    expect(typeOptions({ Direct: 'direct' }, ['direct'], 'legacy-protocol')).toEqual([
      { title: 'Direct', value: 'direct' },
      { title: 'legacy-protocol', value: 'legacy-protocol' },
    ])
    expect(typeOptions({ Direct: 'direct' }, [], undefined)).toEqual([])
  })

  it('allows same-type historical edits but rejects new and type-changing unavailable entities', () => {
    expect(canSaveRuntimeType(runtimeCapabilities, 'inbounds', 'naive', 5, 'naive')).toBe(true)
    expect(canSaveRuntimeType(runtimeCapabilities, 'inbounds', 'naive', 0)).toBe(false)
    expect(canSaveRuntimeType(runtimeCapabilities, 'inbounds', 'naive', 5, 'direct')).toBe(false)
    expect(canSaveRuntimeType(runtimeCapabilities, 'outbounds', 'selector', 0)).toBe(true)
    expect(canSaveRuntimeType(undefined, 'outbounds', 'direct', 0)).toBe(false)
  })
})
