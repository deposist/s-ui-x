import { describe, expect, it } from 'vitest'

import {
  endpointCapabilities,
  inboundTypes,
  outboundCapabilities,
  serviceCapabilities,
} from './capabilities'

const forbidden: Record<string, true> = {
  mieru: true,
  sudoku: true,
  trusttunnel: true,
  mtproxy: true,
  bond: true,
  'core-failover': true,
  masque: true,
  openvpn: true,
  vpn: true,
  ccm: true,
  ocm: true,
  'oom-killer': true,
  profiler: true,
}

describe('generated official capability projection', () => {
  it('contains only official protocol types', () => {
    const projected = [
      ...inboundTypes,
      ...outboundCapabilities.map(({ type }) => type),
      ...endpointCapabilities.map(({ type }) => type),
      ...serviceCapabilities.map(({ type }) => type),
    ]
    expect(projected.filter((type) => forbidden[type])).toEqual([])
  })

  it('keeps official build-tag availability metadata', () => {
    expect(outboundCapabilities.find(({ type }) => type === 'naive')?.buildTag).toBe('with_naive_outbound')
    expect(outboundCapabilities.find(({ type }) => type === 'hysteria2')?.buildTag).toBe('with_quic')
    expect(endpointCapabilities.find(({ type }) => type === 'wireguard')?.buildTag).toBe('with_wireguard')
    expect(endpointCapabilities.find(({ type }) => type === 'tailscale')?.buildTag).toBe('with_tailscale')
  })
})
