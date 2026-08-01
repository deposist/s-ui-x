import { Listen } from "./inbounds"
import { iTls } from "./tls"
import { serviceCapabilities } from './capabilities'

export const SrvTypes = {
  DERP: 'derp',
  Resolved: 'resolved',
  SSMAPI: 'ssm-api',
} as const

export type SrvType = typeof SrvTypes[keyof typeof SrvTypes]


interface SrvBasics extends Listen {
  id: number
  type: SrvType
  tag: string
  tls_id: number
}

export interface DERP extends SrvBasics {
  tls: iTls
  config_path: string
  verify_client_endpoint?: string[]
  verify_client_url?: any[]
  home?: string
  mesh_with?: any[]
  mesh_psk?: string
  mesh_psk_file?: string
  stun?: any
}

export interface Resolved extends SrvBasics {}

export interface SSMAPI extends SrvBasics {
  servers: any
  cache_path?: string
  tls?: iTls
}


type InterfaceMap = {
  derp: DERP
  resolved: Resolved
  'ssm-api': SSMAPI
}

export type Srv = InterfaceMap[keyof InterfaceMap]

const defaultValues: Record<SrvType, Srv> = {
  derp: <DERP>{ type: 'derp', config_path: '', tls_id: 0 },
  resolved: <Resolved>{ type: 'resolved', listen: '::', listen_port: 53 },
  'ssm-api': <SSMAPI>{ type: 'ssm-api', tls_id: 0, servers: {} },
}

export function availableSrvTypes(runtimeCapabilities?: Array<{ type: string; available?: boolean }>): Record<string, SrvType> {
  const source = runtimeCapabilities?.length
    ? runtimeCapabilities.filter(({ available }) => available !== false).map(({ type }) => type)
    : serviceCapabilities.filter(({ buildTag }) => buildTag === '').map(({ type }) => type)
  const allowed = new Set(source)
  return Object.fromEntries(
    Object.entries(SrvTypes).filter(([, type]) => allowed.has(type)),
  ) as Record<string, SrvType>
}

export function defaultSrvType(runtimeCapabilities?: Array<{ type: string; available?: boolean }>): SrvType {
  return Object.values(availableSrvTypes(runtimeCapabilities))[0] ?? SrvTypes.Resolved
}

export function createSrv<T extends Srv>(type: string, json?: Partial<T>): Srv {
  const defaultObject: Srv = { ...defaultValues[type as SrvType], ...(json || {}) }
  return defaultObject
}
