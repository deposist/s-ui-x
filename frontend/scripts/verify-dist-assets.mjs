import { lstat, readFile, readdir } from 'node:fs/promises'
import { basename, dirname, extname, join, normalize, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const distDir = resolve(process.argv[2] ?? fileURLToPath(new URL('../dist', import.meta.url)))

const walk = async (dir) => {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const fullPath = join(dir, entry.name)
    const stat = await lstat(fullPath)
    if (stat.isSymbolicLink()) {
      throw new Error(`dist contains a symbolic link: ${relative(distDir, fullPath)}`)
    }
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath))
    } else if (entry.isFile()) {
      files.push(fullPath)
    } else {
      throw new Error(`dist contains a non-regular entry: ${relative(distDir, fullPath)}`)
    }
  }
  return files
}

const normalizeRelative = (file) => file.split(sep).join('/')

const resolveReference = (source, reference) => {
  const value = reference.trim()
  if (!value || value.startsWith('#') || value.startsWith('//') || /^[a-z][a-z0-9+.-]*:/i.test(value)) return null
  const withoutQuery = value.split(/[?#]/, 1)[0]
  if (!withoutQuery) return null
  let decoded
  try {
    decoded = decodeURIComponent(withoutQuery)
  } catch {
    return { reference: value, path: null }
  }
  const target = decoded.startsWith('/')
    ? decoded.slice(1)
    : normalize(join(dirname(source), decoded))
  return { reference: value, path: normalizeRelative(target) }
}

const extractReferences = (source, text) => {
  const references = []
  const add = (pattern) => {
    for (const match of text.matchAll(pattern)) references.push(match[1])
  }
  if (source.endsWith('.html')) {
    add(/\b(?:src|href)\s*=\s*["']([^"']+)["']/gi)
  }
  if (source.endsWith('.js') || source.endsWith('.mjs')) {
    add(/\bimport\s*\(\s*["']([^"']+)["']\s*\)/g)
    add(/\b(?:import|export)\s+(?:[^"']+?\s+from\s+)?["']([^"']+)["']/g)
    add(/\bnew\s+URL\(\s*["']([^"']+)["']\s*,\s*import\.meta\.url\s*\)/g)
  }
  if (source.endsWith('.css')) add(/\burl\(\s*["']?([^"')]+)["']?\s*\)/gi)
  return references
}

const files = await walk(distDir)
const relativeFiles = files.map(file => normalizeRelative(relative(distDir, file)))
const fileSet = new Set(relativeFiles)
const blocked = relativeFiles.filter(file => {
  const name = basename(file)
  return file === '..' || file.startsWith('../') || name.startsWith('_') || name.startsWith('.')
})
const errors = []

if (blocked.length > 0) {
  errors.push('dist contains files that Go //go:embed directory walks skip or cannot safely embed:')
  errors.push(...blocked.map(file => `- ${file}`))
}

if (!relativeFiles.includes('index.html')) errors.push('dist does not contain the required index.html entrypoint')
if (!relativeFiles.some(file => file !== 'index.html')) errors.push('dist does not contain any production assets besides index.html')

for (const file of relativeFiles) {
  const extension = extname(file)
  if (!['.html', '.js', '.mjs', '.css'].includes(extension)) continue
  const text = await readFile(join(distDir, file), 'utf8')
  for (const reference of extractReferences(file, text)) {
    const resolved = resolveReference(file, reference)
    if (!resolved) continue
    if (!resolved.path || !fileSet.has(resolved.path)) {
      errors.push(`${file} references missing asset ${resolved.reference}`)
    }
  }
}

if (errors.length > 0) {
  for (const error of errors) console.error(error)
  process.exitCode = 1
} else {
  console.log(`verified ${relativeFiles.length} production dist assets and local references`)
}
