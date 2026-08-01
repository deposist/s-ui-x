import { describe, expect, it, vi } from 'vitest'
import { SnapshotSession } from './snapshotSession'

describe('authenticated snapshot session', () => {
  it('loads one snapshot without installing healthy-socket polling', () => {
    const loadSnapshot = vi.fn()
    const connectRealtime = vi.fn()
    const disconnectRealtime = vi.fn()
    const setIntervalSpy = vi.spyOn(globalThis, 'setInterval')
    const session = new SnapshotSession({ loadSnapshot, connectRealtime, disconnectRealtime })

    session.enter()
    session.enter()
    session.enter()

    expect(loadSnapshot).toHaveBeenCalledTimes(1)
    expect(connectRealtime).toHaveBeenCalledTimes(1)
    expect(setIntervalSpy).not.toHaveBeenCalled()
    expect(disconnectRealtime).not.toHaveBeenCalled()
    setIntervalSpy.mockRestore()
  })

  it('disconnects once and starts one fresh snapshot after resume', () => {
    const loadSnapshot = vi.fn()
    const connectRealtime = vi.fn()
    const disconnectRealtime = vi.fn()
    const session = new SnapshotSession({ loadSnapshot, connectRealtime, disconnectRealtime })

    session.enter()
    session.leave()
    session.leave()
    session.enter()

    expect(loadSnapshot).toHaveBeenCalledTimes(2)
    expect(connectRealtime).toHaveBeenCalledTimes(2)
    expect(disconnectRealtime).toHaveBeenCalledTimes(1)
  })
})
