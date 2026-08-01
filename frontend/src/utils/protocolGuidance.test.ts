import { describe, expect, it } from 'vitest'
import type { RuntimeCapabilities } from '@/types/runtimeCapabilities'
import { protocolGuidance } from './protocolGuidance'

const capability = (type: string, available = true) => ({ type, available })
const capabilities: RuntimeCapabilities = {
  buildTags: {},
  inbounds: ['vless', 'vmess', 'trojan', 'shadowsocks', 'socks', 'http'].map((type) => capability(type)),
  outbounds: ['vless', 'vmess', 'trojan', 'shadowsocks', 'socks', 'http'].map((type) => capability(type)),
  groups: [],
  endpoints: [],
  services: [],
}

describe('official protocol guidance', () => {
  it.each(['vless', 'vmess', 'trojan', 'shadowsocks', 'socks', 'http'])(
    'provides a localized field hint for available %s editors',
    (type) => {
      expect(protocolGuidance(capabilities, 'inbounds', type).hintKey).toBe(`guidance.protocols.${type}`)
      expect(protocolGuidance(capabilities, 'outbounds', type).hintKey).toBe(`guidance.protocols.${type}`)
    },
  )

  it('keeps guidance hint-only when no universal safe field preset exists', () => {
    for (const category of ['inbounds', 'outbounds'] as const) {
      for (const type of ['vless', 'vmess', 'trojan', 'shadowsocks', 'socks', 'http']) {
        expect(protocolGuidance(capabilities, category, type).specs).toEqual([])
      }
    }
  })

  it('fails closed for missing, unavailable, and non-priority runtime types', () => {
    expect(protocolGuidance(null, 'inbounds', 'vless')).toEqual({ specs: [] })
    const unavailable = { ...capabilities, outbounds: [capability('vless', false)] }
    expect(protocolGuidance(unavailable, 'outbounds', 'vless')).toEqual({ specs: [] })
    expect(protocolGuidance(capabilities, 'outbounds', 'naive')).toEqual({ specs: [] })
  })

  it('contains no extended-only protocol identifiers', () => {
    const source = JSON.stringify(
      ['vless', 'vmess', 'trojan', 'shadowsocks', 'socks', 'http']
        .flatMap((type) => [protocolGuidance(capabilities, 'inbounds', type), protocolGuidance(capabilities, 'outbounds', type)]),
    )
    for (const forbidden of ['trusttunnel', 'mieru', 'sudoku', 'amnezia', 'mtproxy', 'masque', 'openvpn']) {
      expect(source.toLowerCase()).not.toContain(forbidden)
    }
  })
})
