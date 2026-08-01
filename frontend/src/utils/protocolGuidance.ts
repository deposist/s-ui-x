import { InTypes } from '@/types/inbounds'
import type { RuntimeCapabilities } from '@/types/runtimeCapabilities'
import { runtimeRecommendationTypeAvailable } from '@/types/runtimeCapabilities'

export type ProtocolGuidanceCategory = 'inbounds' | 'outbounds'

const hintKeys: Record<string, string> = {
  [InTypes.VLESS]: 'guidance.protocols.vless',
  [InTypes.VMess]: 'guidance.protocols.vmess',
  [InTypes.Trojan]: 'guidance.protocols.trojan',
  [InTypes.Shadowsocks]: 'guidance.protocols.shadowsocks',
  [InTypes.SOCKS]: 'guidance.protocols.socks',
  [InTypes.HTTP]: 'guidance.protocols.http',
}

 
export const protocolGuidance = (
  capabilities: RuntimeCapabilities | null | undefined,
  category: ProtocolGuidanceCategory,
  type: string,
): { hintKey?: string; specs: [] } => {
  const runtimeCategory = category === 'inbounds' ? 'inbounds' : 'outbounds'
  if (!hintKeys[type] || !runtimeRecommendationTypeAvailable(capabilities, runtimeCategory, type)) {
    return { specs: [] }
  }
  return {
    hintKey: hintKeys[type],
    specs: [],
  }
}
