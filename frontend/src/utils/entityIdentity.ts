// Keep the frontend guard aligned with the backend reference validation.
export function isBlankIdentity(value: string | null | undefined): boolean {
  return value == null || value.trim() === ''
}
