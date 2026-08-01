export const isPreloadError = (err: unknown): boolean => {
  if (!err) return false
  const candidate = err as { message?: unknown, name?: unknown }
  const message = typeof candidate.message === 'string' ? candidate.message : String(err)
  return /Failed to fetch dynamically imported module/i.test(message) ||
    /Importing a module script failed/i.test(message) ||
    /Failed to load module script/i.test(message) ||
    /error loading dynamically imported module/i.test(message) ||
    candidate.name === 'ChunkLoadError'
}
