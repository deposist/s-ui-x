// Mirrors service.IpCertStatus (Go).
export interface IpCertStatus {
  enabled: boolean
  targetIp: string
  applyTarget: string
  issued: boolean
  notAfter: string
  lastIssue: string
  daysRemaining: number
  certPath: string
}

export interface IpCertIssueRequest {
  ip: string
  email: string
  port: number
  applyTarget: string
}
