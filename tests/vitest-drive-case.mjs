import { spawnSync } from 'node:child_process'
import { existsSync, realpathSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

if (process.platform !== 'win32') {
  console.log('Windows drive-case regression skipped on non-Windows')
  process.exit(0)
}

const root = realpathSync.native(fileURLToPath(new URL('..', import.meta.url)))
const frontendDir = path.join(root, 'frontend')
const launcher = path.join(frontendDir, 'scripts', 'run-vitest.mjs')
if (!existsSync(launcher)) throw new Error(`missing Vitest launcher: ${launcher}`)

const lowerDriveRoot = root.length > 1 ? root[0].toLowerCase() + root.slice(1) : root
const result = spawnSync(process.execPath, [launcher, '--version'], {
  cwd: lowerDriveRoot,
  encoding: 'utf8',
})
if (result.error) throw result.error
if (result.status !== 0) {
  throw new Error(`Vitest launcher failed from lower-case drive path (status ${result.status})\n${result.stdout}\n${result.stderr}`)
}
if (!/vitest/i.test(`${result.stdout}\n${result.stderr}`)) {
  throw new Error(`Vitest launcher did not report its version\n${result.stdout}\n${result.stderr}`)
}
console.log(`Vitest launcher passed from ${lowerDriveRoot}`)
