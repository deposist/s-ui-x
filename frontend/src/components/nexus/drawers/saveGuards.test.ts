import { describe, expect, it } from 'vitest'
import { isBlankIdentity } from '@/utils/entityIdentity'
import inboundSource from './InboundDrawer.vue?raw'
import outboundSource from './OutboundDrawer.vue?raw'
import serviceSource from './ServiceDrawer.vue?raw'
import tlsSource from './TlsDrawer.vue?raw'

const blankIdentities = ['', ' ', '   ', '\t', '\n', ' \t ']

const drawerSources: Record<string, string> = {
  Inbound: inboundSource,
  Outbound: outboundSource,
  Service: serviceSource,
  Tls: tlsSource,
}

const drawerSource = (name: string) => drawerSources[name]

describe('drawer save identity guards', () => {
  it.each(blankIdentities)('classifies blank identity %j', (value) => {
    expect(isBlankIdentity(value)).toBe(true)
  })

  it('allows nonblank identities including zero', () => {
    expect(isBlankIdentity('0')).toBe(false)
    expect(isBlankIdentity('resolved-1')).toBe(false)
    expect(isBlankIdentity('tls-1')).toBe(false)
  })

  it.each([
    ['Inbound', 'inbound.tag', 'form.cannotSave.tagRequired'],
    ['Outbound', 'outbound.tag', 'form.cannotSave.tagRequired'],
    ['Service', 'srv.tag', 'form.cannotSave.tagRequired'],
    ['Tls', 'tls.name', 'form.cannotSave.nameRequired'],
  ])('%s drawer derives its disabled reason from the shared blank rule', (name, field, reason) => {
    const source = drawerSource(name)
    expect(source).toContain(`:error="isBlankIdentity(${field})"`)
    expect(source).toContain(`if (isBlankIdentity(this.${field})) return this.$t('${reason}')`)
    expect(source).toContain(':save-disabled-reason="saveBlockedReason"')
  })

  it.each(['Inbound', 'Outbound', 'Service', 'Tls'])('%s drawer rejects direct save calls when blocked', (name) => {
    const source = drawerSource(name)
    expect(source).toMatch(/saveChanges\(\) \{[\s\S]{0,180}saveBlockedReason|async saveChanges\(\) \{[\s\S]{0,180}(?:saveBlockedReason|!this\.validate)/)
  })

  it('preserves inbound port and required-TLS validation reasons', () => {
    const source = drawerSource('Inbound')
    expect(source).toContain("this.$t('form.cannotSave.portRange')")
    expect(source).toContain("this.$t('form.cannotSave.tlsRequired')")
  })
})
