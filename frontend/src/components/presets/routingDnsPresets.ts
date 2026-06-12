import type { Config } from '@/types/config'

export type PresetRegion = 'RU' | 'ZH'

export interface PresetSource {
  name: string
  url: string
}

export interface ApplyPresetOptions {
  proxyOutbound: string
  directOutbound: string
}

export interface ApplyPresetResult {
  config: Config
  changes: string[]
}

export type PresetApplier = (config: Config, options: ApplyPresetOptions, changes: string[]) => void

export interface RoutingDnsPreset {
  id: string
  region: PresetRegion
  titleKey: string
  descriptionKey: string
  sources: PresetSource[]
  apply: PresetApplier
}

const SOURCE_URLS = {
  cnGeosite: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-cn.srs',
  nonCnGeosite: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs',
  cnGeoip: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
  ruBlocked: 'https://raw.githubusercontent.com/runetfreedom/russia-blocked-geoip/release/srs/re-filter.srs',
  ruPrivate: 'https://raw.githubusercontent.com/runetfreedom/russia-blocked-geoip/release/srs/private.srs',
} as const

const cloneConfig = (config: Config): Config => JSON.parse(JSON.stringify(config ?? {}))

const ensureConfigShape = (config: Config) => {
  if (!config.route) config.route = { rules: [], rule_set: [] }
  if (!Array.isArray(config.route.rules)) config.route.rules = []
  if (!Array.isArray(config.route.rule_set)) config.route.rule_set = []
  if (!config.dns) config.dns = { servers: [], rules: [] }
  if (!Array.isArray(config.dns.servers)) config.dns.servers = []
  if (!Array.isArray(config.dns.rules)) config.dns.rules = []
  if (!config.experimental) config.experimental = {}
  if (!config.experimental.cache_file) config.experimental.cache_file = {}
}

const upsertByTag = (items: any[], item: any, changes: string[], label: string) => {
  const index = items.findIndex(existing => existing?.tag === item.tag)
  if (index === -1) {
    items.push(item)
    changes.push(`add ${label} ${item.tag}`)
    return
  }
  items[index] = { ...items[index], ...item }
  changes.push(`update ${label} ${item.tag}`)
}

const asArray = (value: unknown): string[] => {
  if (Array.isArray(value)) return value.map(String)
  if (typeof value === 'string') return [value]
  return []
}

const sameMembers = (left: string[], right: string[]) => {
  if (left.length !== right.length) return false
  const set = new Set(left)
  return right.every(item => set.has(item))
}

const upsertRuleByRuleSet = (items: any[], rule: any, changes: string[], label: string) => {
  const target = asArray(rule.rule_set)
  const index = items.findIndex(existing => sameMembers(asArray(existing?.rule_set), target))
  if (index === -1) {
    items.push(rule)
    changes.push(`add ${label} ${target.join(', ')}`)
    return
  }
  items[index] = { ...items[index], ...rule }
  changes.push(`update ${label} ${target.join(', ')}`)
}

const remoteRuleSet = (tag: string, url: string, downloadDetour: string) => ({
  type: 'remote',
  tag,
  format: 'binary',
  url,
  download_detour: downloadDetour,
  update_interval: '24h',
})

const dnsServer = (tag: string, server: string, detour: string) => ({
  type: 'udp',
  tag,
  server,
  server_port: 53,
  detour,
})

const applyCommon = (config: Config, options: ApplyPresetOptions, changes: string[]) => {
  if (config.experimental.cache_file?.enabled !== true) {
    config.experimental.cache_file!.enabled = true
    changes.push('enable experimental.cache_file')
  }

  upsertByTag(config.dns.servers as any[], dnsServer('preset-dns-direct', '223.5.5.5', options.directOutbound), changes, 'dns server')
  upsertByTag(config.dns.servers as any[], dnsServer('preset-dns-proxy', '1.1.1.1', options.proxyOutbound), changes, 'dns server')
}

const applyZhMainlandDirect: PresetApplier = (config, options, changes) => {
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('geosite-cn', SOURCE_URLS.cnGeosite, options.directOutbound), changes, 'rule-set')
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('geoip-cn', SOURCE_URLS.cnGeoip, options.directOutbound), changes, 'rule-set')
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('geosite-non-cn', SOURCE_URLS.nonCnGeosite, options.proxyOutbound), changes, 'rule-set')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['geosite-cn', 'geoip-cn'], outbound: options.directOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['geosite-non-cn'], outbound: options.proxyOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['geosite-cn'], server: 'preset-dns-direct' }, changes, 'dns rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['geosite-non-cn'], server: 'preset-dns-proxy' }, changes, 'dns rule')
}

