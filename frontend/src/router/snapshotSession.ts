export interface SnapshotSessionDependencies {
  loadSnapshot: () => void | Promise<void>
  connectRealtime: () => void | Promise<void>
  disconnectRealtime: () => void
}

/**
 * Owns authenticated page entry and exit. WsRuntime remains the sole owner of
 * recurring degraded polling and reconnects, so a healthy socket never
 * competes with an unconditional router interval.
 */
export class SnapshotSession {
  private active = false

  constructor(private readonly deps: SnapshotSessionDependencies) {}

  enter() {
    if (this.active) return
    this.active = true
    void this.deps.loadSnapshot()
    void this.deps.connectRealtime()
  }

  leave() {
    if (!this.active) return
    this.active = false
    this.deps.disconnectRealtime()
  }
}
