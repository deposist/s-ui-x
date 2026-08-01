import { describe, expect, it } from 'vitest'
import classicInbound from '@/layouts/modals/Inbound.vue?raw'
import classicOutbound from '@/layouts/modals/Outbound.vue?raw'
import nexusInbound from '@/components/nexus/drawers/InboundDrawer.vue?raw'
import nexusOutbound from '@/components/nexus/drawers/OutboundDrawer.vue?raw'

const editors = [
  ['Classic inbound', classicInbound, 'inbounds', 'inbound'],
  ['Classic outbound', classicOutbound, 'outbounds', 'outbound'],
  ['Nexus inbound', nexusInbound, 'inbounds', 'inbound'],
  ['Nexus outbound', nexusOutbound, 'outbounds', 'outbound'],
] as const

describe('protocol guidance editor contracts', () => {
  it.each(editors)('%s wires runtime capabilities and the current model', (_, source, category, model) => {
    expect(source).toContain("import ProtocolGuidance from '@/components/recommendations/ProtocolGuidance.vue'")
    expect(source).toMatch(/components:\s*\{[\s\S]*ProtocolGuidance/)
    expect(source).toContain('<ProtocolGuidance')
    expect(source).toContain(':capabilities="capabilities"')
    expect(source).toContain(`category="${category}"`)
    expect(source).toContain(":mode=\"id > 0 ? 'edit' : 'create'\"")
    expect(source).toContain(`:model="${model}"`)
    expect(source).toContain('return Data().capabilities')
  })
})