const applyZhNonCnProxy: PresetApplier = (config, options, changes) => {
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('geosite-non-cn', SOURCE_URLS.nonCnGeosite, options.proxyOutbound), changes, 'rule-set')
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('geosite-cn', SOURCE_URLS.cnGeosite, options.directOutbound), changes, 'rule-set')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['geosite-non-cn'], outbound: options.proxyOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['geosite-cn'], outbound: options.directOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['geosite-cn'], server: 'preset-dns-direct' }, changes, 'dns rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['geosite-non-cn'], server: 'preset-dns-proxy' }, changes, 'dns rule')
}

const applyRuBlockedProxy: PresetApplier = (config, options, changes) => {
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('ru-blocked', SOURCE_URLS.ruBlocked, options.proxyOutbound), changes, 'rule-set')
  upsertByTag(config.route.rule_set as any[], remoteRuleSet('ru-private', SOURCE_URLS.ruPrivate, options.directOutbound), changes, 'rule-set')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['ru-private'], outbound: options.directOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.route.rules as any[], { rule_set: ['ru-blocked'], outbound: options.proxyOutbound }, changes, 'route rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['ru-private'], server: 'preset-dns-direct' }, changes, 'dns rule')
  upsertRuleByRuleSet(config.dns.rules as any[], { action: 'route', rule_set: ['ru-blocked'], server: 'preset-dns-proxy' }, changes, 'dns rule')
}

export const routingDnsPresetCatalog: RoutingDnsPreset[] = [
  {
    id: 'zh-mainland-direct',
    region: 'ZH',
    titleKey: 'presets.zhMainlandDirect',
    descriptionKey: 'presets.zhMainlandDirectDesc',
    sources: [
      { name: 'SagerNet/sing-geosite: geolocation-cn', url: SOURCE_URLS.cnGeosite },
      { name: 'SagerNet/sing-geoip: cn', url: SOURCE_URLS.cnGeoip },
      { name: 'SagerNet/sing-geosite: geolocation-!cn', url: SOURCE_URLS.nonCnGeosite },
    ],
    apply: applyZhMainlandDirect,
  },
  {
    id: 'zh-non-cn-proxy',
    region: 'ZH',
    titleKey: 'presets.zhNonCnProxy',
    descriptionKey: 'presets.zhNonCnProxyDesc',
    sources: [
      { name: 'SagerNet/sing-geosite: geolocation-!cn', url: SOURCE_URLS.nonCnGeosite },
      { name: 'SagerNet/sing-geosite: geolocation-cn', url: SOURCE_URLS.cnGeosite },
    ],
    apply: applyZhNonCnProxy,
  },
  {
    id: 'ru-blocked-proxy',
    region: 'RU',
    titleKey: 'presets.ruBlockedProxy',
    descriptionKey: 'presets.ruBlockedProxyDesc',
    sources: [
      { name: 'runetfreedom/russia-blocked-geoip: re-filter', url: SOURCE_URLS.ruBlocked },
      { name: 'runetfreedom/russia-blocked-geoip: private', url: SOURCE_URLS.ruPrivate },
    ],
    apply: applyRuBlockedProxy,
  },
]

export const applyRoutingDnsPreset = (
  input: Config,
  presetId: string,
  options: ApplyPresetOptions,
): ApplyPresetResult => {
  if (!options.proxyOutbound || !options.directOutbound) {
    throw new Error('proxyOutbound and directOutbound are required')
  }

  const preset = routingDnsPresetCatalog.find(item => item.id === presetId)
  if (!preset) throw new Error(`unknown preset: ${presetId}`)

  const config = cloneConfig(input)
  const changes: string[] = []
  ensureConfigShape(config)
  applyCommon(config, options, changes)
  // The apply function is bound to the catalog entry, so a preset can never be
  // "found" yet silently skipped at apply time.
  preset.apply(config, options, changes)

  return { config, changes }
}

export const validatePresetCatalogShape = () => {
  return routingDnsPresetCatalog.every(preset =>
    typeof preset.apply === 'function' &&
    preset.sources.length > 0 &&
    preset.sources.every(source => {
      const parsed = new URL(source.url)
      return parsed.protocol === 'https:' &&
        parsed.username === '' &&
        parsed.password === '' &&
        parsed.pathname.endsWith('.srs')
    }))
}
