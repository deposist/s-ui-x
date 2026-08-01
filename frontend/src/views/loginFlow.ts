import type { Msg } from '@/plugins/httputil'

export type LoginFlowResponse = Pick<Msg, 'success' | 'msg' | 'obj'>

export function forcedPasswordResetUsername(response: LoginFlowResponse, fallback: string): string | null {
  if (!response.obj || typeof response.obj !== 'object') return null
  const obj = response.obj as Record<string, unknown>
  if (obj.forcePasswordReset !== true) return null
  return typeof obj.username === 'string' && obj.username !== '' ? obj.username : fallback
}

export function authErrorMessage(response: LoginFlowResponse, fallback: string): string {
  return response.msg || fallback
}

export function forcedPasswordChangePayload(oldPass: string, newUsername: string, newPass: string) {
  return { oldPass, newUsername, newPass }
}
