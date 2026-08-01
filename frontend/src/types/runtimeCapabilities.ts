import { InTypes } from './inbounds'
import { OutTypes } from './outbounds'

export interface RuntimeCapabilityEntry {
  type: string
  buildTag?: string
  available: boolean
  [key: string]: unknown
}

export interface RuntimeInboundCapability extends RuntimeCapabilityEntry {
  clientDelivery?: string
  hasUsers?: boolean
  hasInData?: boolean
  hasTlsTemplate?: boolean
  muxAvailable?: boolean
  onlyTls?: boolean
  uiEditor?: string
}

export interface RuntimeGroupCapability extends RuntimeCapabilityEntry {
  coreType?: string
  assembledAs?: string
  panelManaged?: boolean
}

export interface RuntimeCapabilities {
  buildTags: Record<string, boolean>
  inbounds: RuntimeInboundCapability[]
  outbounds: RuntimeCapabilityEntry[]
  groups: RuntimeGroupCapability[]
  endpoints: RuntimeCapabilityEntry[]
  services: RuntimeCapabilityEntry[]
}

export type RuntimeCapabilityCategory = keyof Pick<RuntimeCapabilities, 'inbounds' | 'outbounds' | 'endpoints' | 'services' | 'groups'>
export interface TypeOption {
  title: string
  value: string
}
function isBuildTags(value: unknown): value is Record<string, boolean> {
  return value !== null
    && typeof value === 'object'
    && Object.values(value as Record<string, unknown>).every((tag) => typeof tag === 'boolean')
}

function isCapabilityEntry(value: unknown): value is RuntimeCapabilityEntry {
  if (value === null || typeof value !== 'object') return false
  const entry = value as { type?: unknown; available?: unknown }
  return typeof entry.type === 'string'
    && entry.type.length > 0
    && typeof entry.available === 'boolean'
}

function isCapabilityArray(value: unknown): value is RuntimeCapabilityEntry[] {
  return Array.isArray(value) && value.every(isCapabilityEntry)
}

export function isRuntimeCapabilities(value: unknown): value is RuntimeCapabilities {
  if (value === null || typeof value !== 'object') return false
  const candidate = value as Partial<RuntimeCapabilities> & { buildTags?: unknown }
  if (!isBuildTags(candidate.buildTags)) return false
  return isCapabilityArray(candidate.inbounds)
    && isCapabilityArray(candidate.outbounds)
    && isCapabilityArray(candidate.groups)
    && isCapabilityArray(candidate.endpoints)
    && isCapabilityArray(candidate.services)
}

export function runtimeEntries(
  capabilities: RuntimeCapabilities | null | undefined,
  category: RuntimeCapabilityCategory,
): RuntimeCapabilityEntry[] | undefined {
  return capabilities?.[category]
}

export function availableRuntimeTypes(
  capabilities: RuntimeCapabilities | null | undefined,
  category: RuntimeCapabilityCategory,
): string[] {
  return (runtimeEntries(capabilities, category) ?? [])
    .filter((entry) => entry.available)
    .map((entry) => entry.type)
}

export function isRuntimeTypeAvailable(
  capabilities: RuntimeCapabilities | null | undefined,
  category: RuntimeCapabilityCategory,
  type: string,
): boolean {
  return runtimeEntries(capabilities, category)?.some((entry) => entry.type === type && entry.available) ?? false
}

export function availableInboundEditorTypes(capabilities: RuntimeCapabilities | null | undefined): string[] {
  const available = new Set(availableRuntimeTypes(capabilities, 'inbounds'))
  return Object.values(InTypes).filter((type) => available.has(type))
}

export function availableOutboundEditorTypes(capabilities: RuntimeCapabilities | null | undefined): string[] {
  const available = new Set([
    ...availableRuntimeTypes(capabilities, 'outbounds'),
    ...availableRuntimeTypes(capabilities, 'groups'),
  ])
  return Object.values(OutTypes).filter((type) => available.has(type))
}

export function typeOptions(
  registry: Record<string, string>,
  availableTypes: readonly string[],
  historicalType?: string,
): TypeOption[] {
  const registryEntries = Object.entries(registry)
  const allowed = new Set(availableTypes)
  const historicalEntry = historicalType && registryEntries.find(([, value]) => value === historicalType)
  if (historicalEntry) allowed.add(historicalEntry[1])
  const options = registryEntries
    .filter(([, value]) => allowed.has(value))
    .map(([title, value]) => ({ title, value }))
  if (historicalType && !registryEntries.some(([, value]) => value === historicalType)) {
    options.push({ title: historicalType, value: historicalType })
  }
  return options
}

export function defaultInboundType(capabilities: RuntimeCapabilities | null | undefined): string | undefined {
  const available = availableInboundEditorTypes(capabilities)
  return available.includes(InTypes.Direct) ? InTypes.Direct : available[0]
}

export function defaultOutboundType(capabilities: RuntimeCapabilities | null | undefined): string | undefined {
  const available = availableOutboundEditorTypes(capabilities)
  return available.includes(OutTypes.Direct) ? OutTypes.Direct : available[0]
}

export function runtimeRecommendationTypeAvailable(
  capabilities: RuntimeCapabilities | null | undefined,
  category: 'inbounds' | 'outbounds',
  type: string,
): boolean {
  if (category === 'outbounds') {
    return isRuntimeTypeAvailable(capabilities, 'outbounds', type)
      || isRuntimeTypeAvailable(capabilities, 'groups', type)
  }
  return isRuntimeTypeAvailable(capabilities, category, type)
}
export function canSaveRuntimeType(
  capabilities: RuntimeCapabilities | null | undefined,
  category: 'inbounds' | 'outbounds',
  type: string,
  id: number,
  originalType?: string,
): boolean {
  if (id > 0 && type === originalType) return true
  return runtimeRecommendationTypeAvailable(capabilities, category, type)
}
