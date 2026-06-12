import { describe, expect, it } from 'vitest'

import type { Config } from '@/types/config'
import {
  applyRoutingDnsPreset,
  routingDnsPresetCatalog,
  validatePresetCatalogShape,
} from './routingDnsPresets'

const baseConfig = (): Config => ({
  log: {},
  dns: { servers: [], rules: [] },
  inbounds: [],
  outbounds: [],
  route: { rules: [], rule_set: [] },
  experimental: {},
})

describe('routing DNS preset catalog', () => {
  it('uses safe HTTPS SRS source URLs', () => {
    expect(validatePresetCatalogShape()).toBe(true)
    for (const preset of routingDnsPresetCatalog) {
      for (const source of preset.sources) {
        expect(source.url).toMatch(/^https:\/\//)
        expect(source.url).toContain('raw.githubusercontent.com')
        expect(source.url).not.toContain('@')
        expect(source.url).toMatch(/\.srs$/)
      }
    }
  })

  it('applies selected outbound tags without hardcoded proxy tags', () => {
    const result = applyRoutingDnsPreset(baseConfig(), 'ru-blocked-proxy', {
      proxyOutbound: 'my-proxy',
      directOutbound: 'my-direct',
    })

    expect(result.config.experimental.cache_file?.enabled).toBe(true)
    expect(result.config.route.rule_set.map((item: any) => item.download_detour)).toEqual(['my-proxy', 'my-direct'])
    expect(result.config.route.rules.map((item: any) => item.outbound)).toEqual(['my-direct', 'my-proxy'])
    expect(JSON.stringify(result.config)).not.toContain('"proxy"')
  })

  it('routes each side of zh-mainland-direct through its own outbound', () => {
    const { config } = applyRoutingDnsPreset(baseConfig(), 'zh-mainland-direct', {
      proxyOutbound: 'my-proxy',
      directOutbound: 'my-direct',
    })

    const detourByTag = Object.fromEntries(
      (config.route.rule_set as any[]).map(rs => [rs.tag, rs.download_detour]))
    expect(detourByTag['geosite-cn']).toBe('my-direct')
    expect(detourByTag['geoip-cn']).toBe('my-direct')
    expect(detourByTag['geosite-non-cn']).toBe('my-proxy')

    const outboundByRuleSet = (config.route.rules as any[]).map(r => ({ rule_set: r.rule_set, outbound: r.outbound }))
    expect(outboundByRuleSet).toContainEqual({ rule_set: ['geosite-cn', 'geoip-cn'], outbound: 'my-direct' })
    expect(outboundByRuleSet).toContainEqual({ rule_set: ['geosite-non-cn'], outbound: 'my-proxy' })
  })

  it('throws for an unknown preset id instead of silently doing nothing', () => {
    expect(() => applyRoutingDnsPreset(baseConfig(), 'does-not-exist', {
      proxyOutbound: 'my-proxy',
      directOutbound: 'my-direct',
    })).toThrow(/unknown preset/)
  })

  it('updates existing tagged objects instead of duplicating them', () => {
    const first = applyRoutingDnsPreset(baseConfig(), 'zh-mainland-direct', {
      proxyOutbound: 'proxy-a',
      directOutbound: 'direct-a',
    }).config
    const second = applyRoutingDnsPreset(first, 'zh-mainland-direct', {
      proxyOutbound: 'proxy-b',
      directOutbound: 'direct-b',
    }).config

    expect(second.route.rule_set.filter((item: any) => item.tag === 'geosite-cn')).toHaveLength(1)
    expect(second.dns.servers.filter((item: any) => item.tag === 'preset-dns-proxy')).toHaveLength(1)
    expect(second.route.rules.filter((item: any) => item.rule_set?.includes('geosite-non-cn'))).toHaveLength(1)
    expect(JSON.stringify(second)).toContain('proxy-b')
  })
})
