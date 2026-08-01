import { describe, expect, it } from 'vitest'
import classicInbound from '@/layouts/modals/Inbound.vue?raw'
import classicOutbound from '@/layouts/modals/Outbound.vue?raw'
import nexusInbound from '@/components/nexus/drawers/InboundDrawer.vue?raw'
import nexusOutbound from '@/components/nexus/drawers/OutboundDrawer.vue?raw'

const editors = [
  ['Classic inbound', classicInbound, 'availableInboundEditorTypes', 'defaultInboundType', "canSaveType('inbounds'"],
  ['Classic outbound', classicOutbound, 'availableOutboundEditorTypes', 'defaultOutboundType', "canSaveType('outbounds'"],
  ['Nexus inbound', nexusInbound, 'availableInboundEditorTypes', 'defaultInboundType', "canSaveType('inbounds'"],
  ['Nexus outbound', nexusOutbound, 'availableOutboundEditorTypes', 'defaultOutboundType', "canSaveType('outbounds'"],
] as const

describe('runtime-capability editor contracts', () => {
  it.each(editors)('%s filters choices, seeds a runtime default, and blocks unsupported saves', (name, source, availableHelper, defaultHelper, saveGuard) => {
    expect(source).toContain(':items="typeOptions"')
    expect(source).toContain(`${availableHelper}(Data().capabilities)`)
    expect(source.match(new RegExp(`${defaultHelper}\\(Data\\(\\)\\.capabilities\\)`, 'g'))?.length).toBeGreaterThanOrEqual(2)
    expect(source).toMatch(/this\.(?:inbound|outbound)\?\.type/)
    expect(source).toContain(saveGuard)
    expect(source).toContain('form.cannotSave.capabilityUnavailable')

    if (name.startsWith('Classic')) {
      expect(source).toContain(':disabled="loading || !validate"')
      expect(source).toContain('<span v-if="saveBlockedReason" class="text-error text-caption">')
      expect(source).toContain('{{ saveBlockedReason }}')
    }
  })
})
